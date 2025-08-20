package execution

import (
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// 任务执行领域事件定义

// TaskExecutionCreatedEvent 任务执行创建事件
type TaskExecutionCreatedEvent struct {
	ExecutionID   types.ID  `json:"execution_id"`
	TaskID        types.ID  `json:"task_id"`
	ScheduledTime time.Time `json:"scheduled_time"`
	CreatedAt     time.Time `json:"created_at"`
}

// EventType 返回事件类型
func (e TaskExecutionCreatedEvent) EventType() string {
	return "execution.created"
}

// AggregateID 返回聚合根ID
func (e TaskExecutionCreatedEvent) AggregateID() types.ID {
	return e.ExecutionID
}

// OccurredOn 返回事件发生时间
func (e TaskExecutionCreatedEvent) OccurredOn() time.Time {
	return e.CreatedAt
}

// TaskExecutionStartedEvent 任务执行开始事件
type TaskExecutionStartedEvent struct {
	ExecutionID types.ID  `json:"execution_id"`
	TaskID      types.ID  `json:"task_id"`
	ExecutorID  types.ID  `json:"executor_id"`
	StartedAt   time.Time `json:"started_at"`
}

// EventType 返回事件类型
func (e TaskExecutionStartedEvent) EventType() string {
	return "execution.started"
}

// AggregateID 返回聚合根ID
func (e TaskExecutionStartedEvent) AggregateID() types.ID {
	return e.ExecutionID
}

// OccurredOn 返回事件发生时间
func (e TaskExecutionStartedEvent) OccurredOn() time.Time {
	return e.StartedAt
}

// TaskExecutionCompletedEvent 任务执行成功完成事件
type TaskExecutionCompletedEvent struct {
	ExecutionID types.ID        `json:"execution_id"`
	TaskID      types.ID        `json:"task_id"`
	Status      ExecutionStatus `json:"status"`
	Duration    time.Duration   `json:"duration"`
	CompletedAt time.Time       `json:"completed_at"`
}

// EventType 返回事件类型
func (e TaskExecutionCompletedEvent) EventType() string {
	return "execution.completed"
}

// AggregateID 返回聚合根ID
func (e TaskExecutionCompletedEvent) AggregateID() types.ID {
	return e.ExecutionID
}

// OccurredOn 返回事件发生时间
func (e TaskExecutionCompletedEvent) OccurredOn() time.Time {
	return e.CompletedAt
}

// TaskExecutionFailedEvent 任务执行失败事件
type TaskExecutionFailedEvent struct {
	ExecutionID types.ID        `json:"execution_id"`
	TaskID      types.ID        `json:"task_id"`
	Status      ExecutionStatus `json:"status"`
	Error       string          `json:"error"`
	Retry       int             `json:"retry"`
	FailedAt    time.Time       `json:"failed_at"`
}

// EventType 返回事件类型
func (e TaskExecutionFailedEvent) EventType() string {
	return "execution.failed"
}

// AggregateID 返回聚合根ID
func (e TaskExecutionFailedEvent) AggregateID() types.ID {
	return e.ExecutionID
}

// OccurredOn 返回事件发生时间
func (e TaskExecutionFailedEvent) OccurredOn() time.Time {
	return e.FailedAt
}

// TaskExecutionCancelledEvent 任务执行取消事件
type TaskExecutionCancelledEvent struct {
	ExecutionID types.ID  `json:"execution_id"`
	TaskID      types.ID  `json:"task_id"`
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}

// EventType 返回事件类型
func (e TaskExecutionCancelledEvent) EventType() string {
	return "execution.cancelled"
}

// AggregateID 返回聚合根ID
func (e TaskExecutionCancelledEvent) AggregateID() types.ID {
	return e.ExecutionID
}

// OccurredOn 返回事件发生时间
func (e TaskExecutionCancelledEvent) OccurredOn() time.Time {
	return e.CancelledAt
}

// TaskExecutionSkippedEvent 任务执行跳过事件
type TaskExecutionSkippedEvent struct {
	ExecutionID types.ID  `json:"execution_id"`
	TaskID      types.ID  `json:"task_id"`
	Reason      string    `json:"reason"`
	SkippedAt   time.Time `json:"skipped_at"`
}

// EventType 返回事件类型
func (e TaskExecutionSkippedEvent) EventType() string {
	return "execution.skipped"
}

// AggregateID 返回聚合根ID
func (e TaskExecutionSkippedEvent) AggregateID() types.ID {
	return e.ExecutionID
}

// OccurredOn 返回事件发生时间
func (e TaskExecutionSkippedEvent) OccurredOn() time.Time {
	return e.SkippedAt
}

// TaskExecutionRetriedEvent 任务执行重试事件
type TaskExecutionRetriedEvent struct {
	ExecutionID types.ID  `json:"execution_id"`
	TaskID      types.ID  `json:"task_id"`
	RetryCount  int       `json:"retry_count"`
	RetriedAt   time.Time `json:"retried_at"`
}

// EventType 返回事件类型
func (e TaskExecutionRetriedEvent) EventType() string {
	return "execution.retried"
}

// AggregateID 返回聚合根ID
func (e TaskExecutionRetriedEvent) AggregateID() types.ID {
	return e.ExecutionID
}

// OccurredOn 返回事件发生时间
func (e TaskExecutionRetriedEvent) OccurredOn() time.Time {
	return e.RetriedAt
}
