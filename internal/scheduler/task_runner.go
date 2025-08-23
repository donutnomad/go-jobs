package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/loadbalance"
	"github.com/jobs/scheduler/internal/orm"
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
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case "open":
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = "half-open"
			cb.failureCount = 0
			cb.successCount = 0
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()
	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()
		if cb.failureCount >= cb.threshold {
			cb.state = "open"
		}
		return err
	}

	if cb.state == "half-open" {
		cb.successCount++
		if cb.successCount >= 2 {
			cb.state = "closed"
			cb.failureCount = 0
		}
	}
	return nil
}

// TaskRunner 任务执行器
type TaskRunner struct {
	storage       *orm.Storage
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

// NewTaskRunner 创建任务执行器
func NewTaskRunner(
	storage *orm.Storage,
	lbManager *loadbalance.Manager,
	logger *zap.Logger,
	maxWorkers int,
	callbackURL func(id uint64) string,
	taskRepo task.Repo,
	executionRepo execution.Repo,
	executorRepo executor.Repo,
) *TaskRunner {
	return &TaskRunner{
		storage:   storage,
		lbManager: lbManager,
		logger:    logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxWorkers:    maxWorkers,
		taskCh:        make(chan *taskJob, maxWorkers*2),
		stopCh:        make(chan struct{}),
		timeouts:      make(map[uint64]*time.Timer),
		breakers:      make(map[uint64]*CircuitBreaker),
		callbackURL:   callbackURL,
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

func (r *TaskRunner) submit(task *task.Task, record *execution.TaskExecution) {
	select {
	case r.taskCh <- &taskJob{task: task, execution: record}:
		r.logger.Debug("task submitted",
			zap.Uint64("task_id", task.ID),
			zap.Uint64("execution_id", record.ID))
	default:
		r.logger.Warn("task queue is full, dropping task",
			zap.Uint64("task_id", task.ID),
			zap.Uint64("execution_id", record.ID))

		// 更新执行状态为失败
		ctx := context.Background()
		record.Status = execution.ExecutionStatusFailed
		record.Logs = "Task queue is full"
		now := time.Now()
		record.EndTime = &now
		r.executionRepo.Save(ctx, record)
	}
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

	exec, err := r.executionRepo.GetByID(ctx, executionId)
	if err != nil {
		r.logger.Error("failed to load task execution",
			zap.Uint64("execution_id", executionId),
			zap.Error(err))
		return
	}
	if tsk.Parameters == nil {
		tsk.Parameters = make(map[string]any)
	}
	for k, v := range parameters {
		tsk.Parameters[k] = v
	}

	r.submit(tsk, exec)
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

// executeTask 执行任务（使用循环替代递归，避免栈溢出）
func (r *TaskRunner) executeTask(tsk *task.Task, exec *execution.TaskExecution) {
	ctx := context.Background()

	r.logger.Info("executing task",
		zap.Uint64("task_id", tsk.ID),
		zap.String("task_name", tsk.Name),
		zap.Uint64("execution_id", exec.ID))

	// 更新执行状态为运行中
	now := time.Now()
	exec.Status = execution.ExecutionStatusRunning
	exec.StartTime = &now
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
		executors, err := r.executorRepo.GetHealthyExecutorsForTask(ctx, tsk.ID)
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

		// 更新执行记录中的执行器ID
		exec.ExecutorID = selectedExecutor.ID
		exec.RetryCount = attempt
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

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		//req.Header.Set("X-Execution-ID", cast.ToString(execution.ID))

		// 发送请求
		resp, err := r.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to call executor: %w", err)
		}
		defer resp.Body.Close()

		fmt.Println(url)
		fmt.Println("结果是:", resp.StatusCode)
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

// failExecution 标记执行失败
func (r *TaskRunner) failExecution(exec *execution.TaskExecution, reason string) {
	ctx := context.Background()
	now := time.Now()
	exec.Status = execution.ExecutionStatusFailed
	exec.EndTime = &now
	exec.Logs = reason

	if err := r.executionRepo.Save(ctx, exec); err != nil {
		r.logger.Error("failed to update execution status",
			zap.Uint64("execution_id", exec.ID),
			zap.Error(err))
	}

	r.logger.Error("task execution failed",
		zap.Uint64("execution_id", exec.ID),
		zap.String("reason", reason))
}

// scheduleTimeout 设置超时定时器（避免goroutine泄漏）
func (r *TaskRunner) scheduleTimeout(executionID uint64, timeout time.Duration) {
	r.timeoutMu.Lock()
	defer r.timeoutMu.Unlock()

	// 如果已有定时器，先取消
	if oldTimer, exists := r.timeouts[executionID]; exists {
		oldTimer.Stop()
	}

	// 创建新的定时器
	timer := time.AfterFunc(timeout, func() {
		r.handleTimeout(executionID)
	})

	r.timeouts[executionID] = timer
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

	// 如果仍在运行，标记为超时
	if current.Status == execution.ExecutionStatusRunning {
		now := time.Now()
		current.Status = execution.ExecutionStatusTimeout
		current.EndTime = &now
		current.Logs = "Execution timeout"

		if err := r.executionRepo.Save(ctx, current); err != nil {
			r.logger.Error("failed to update execution status",
				zap.Uint64("execution_id", executionID),
				zap.Error(err))
		}

		r.logger.Warn("task execution timeout",
			zap.Uint64("execution_id", executionID))
	}
}
