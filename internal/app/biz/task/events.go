package task

import (
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// 领域事件定义
// 遵循DDD事件驱动架构，记录重要的业务状态变化

// TaskCreatedEvent 任务创建事件
type TaskCreatedEvent struct {
	TaskID    types.ID  `json:"task_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// EventType 返回事件类型
func (e TaskCreatedEvent) EventType() string {
	return "task.created"
}

// AggregateID 返回聚合根ID
func (e TaskCreatedEvent) AggregateID() types.ID {
	return e.TaskID
}

// OccurredOn 返回事件发生时间
func (e TaskCreatedEvent) OccurredOn() time.Time {
	return e.CreatedAt
}

// TaskUpdatedEvent 任务更新事件
type TaskUpdatedEvent struct {
	TaskID    types.ID    `json:"task_id"`
	Field     string      `json:"field"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// EventType 返回事件类型
func (e TaskUpdatedEvent) EventType() string {
	return "task.updated"
}

// AggregateID 返回聚合根ID
func (e TaskUpdatedEvent) AggregateID() types.ID {
	return e.TaskID
}

// OccurredOn 返回事件发生时间
func (e TaskUpdatedEvent) OccurredOn() time.Time {
	return e.UpdatedAt
}

// TaskStatusChangedEvent 任务状态变化事件
type TaskStatusChangedEvent struct {
	TaskID         types.ID   `json:"task_id"`
	PreviousStatus TaskStatus `json:"previous_status"`
	CurrentStatus  TaskStatus `json:"current_status"`
	ChangedAt      time.Time  `json:"changed_at"`
}

// EventType 返回事件类型
func (e TaskStatusChangedEvent) EventType() string {
	return "task.status_changed"
}

// AggregateID 返回聚合根ID
func (e TaskStatusChangedEvent) AggregateID() types.ID {
	return e.TaskID
}

// OccurredOn 返回事件发生时间
func (e TaskStatusChangedEvent) OccurredOn() time.Time {
	return e.ChangedAt
}

// TaskDeletedEvent 任务删除事件
type TaskDeletedEvent struct {
	TaskID    types.ID  `json:"task_id"`
	Name      string    `json:"name"`
	DeletedAt time.Time `json:"deleted_at"`
}

// EventType 返回事件类型
func (e TaskDeletedEvent) EventType() string {
	return "task.deleted"
}

// AggregateID 返回聚合根ID
func (e TaskDeletedEvent) AggregateID() types.ID {
	return e.TaskID
}

// OccurredOn 返回事件发生时间
func (e TaskDeletedEvent) OccurredOn() time.Time {
	return e.DeletedAt
}

// TaskScheduledEvent 任务被调度事件
type TaskScheduledEvent struct {
	TaskID        types.ID  `json:"task_id"`
	ExecutionID   types.ID  `json:"execution_id"`
	ScheduledTime time.Time `json:"scheduled_time"`
	ExecutorID    *types.ID `json:"executor_id,omitempty"`
}

// EventType 返回事件类型
func (e TaskScheduledEvent) EventType() string {
	return "task.scheduled"
}

// AggregateID 返回聚合根ID
func (e TaskScheduledEvent) AggregateID() types.ID {
	return e.TaskID
}

// OccurredOn 返回事件发生时间
func (e TaskScheduledEvent) OccurredOn() time.Time {
	return e.ScheduledTime
}

// DomainEvent 领域事件接口
type DomainEvent interface {
	EventType() string
	AggregateID() types.ID
	OccurredOn() time.Time
}
