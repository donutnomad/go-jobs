package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jobs/scheduler/internal/loadbalance"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
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
	storage    *orm.Storage
	lbManager  *loadbalance.Manager
	logger     *zap.Logger
	httpClient *http.Client

	maxWorkers int
	taskCh     chan *taskJob
	stopCh     chan struct{}
	wg         sync.WaitGroup

	// 超时管理器，避免goroutine泄漏
	timeoutMu sync.RWMutex
	timeouts  map[string]*time.Timer

	// 熔断器管理，每个执行器一个熔断器
	breakerMu sync.RWMutex
	breakers  map[string]*CircuitBreaker
}

type taskJob struct {
	task      *models.Task
	execution *models.TaskExecution
}

// NewTaskRunner 创建任务执行器
func NewTaskRunner(
	storage *orm.Storage,
	lbManager *loadbalance.Manager,
	logger *zap.Logger,
	maxWorkers int,
) *TaskRunner {
	return &TaskRunner{
		storage:   storage,
		lbManager: lbManager,
		logger:    logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxWorkers: maxWorkers,
		taskCh:     make(chan *taskJob, maxWorkers*2),
		stopCh:     make(chan struct{}),
		timeouts:   make(map[string]*time.Timer),
		breakers:   make(map[string]*CircuitBreaker),
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

// Submit 提交任务
func (r *TaskRunner) Submit(task *models.Task, execution *models.TaskExecution) {
	select {
	case r.taskCh <- &taskJob{task: task, execution: execution}:
		r.logger.Debug("task submitted",
			zap.String("task_id", task.ID),
			zap.String("execution_id", execution.ID))
	default:
		r.logger.Warn("task queue is full, dropping task",
			zap.String("task_id", task.ID),
			zap.String("execution_id", execution.ID))

		// 更新执行状态为失败
		execution.Status = models.ExecutionStatusFailed
		execution.Logs = "Task queue is full"
		now := time.Now()
		execution.EndTime = &now
		r.storage.DB().Save(execution)
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

// executeTask 执行任务（使用循环替代递归，避免栈溢出）
func (r *TaskRunner) executeTask(task *models.Task, execution *models.TaskExecution) {
	ctx := context.Background()

	r.logger.Info("executing task",
		zap.String("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.String("execution_id", execution.ID))

	// 更新执行状态为运行中
	now := time.Now()
	execution.Status = models.ExecutionStatusRunning
	execution.StartTime = &now
	if err := r.storage.DB().Save(execution).Error; err != nil {
		r.logger.Error("failed to update execution status",
			zap.String("execution_id", execution.ID),
			zap.Error(err))
	}

	// 使用循环处理重试，避免递归调用
	var lastErr error
	maxRetries := task.MaxRetry
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
				zap.String("task_id", task.ID),
				zap.String("execution_id", execution.ID),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff))
			time.Sleep(backoff)
		}

		// 获取健康的执行器
		executors, err := models.NewExecutorRepo(r.storage.DB()).GetHealthyExecutors(ctx, task.ID)
		if err != nil || len(executors) == 0 {
			lastErr = fmt.Errorf("no healthy executors available")
			continue
		}

		// 使用负载均衡策略选择执行器
		selectedExecutor, err := r.lbManager.SelectExecutor(ctx, task, executors)
		if err != nil {
			lastErr = fmt.Errorf("failed to select executor: %w", err)
			continue
		}

		// 更新执行记录中的执行器ID
		execution.ExecutorID = &selectedExecutor.ID
		execution.RetryCount = attempt
		if err := r.storage.DB().Save(execution).Error; err != nil {
			r.logger.Error("failed to update executor id",
				zap.String("execution_id", execution.ID),
				zap.Error(err))
		}

		// 调用执行器
		err = r.callExecutor(ctx, task, execution, selectedExecutor)
		if err != nil {
			lastErr = err
			continue
		}

		// 执行成功，设置超时监控
		if task.TimeoutSeconds > 0 {
			// 使用context取消机制替代goroutine
			r.scheduleTimeout(execution.ID, time.Duration(task.TimeoutSeconds)*time.Second)
		}
		return
	}

	// 所有重试都失败
	r.failExecution(execution, fmt.Sprintf("Execution failed after %d attempts: %v", maxRetries+1, lastErr))
}

// getOrCreateBreaker 获取或创建执行器的熔断器
func (r *TaskRunner) getOrCreateBreaker(executorID string) *CircuitBreaker {
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
func (r *TaskRunner) RemoveBreaker(executorID string) {
	r.breakerMu.Lock()
	defer r.breakerMu.Unlock()
	delete(r.breakers, executorID)
	r.logger.Debug("circuit breaker removed for offline executor",
		zap.String("executor_id", executorID))
}

// ResetBreaker 重置执行器的熔断器（当执行器恢复上线时调用）
func (r *TaskRunner) ResetBreaker(executorID string) {
	r.breakerMu.Lock()
	defer r.breakerMu.Unlock()
	if breaker, exists := r.breakers[executorID]; exists {
		breaker.mu.Lock()
		breaker.state = "closed"
		breaker.failureCount = 0
		breaker.successCount = 0
		breaker.mu.Unlock()
		r.logger.Debug("circuit breaker reset for recovered executor",
			zap.String("executor_id", executorID))
	}
}

// callExecutor 调用执行器（带熔断器保护）
func (r *TaskRunner) callExecutor(ctx context.Context, task *models.Task, execution *models.TaskExecution, exec *models.Executor) error {
	// 获取该执行器的熔断器
	breaker := r.getOrCreateBreaker(exec.ID)

	// 通过熔断器调用
	return breaker.Call(func() error {
		// 构建请求
		url := fmt.Sprintf("%s/execute", exec.BaseURL)

		payload := map[string]any{
			"execution_id": execution.ID,
			"task_id":      task.ID,
			"task_name":    task.Name,
			"parameters":   task.Parameters,
			"callback_url": fmt.Sprintf("http://localhost:8080/api/v1/executions/%s/callback", execution.ID),
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
		req.Header.Set("X-Execution-ID", execution.ID)

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
			zap.String("task_id", task.ID),
			zap.String("execution_id", execution.ID),
			zap.String("executor_id", exec.ID))

		return nil
	})
}

// failExecution 标记执行失败
func (r *TaskRunner) failExecution(execution *models.TaskExecution, reason string) {
	now := time.Now()
	execution.Status = models.ExecutionStatusFailed
	execution.EndTime = &now
	execution.Logs = reason

	if err := r.storage.DB().Save(execution).Error; err != nil {
		r.logger.Error("failed to update execution status",
			zap.String("execution_id", execution.ID),
			zap.Error(err))
	}

	r.logger.Error("task execution failed",
		zap.String("execution_id", execution.ID),
		zap.String("reason", reason))
}

// scheduleTimeout 设置超时定时器（避免goroutine泄漏）
func (r *TaskRunner) scheduleTimeout(executionID string, timeout time.Duration) {
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

// cancelTimeout 取消超时定时器
func (r *TaskRunner) cancelTimeout(executionID string) {
	r.timeoutMu.Lock()
	defer r.timeoutMu.Unlock()

	if timer, exists := r.timeouts[executionID]; exists {
		timer.Stop()
		delete(r.timeouts, executionID)
	}
}

// handleTimeout 处理执行超时
func (r *TaskRunner) handleTimeout(executionID string) {
	// 清理定时器记录
	r.timeoutMu.Lock()
	delete(r.timeouts, executionID)
	r.timeoutMu.Unlock()

	// 重新加载执行记录
	var current models.TaskExecution
	if err := r.storage.DB().Where("id = ?", executionID).First(&current).Error; err != nil {
		r.logger.Error("failed to load execution",
			zap.String("execution_id", executionID),
			zap.Error(err))
		return
	}

	// 如果仍在运行，标记为超时
	if current.Status == models.ExecutionStatusRunning {
		now := time.Now()
		current.Status = models.ExecutionStatusTimeout
		current.EndTime = &now
		current.Logs = "Execution timeout"

		if err := r.storage.DB().Save(&current).Error; err != nil {
			r.logger.Error("failed to update execution status",
				zap.String("execution_id", executionID),
				zap.Error(err))
		}

		r.logger.Warn("task execution timeout",
			zap.String("execution_id", executionID))
	}
}

type ExecutionCallbackRequest struct {
	ExecutionID string                 `json:"execution_id" binding:"required"`
	Status      models.ExecutionStatus `json:"status" binding:"required"`
	Result      map[string]any         `json:"result"`
	Logs        string                 `json:"logs"`
}

// HandleCallback 处理执行回调
func (r *TaskRunner) HandleCallback(ctx context.Context, executionID string, req ExecutionCallbackRequest) error {
	// 取消超时定时器（如果存在）
	r.cancelTimeout(executionID)

	// 加载执行记录
	var execution models.TaskExecution
	if err := r.storage.DB().Where("id = ?", executionID).First(&execution).Error; err != nil {
		return fmt.Errorf("execution not found: %w", err)
	}

	// 更新执行状态
	now := time.Now()
	execution.Status = req.Status
	execution.EndTime = &now
	execution.Result = req.Result
	execution.Logs = req.Logs

	if err := r.storage.DB().Save(&execution).Error; err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	r.logger.Info("execution callback received",
		zap.String("execution_id", executionID),
		zap.String("status", string(req.Status)))

	return nil
}
