package scheduler

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "sync"
    "time"

	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/loadbalance"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

// CircuitBreaker 简单的熔断器实现
type CircuitBreaker struct {
    mu              sync.RWMutex
    failureCount    int
    successCount    int
    lastFailureTime time.Time
	state           string // "closed", "open", "half-open"
	threshold       int
	timeout         time.Duration
	resetTimeout    time.Duration
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:        "closed",
		threshold:    3, // 3次失败后打开
		timeout:      30 * time.Second,
		resetTimeout: 60 * time.Second, // 60秒后尝试恢复
	}
}

// Call 通过熔断器调用函数
var ErrCircuitOpen = errors.New("circuit breaker is open")

func (cb *CircuitBreaker) Call(fn func() error) error {
    // Fast path: check state under lock
    cb.mu.Lock()
    switch cb.state {
    case "open":
        if time.Since(cb.lastFailureTime) > cb.resetTimeout {
            // Move to half-open; allow a trial call
            cb.state = "half-open"
            cb.failureCount = 0
            cb.successCount = 0
        } else {
            cb.mu.Unlock()
            return ErrCircuitOpen
        }
    }
    cb.mu.Unlock()

    // Execute the protected function outside the lock
    err := fn()

    // Update breaker state based on result
    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.failureCount++
        cb.lastFailureTime = time.Now()
        if cb.state == "half-open" || cb.failureCount >= cb.threshold {
            cb.state = "open"
        }
        return err
    }

    if cb.state == "half-open" {
        cb.successCount++
        if cb.successCount >= 2 { // promote to closed after consecutive successes
            cb.state = "closed"
            cb.failureCount = 0
            cb.successCount = 0
        }
    }
    return nil
}

// TaskRunner 任务执行器
type TaskRunner struct {
	lbManager     *loadbalance.Manager
	logger        *zap.Logger
	httpClient    *http.Client
	taskRepo      task.Repo
	executionRepo execution.Repo
	executorRepo  executor.Repo

	maxWorkers int
	taskCh     chan *taskJob
	stopCh     chan struct{}
	wg         sync.WaitGroup

	// 超时管理器，避免goroutine泄漏
	timeoutMu sync.RWMutex
	timeouts  map[uint64]*time.Timer

	// 熔断器管理，每个执行器一个熔断器
	breakerMu sync.RWMutex
	breakers  map[uint64]*CircuitBreaker

	callbackURL func(id uint64) string
}

type taskJob struct {
	task      *task.Task
	execution *execution.TaskExecution
}

// submit pushes a prepared task+execution to the queue.
// Non-blocking: if the queue is full, marks execution as failed.
func (r *TaskRunner) submit(tsk *task.Task, exec *execution.TaskExecution) {
	select {
	case r.taskCh <- &taskJob{task: tsk, execution: exec}:
		r.logger.Debug("task submitted",
			zap.Uint64("task_id", tsk.ID),
			zap.Uint64("execution_id", exec.ID))
	default:
		r.logger.Warn("task queue is full, dropping task",
			zap.Uint64("task_id", tsk.ID),
			zap.Uint64("execution_id", exec.ID))
		// Use domain method to mark failed when queue is full
		ctx := context.Background()
		exec.MarkFailed("Task queue is full")
		_ = r.executionRepo.Save(ctx, exec)
	}
}

type TaskRunnerConfig struct {
	MaxWorkers  int
	CallbackURL func(id uint64) string
}

// NewTaskRunner 创建任务执行器
func NewTaskRunner(
	lbManager *loadbalance.Manager,
	logger *zap.Logger,
	cfg TaskRunnerConfig,
	taskRepo task.Repo,
	executionRepo execution.Repo,
	executorRepo executor.Repo,
) *TaskRunner {
	return &TaskRunner{
		lbManager: lbManager,
		logger:    logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxWorkers:    cfg.MaxWorkers,
		taskCh:        make(chan *taskJob, cfg.MaxWorkers*2),
		stopCh:        make(chan struct{}),
		timeouts:      make(map[uint64]*time.Timer),
		breakers:      make(map[uint64]*CircuitBreaker),
		callbackURL:   cfg.CallbackURL,
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		executorRepo:  executorRepo,
	}
}

// Start 启动任务执行器
func (r *TaskRunner) Start() {
	for i := 0; i < r.maxWorkers; i++ {
		r.wg.Add(1)
		go r.worker(i)
	}
	r.logger.Info("task runner started",
		zap.Int("workers", r.maxWorkers))
}

// Stop 停止任务执行器
func (r *TaskRunner) Stop() {
	close(r.stopCh)
	r.wg.Wait()

	// 清理所有定时器
	r.timeoutMu.Lock()
	for id, timer := range r.timeouts {
		timer.Stop()
		delete(r.timeouts, id)
	}
	r.timeoutMu.Unlock()

	r.logger.Info("task runner stopped")
}

func (r *TaskRunner) Submit(taskId uint64, parameters map[string]any, executionId uint64) {
	ctx := context.Background()
	tsk, err := r.taskRepo.GetByID(ctx, taskId)
	if err != nil {
		r.logger.Error("failed to load task",
			zap.Uint64("task_id", taskId),
			zap.Error(err))
		return
	}

	record, err := r.executionRepo.GetByID(ctx, executionId)
	if err != nil {
		r.logger.Error("failed to load task execution",
			zap.Uint64("execution_id", executionId),
			zap.Error(err))
		return
	}
	// Merge external parameters into the task entity
	tsk.MergeParameters(parameters)

	select {
	case r.taskCh <- &taskJob{task: tsk, execution: record}:
		r.logger.Debug("task submitted",
			zap.Uint64("task_id", tsk.ID),
			zap.Uint64("execution_id", record.ID))
	default:
		r.logger.Warn("task queue is full, dropping task",
			zap.Uint64("task_id", tsk.ID),
			zap.Uint64("execution_id", record.ID))

		// 使用领域方法更新为失败
		ctx := context.Background()
		record.MarkFailed("Task queue is full")
		_ = r.executionRepo.Save(ctx, record)
	}
}

// worker 工作协程
func (r *TaskRunner) worker(id int) {
	defer r.wg.Done()

	r.logger.Debug("worker started", zap.Int("worker_id", id))

	for {
		select {
		case job := <-r.taskCh:
			r.executeTask(job.task, job.execution)
		case <-r.stopCh:
			r.logger.Debug("worker stopped", zap.Int("worker_id", id))
			return
		}
	}
}

func (r *TaskRunner) executeTask(tsk *task.Task, exec *execution.TaskExecution) {
	ctx := context.Background()

	r.logger.Info("executing task",
		zap.Uint64("task_id", tsk.ID),
		zap.String("task_name", tsk.Name),
		zap.Uint64("execution_id", exec.ID))

	// 更新执行状态为运行中（封装为领域方法）
	exec.StartNow()
	if err := r.executionRepo.Save(ctx, exec); err != nil {
		r.logger.Error("failed to update execution status",
			zap.Uint64("execution_id", exec.ID),
			zap.Error(err))
	}

	// 使用循环处理重试，避免递归调用
	var lastErr error
	maxRetries := tsk.MaxRetry
	if maxRetries < 0 {
		maxRetries = 0
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避策略：1s, 2s, 4s, 8s... 最大30s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			r.logger.Info("retrying task execution",
				zap.Uint64("task_id", tsk.ID),
				zap.Uint64("execution_id", exec.ID),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff))
			time.Sleep(backoff)
		}

		// 获取健康的执行器
		executors, err := r.executorRepo.GetHealthyExecutorsByTask(ctx, tsk.ID)
		if err != nil || len(executors) == 0 {
			lastErr = fmt.Errorf("no healthy executors available")
			continue
		}

		// 使用负载均衡策略选择执行器
		selectedExecutor, err := r.lbManager.SelectExecutor(ctx, tsk, executors)
		if err != nil {
			lastErr = fmt.Errorf("failed to select executor: %w", err)
			continue
		}

		// 更新执行记录中的执行器ID（封装为领域方法）
		exec.AssignExecutor(selectedExecutor.ID, attempt)
		if err := r.executionRepo.Save(ctx, exec); err != nil {
			r.logger.Error("failed to update executor id",
				zap.Uint64("execution_id", exec.ID),
				zap.Error(err))
		}

		// 调用执行器
		err = r.callExecutor(ctx, tsk, exec, selectedExecutor)
		if err != nil {
			lastErr = err
			continue
		}

		// 执行成功，设置超时监控
		if tsk.TimeoutSeconds > 0 {
			// 使用context取消机制替代goroutine
			r.scheduleTimeout(exec.ID, time.Duration(tsk.TimeoutSeconds)*time.Second)
		}
		return
	}

	// 所有重试都失败
	r.failExecution(exec, fmt.Sprintf("Execution failed after %d attempts: %v", maxRetries+1, lastErr))
}

// getOrCreateBreaker 获取或创建执行器的熔断器
func (r *TaskRunner) getOrCreateBreaker(executorID uint64) *CircuitBreaker {
	r.breakerMu.RLock()
	breaker, exists := r.breakers[executorID]
	r.breakerMu.RUnlock()

	if !exists {
		r.breakerMu.Lock()
		breaker, exists = r.breakers[executorID]
		if !exists {
			breaker = NewCircuitBreaker()
			r.breakers[executorID] = breaker
		}
		r.breakerMu.Unlock()
	}

	return breaker
}

// RemoveBreaker 移除执行器的熔断器（当执行器下线时调用）
func (r *TaskRunner) RemoveBreaker(executorID uint64) {
	r.breakerMu.Lock()
	defer r.breakerMu.Unlock()
	delete(r.breakers, executorID)
	r.logger.Debug("circuit breaker removed for offline executor",
		zap.Uint64("executor_id", executorID))
}

// ResetBreaker 重置执行器的熔断器（当执行器恢复上线时调用）
func (r *TaskRunner) ResetBreaker(executorID uint64) {
	r.breakerMu.Lock()
	defer r.breakerMu.Unlock()
	if breaker, exists := r.breakers[executorID]; exists {
		breaker.mu.Lock()
		breaker.state = "closed"
		breaker.failureCount = 0
		breaker.successCount = 0
		breaker.mu.Unlock()
		r.logger.Debug("circuit breaker reset for recovered executor",
			zap.Uint64("executor_id", executorID))
	}
}

// callExecutor 调用执行器（带熔断器保护）
func (r *TaskRunner) callExecutor(ctx context.Context, task *task.Task, execution *execution.TaskExecution, exec *executor.Executor) error {
    // 获取该执行器的熔断器
    breaker := r.getOrCreateBreaker(exec.ID)

    // 通过熔断器调用
    return breaker.Call(func() error {
        // 构建请求
        url := exec.GetExecURL()

        // Derive a per-request timeout from task settings with sane bounds
        reqTimeout := r.requestTimeoutForTask(task)
        ctxReq, cancel := context.WithTimeout(ctx, reqTimeout)
        defer cancel()

		payload := map[string]any{
			"execution_id": cast.ToString(execution.ID),
			"task_id":      cast.ToString(task.ID),
			"task_name":    task.Name,
			"parameters":   task.Parameters,
			"callback_url": r.callbackURL(execution.ID),
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

        req, err := http.NewRequestWithContext(ctxReq, http.MethodPost, url, bytes.NewBuffer(jsonData))
        if err != nil {
            return fmt.Errorf("failed to create request: %w", err)
        }

		req.Header.Set("Content-Type", "application/json")

		// 发送请求
		resp, err := r.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to call executor: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			return fmt.Errorf("executor returned status %d", resp.StatusCode)
		}

		r.logger.Info("successfully called executor",
			zap.Uint64("task_id", task.ID),
			zap.Uint64("execution_id", execution.ID),
			zap.Uint64("executor_id", exec.ID))

		return nil
    })
}

// requestTimeoutForTask returns a reasonable HTTP request timeout derived from the task timeout.
// It ensures a minimum of 1s and caps at 30s to avoid long hangs.
func (r *TaskRunner) requestTimeoutForTask(t *task.Task) time.Duration {
    const (
        defaultTimeout = 10 * time.Second
        maxTimeout     = 30 * time.Second
        minTimeout     = 1 * time.Second
    )
    if t != nil && t.TimeoutSeconds > 0 {
        d := time.Duration(t.TimeoutSeconds) * time.Second
        if d < minTimeout {
            return minTimeout
        }
        if d > maxTimeout {
            return maxTimeout
        }
        return d
    }
    return defaultTimeout
}

// failExecution 标记执行失败
func (r *TaskRunner) failExecution(exec *execution.TaskExecution, reason string) {
	ctx := context.Background()
	exec.MarkFailed(reason)

	if err := r.executionRepo.Save(ctx, exec); err != nil {
		r.logger.Error("failed to update execution status",
			zap.Uint64("execution_id", exec.ID),
			zap.Error(err))
	}

	r.logger.Error("task execution failed",
		zap.Uint64("execution_id", exec.ID),
		zap.String("reason", reason))
}

func (r *TaskRunner) scheduleTimeout(executionID uint64, timeout time.Duration) {
	r.timeoutMu.Lock()
	defer r.timeoutMu.Unlock()

	// 如果已有定时器，先取消
	if oldTimer, exists := r.timeouts[executionID]; exists {
		oldTimer.Stop()
	}

	// 创建新的定时器
	r.timeouts[executionID] = time.AfterFunc(timeout, func() {
		r.handleTimeout(executionID)
	})
}

// CancelTimeout 取消超时定时器
func (r *TaskRunner) CancelTimeout(executionID uint64) {
	r.timeoutMu.Lock()
	defer r.timeoutMu.Unlock()

	if timer, exists := r.timeouts[executionID]; exists {
		timer.Stop()
		delete(r.timeouts, executionID)
	}
}

// handleTimeout 处理执行超时
func (r *TaskRunner) handleTimeout(executionID uint64) {
	ctx := context.Background()

	// 清理定时器记录
	r.timeoutMu.Lock()
	delete(r.timeouts, executionID)
	r.timeoutMu.Unlock()

	current, err := r.executionRepo.GetByID(ctx, executionID)
	if err != nil {
		r.logger.Error("failed to load execution",
			zap.Uint64("execution_id", executionID),
			zap.Error(err))
		return
	}
	if current.Status != execution.ExecutionStatusRunning {
		return
	}

	// 如果仍在运行，标记为超时
	current.MarkTimeout()
	if err := r.executionRepo.Save(ctx, current); err != nil {
		r.logger.Error("failed to update execution status",
			zap.Uint64("execution_id", executionID),
			zap.Error(err))
	}

	r.logger.Warn("task execution timeout",
		zap.Uint64("execution_id", executionID))
}
