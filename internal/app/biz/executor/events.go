package executor

import (
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// 执行器领域事件定义

// ExecutorRegisteredEvent 执行器注册事件
type ExecutorRegisteredEvent struct {
	ExecutorID types.ID  `json:"executor_id"`
	Name       string    `json:"name"`
	InstanceID string    `json:"instance_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// EventType 返回事件类型
func (e ExecutorRegisteredEvent) EventType() string {
	return "executor.registered"
}

// AggregateID 返回聚合根ID
func (e ExecutorRegisteredEvent) AggregateID() types.ID {
	return e.ExecutorID
}

// OccurredOn 返回事件发生时间
func (e ExecutorRegisteredEvent) OccurredOn() time.Time {
	return e.CreatedAt
}

// ExecutorStatusChangedEvent 执行器状态变化事件
type ExecutorStatusChangedEvent struct {
	ExecutorID     types.ID       `json:"executor_id"`
	PreviousStatus ExecutorStatus `json:"previous_status"`
	CurrentStatus  ExecutorStatus `json:"current_status"`
	Reason         string         `json:"reason,omitempty"`
	ChangedAt      time.Time      `json:"changed_at"`
}

// EventType 返回事件类型
func (e ExecutorStatusChangedEvent) EventType() string {
	return "executor.status_changed"
}

// AggregateID 返回聚合根ID
func (e ExecutorStatusChangedEvent) AggregateID() types.ID {
	return e.ExecutorID
}

// OccurredOn 返回事件发生时间
func (e ExecutorStatusChangedEvent) OccurredOn() time.Time {
	return e.ChangedAt
}

// ExecutorHealthDegradedEvent 执行器健康状态恶化事件
type ExecutorHealthDegradedEvent struct {
	ExecutorID types.ID  `json:"executor_id"`
	Failures   int       `json:"failures"`
	DegradedAt time.Time `json:"degraded_at"`
}

// EventType 返回事件类型
func (e ExecutorHealthDegradedEvent) EventType() string {
	return "executor.health_degraded"
}

// AggregateID 返回聚合根ID
func (e ExecutorHealthDegradedEvent) AggregateID() types.ID {
	return e.ExecutorID
}

// OccurredOn 返回事件发生时间
func (e ExecutorHealthDegradedEvent) OccurredOn() time.Time {
	return e.DegradedAt
}

// ExecutorHealthRecoveredEvent 执行器健康恢复事件
type ExecutorHealthRecoveredEvent struct {
	ExecutorID  types.ID  `json:"executor_id"`
	RecoveredAt time.Time `json:"recovered_at"`
}

// EventType 返回事件类型
func (e ExecutorHealthRecoveredEvent) EventType() string {
	return "executor.health_recovered"
}

// AggregateID 返回聚合根ID
func (e ExecutorHealthRecoveredEvent) AggregateID() types.ID {
	return e.ExecutorID
}

// OccurredOn 返回事件发生时间
func (e ExecutorHealthRecoveredEvent) OccurredOn() time.Time {
	return e.RecoveredAt
}

// ExecutorUpdatedEvent 执行器更新事件
type ExecutorUpdatedEvent struct {
	ExecutorID types.ID    `json:"executor_id"`
	Field      string      `json:"field"`
	OldValue   interface{} `json:"old_value"`
	NewValue   interface{} `json:"new_value"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// EventType 返回事件类型
func (e ExecutorUpdatedEvent) EventType() string {
	return "executor.updated"
}

// AggregateID 返回聚合根ID
func (e ExecutorUpdatedEvent) AggregateID() types.ID {
	return e.ExecutorID
}

// OccurredOn 返回事件发生时间
func (e ExecutorUpdatedEvent) OccurredOn() time.Time {
	return e.UpdatedAt
}

// ExecutorUnregisteredEvent 执行器注销事件
type ExecutorUnregisteredEvent struct {
	ExecutorID     types.ID  `json:"executor_id"`
	Name           string    `json:"name"`
	InstanceID     string    `json:"instance_id"`
	UnregisteredAt time.Time `json:"unregistered_at"`
	Reason         string    `json:"reason,omitempty"`
}

// EventType 返回事件类型
func (e ExecutorUnregisteredEvent) EventType() string {
	return "executor.unregistered"
}

// AggregateID 返回聚合根ID
func (e ExecutorUnregisteredEvent) AggregateID() types.ID {
	return e.ExecutorID
}

// OccurredOn 返回事件发生时间
func (e ExecutorUnregisteredEvent) OccurredOn() time.Time {
	return e.UnregisteredAt
}
