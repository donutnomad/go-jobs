package execution

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/app/types"
)

// TaskExecution 任务执行领域实体
type TaskExecution struct {
	// 基础属性
	id           types.ID
	taskID       types.ID
	context      ExecutionContext
	status       ExecutionStatus
	result       ExecutionResult
	retryPolicy  RetryPolicy
	currentRetry int

	// 时间戳
	createdAt time.Time
	updatedAt time.Time

	// 领域事件
	domainEvents []interface{}
}

// NewTaskExecution 创建新的任务执行（工厂方法）
func NewTaskExecution(taskID types.ID, parameters types.JSONMap, scheduledTime time.Time) (*TaskExecution, error) {
	if taskID.IsZero() {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	// 生成执行ID
	id := types.ID(uuid.New().String())

	// 创建执行上下文
	context := NewExecutionContext(taskID, parameters, scheduledTime)

	now := time.Now()
	execution := &TaskExecution{
		id:           id,
		taskID:       taskID,
		context:      context,
		status:       ExecutionStatusPending,
		result:       NewExecutionResult(ExecutionStatusPending),
		retryPolicy:  NewRetryPolicy(3), // 默认重试3次
		currentRetry: 0,
		createdAt:    now,
		updatedAt:    now,
		domainEvents: make([]interface{}, 0),
	}

	// 添加领域事件
	execution.addDomainEvent(TaskExecutionCreatedEvent{
		ExecutionID:   id,
		TaskID:        taskID,
		ScheduledTime: scheduledTime,
		CreatedAt:     now,
	})

	return execution, nil
}

// ID 获取执行ID
func (e *TaskExecution) ID() types.ID {
	return e.id
}

// TaskID 获取任务ID
func (e *TaskExecution) TaskID() types.ID {
	return e.taskID
}

// Context 获取执行上下文
func (e *TaskExecution) Context() ExecutionContext {
	return e.context
}

// Status 获取执行状态
func (e *TaskExecution) Status() ExecutionStatus {
	return e.status
}

// Result 获取执行结果
func (e *TaskExecution) Result() ExecutionResult {
	return e.result
}

// RetryPolicy 获取重试策略
func (e *TaskExecution) RetryPolicy() RetryPolicy {
	return e.retryPolicy
}

// CurrentRetry 获取当前重试次数
func (e *TaskExecution) CurrentRetry() int {
	return e.currentRetry
}

// CreatedAt 获取创建时间
func (e *TaskExecution) CreatedAt() time.Time {
	return e.createdAt
}

// UpdatedAt 获取更新时间
func (e *TaskExecution) UpdatedAt() time.Time {
	return e.updatedAt
}

// 业务方法

// IsPending 判断是否待执行
func (e *TaskExecution) IsPending() bool {
	return e.status == ExecutionStatusPending
}

// IsRunning 判断是否正在运行
func (e *TaskExecution) IsRunning() bool {
	return e.status.IsRunning()
}

// IsCompleted 判断是否已完成（包括成功和失败）
func (e *TaskExecution) IsCompleted() bool {
	return e.status.IsTerminal()
}

// IsSuccessful 判断是否执行成功
func (e *TaskExecution) IsSuccessful() bool {
	return e.status.IsSuccessful()
}

// IsFailed 判断是否执行失败
func (e *TaskExecution) IsFailed() bool {
	return e.status.IsFailed()
}

// CanBeRetried 判断是否可以重试
func (e *TaskExecution) CanBeRetried() bool {
	return e.status.CanBeRetried() && e.retryPolicy.ShouldRetry(e.currentRetry)
}

// CanBeStopped 判断是否可以被停止
func (e *TaskExecution) CanBeStopped() bool {
	return e.status.CanBeStopped()
}

// HasExecutor 判断是否有执行器
func (e *TaskExecution) HasExecutor() bool {
	return e.context.HasExecutor()
}

// GetExecutorID 获取执行器ID
func (e *TaskExecution) GetExecutorID() types.ID {
	return e.context.GetExecutorID()
}

// IsScheduled 判断是否已调度
func (e *TaskExecution) IsScheduled() bool {
	return e.context.IsScheduled()
}

// GetScheduledTime 获取调度时间
func (e *TaskExecution) GetScheduledTime() time.Time {
	return e.context.ScheduledTime
}

// GetDuration 获取执行时长
func (e *TaskExecution) GetDuration() time.Duration {
	if e.result.Duration != nil {
		return *e.result.Duration
	}
	return 0
}

// 状态变更方法

// Start 开始执行
func (e *TaskExecution) Start(executorID types.ID) error {
	if e.status != ExecutionStatusPending {
		return fmt.Errorf("execution is not in pending status: %s", e.status)
	}

	// 更新上下文
	e.context = e.context.WithExecutor(executorID)

	// 更新状态
	e.status = ExecutionStatusRunning
	e.updatedAt = time.Now()

	// 更新结果
	e.result = NewExecutionResult(ExecutionStatusRunning)
	startTime := time.Now()
	e.result = e.result.WithTiming(&startTime, nil)

	// 添加领域事件
	e.addDomainEvent(TaskExecutionStartedEvent{
		ExecutionID: e.id,
		TaskID:      e.taskID,
		ExecutorID:  executorID,
		StartedAt:   e.updatedAt,
	})

	return nil
}

// Complete 完成执行（成功或失败）
func (e *TaskExecution) Complete(status ExecutionStatus, result types.JSONMap, logs, errorMsg string) error {
	if !e.status.IsRunning() {
		return fmt.Errorf("execution is not running: %s", e.status)
	}

	if !status.IsTerminal() {
		return fmt.Errorf("invalid terminal status: %s", status)
	}

	// 更新状态
	e.status = status
	e.updatedAt = time.Now()

	// 更新结果
	endTime := time.Now()
	e.result = e.result.WithResult(result).WithLogs(logs)
	if errorMsg != "" {
		e.result = e.result.WithError(errorMsg)
	}

	// 更新时间信息
	if e.result.StartTime != nil {
		e.result = e.result.WithTiming(e.result.StartTime, &endTime)
	}

	// 添加领域事件
	if status.IsSuccessful() {
		e.addDomainEvent(TaskExecutionCompletedEvent{
			ExecutionID: e.id,
			TaskID:      e.taskID,
			Status:      status,
			Duration:    e.GetDuration(),
			CompletedAt: e.updatedAt,
		})
	} else {
		e.addDomainEvent(TaskExecutionFailedEvent{
			ExecutionID: e.id,
			TaskID:      e.taskID,
			Status:      status,
			Error:       errorMsg,
			Retry:       e.currentRetry,
			FailedAt:    e.updatedAt,
		})
	}

	return nil
}

// Cancel 取消执行
func (e *TaskExecution) Cancel(reason string) error {
	if !e.status.CanBeStopped() {
		return fmt.Errorf("execution cannot be stopped: %s", e.status)
	}

	e.status = ExecutionStatusCancelled
	e.updatedAt = time.Now()

	// 更新结果
	e.result = e.result.WithError(reason)
	if e.result.StartTime != nil {
		endTime := time.Now()
		e.result = e.result.WithTiming(e.result.StartTime, &endTime)
	}

	// 添加领域事件
	e.addDomainEvent(TaskExecutionCancelledEvent{
		ExecutionID: e.id,
		TaskID:      e.taskID,
		Reason:      reason,
		CancelledAt: e.updatedAt,
	})

	return nil
}

// Skip 跳过执行
func (e *TaskExecution) Skip(reason string) error {
	if e.status != ExecutionStatusPending {
		return fmt.Errorf("execution is not in pending status: %s", e.status)
	}

	e.status = ExecutionStatusSkipped
	e.updatedAt = time.Now()

	// 更新结果
	e.result = NewExecutionResult(ExecutionStatusSkipped).WithLogs(reason)

	// 添加领域事件
	e.addDomainEvent(TaskExecutionSkippedEvent{
		ExecutionID: e.id,
		TaskID:      e.taskID,
		Reason:      reason,
		SkippedAt:   e.updatedAt,
	})

	return nil
}

// Timeout 标记为超时
func (e *TaskExecution) Timeout() error {
	if !e.status.IsRunning() {
		return fmt.Errorf("execution is not running: %s", e.status)
	}

	e.status = ExecutionStatusTimeout
	e.updatedAt = time.Now()

	// 更新结果
	endTime := time.Now()
	e.result = e.result.WithError("execution timeout")
	if e.result.StartTime != nil {
		e.result = e.result.WithTiming(e.result.StartTime, &endTime)
	}

	// 添加领域事件
	e.addDomainEvent(TaskExecutionFailedEvent{
		ExecutionID: e.id,
		TaskID:      e.taskID,
		Status:      ExecutionStatusTimeout,
		Error:       "execution timeout",
		Retry:       e.currentRetry,
		FailedAt:    e.updatedAt,
	})

	return nil
}

// Retry 重试执行
func (e *TaskExecution) Retry() error {
	if !e.CanBeRetried() {
		return fmt.Errorf("execution cannot be retried")
	}

	// 增加重试次数
	e.currentRetry++

	// 重置状态为待执行
	e.status = ExecutionStatusPending
	e.updatedAt = time.Now()

	// 重置结果
	e.result = NewExecutionResult(ExecutionStatusPending)

	// 添加领域事件
	e.addDomainEvent(TaskExecutionRetriedEvent{
		ExecutionID: e.id,
		TaskID:      e.taskID,
		RetryCount:  e.currentRetry,
		RetriedAt:   e.updatedAt,
	})

	return nil
}

// 更新方法

// UpdateRetryPolicy 更新重试策略
func (e *TaskExecution) UpdateRetryPolicy(policy RetryPolicy) error {
	e.retryPolicy = policy
	e.updatedAt = time.Now()
	return nil
}

// UpdateCallbackURL 更新回调URL
func (e *TaskExecution) UpdateCallbackURL(callbackURL string) error {
	e.context = e.context.WithCallbackURL(callbackURL)
	e.updatedAt = time.Now()
	return nil
}

// UpdateParameters 更新执行参数
func (e *TaskExecution) UpdateParameters(parameters types.JSONMap) error {
	if e.status != ExecutionStatusPending {
		return fmt.Errorf("cannot update parameters of running or completed execution")
	}

	// 创建新的上下文
	newContext := NewExecutionContext(e.taskID, parameters, e.context.ScheduledTime)
	if e.context.HasExecutor() {
		newContext = newContext.WithExecutor(e.context.GetExecutorID())
	}
	if e.context.CallbackURL != "" {
		newContext = newContext.WithCallbackURL(e.context.CallbackURL)
	}

	e.context = newContext
	e.updatedAt = time.Now()

	return nil
}

// 领域事件处理

// GetDomainEvents 获取领域事件
func (e *TaskExecution) GetDomainEvents() []interface{} {
	events := make([]interface{}, len(e.domainEvents))
	copy(events, e.domainEvents)
	return events
}

// ClearDomainEvents 清除领域事件
func (e *TaskExecution) ClearDomainEvents() {
	e.domainEvents = make([]interface{}, 0)
}

// addDomainEvent 添加领域事件
func (e *TaskExecution) addDomainEvent(event interface{}) {
	e.domainEvents = append(e.domainEvents, event)
}

// 验证方法

// Validate 验证任务执行的完整性
func (e *TaskExecution) Validate() error {
	if e.id.IsZero() {
		return fmt.Errorf("execution ID cannot be empty")
	}
	if e.taskID.IsZero() {
		return fmt.Errorf("task ID cannot be empty")
	}
	if e.currentRetry < 0 {
		return fmt.Errorf("retry count cannot be negative")
	}
	if e.currentRetry > e.retryPolicy.MaxRetries {
		return fmt.Errorf("retry count exceeds max retries")
	}
	return nil
}
