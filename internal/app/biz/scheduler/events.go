package scheduler

import (
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// 调度器领域事件定义

// SchedulerInstanceStartedEvent 调度器实例启动事件
type SchedulerInstanceStartedEvent struct {
	SchedulerID types.ID  `json:"scheduler_id"`
	InstanceID  string    `json:"instance_id"`
	StartedAt   time.Time `json:"started_at"`
}

// EventType 返回事件类型
func (e SchedulerInstanceStartedEvent) EventType() string {
	return "scheduler.instance_started"
}

// AggregateID 返回聚合根ID
func (e SchedulerInstanceStartedEvent) AggregateID() types.ID {
	return e.SchedulerID
}

// OccurredOn 返回事件发生时间
func (e SchedulerInstanceStartedEvent) OccurredOn() time.Time {
	return e.StartedAt
}

// SchedulerStatusChangedEvent 调度器状态变化事件
type SchedulerStatusChangedEvent struct {
	SchedulerID    types.ID        `json:"scheduler_id"`
	InstanceID     string          `json:"instance_id"`
	PreviousStatus SchedulerStatus `json:"previous_status"`
	CurrentStatus  SchedulerStatus `json:"current_status"`
	Reason         string          `json:"reason,omitempty"`
	ChangedAt      time.Time       `json:"changed_at"`
}

// EventType 返回事件类型
func (e SchedulerStatusChangedEvent) EventType() string {
	return "scheduler.status_changed"
}

// AggregateID 返回聚合根ID
func (e SchedulerStatusChangedEvent) AggregateID() types.ID {
	return e.SchedulerID
}

// OccurredOn 返回事件发生时间
func (e SchedulerStatusChangedEvent) OccurredOn() time.Time {
	return e.ChangedAt
}

// LeaderElectionStartedEvent 领导者选举开始事件
type LeaderElectionStartedEvent struct {
	SchedulerID types.ID  `json:"scheduler_id"`
	InstanceID  string    `json:"instance_id"`
	StartedAt   time.Time `json:"started_at"`
}

// EventType 返回事件类型
func (e LeaderElectionStartedEvent) EventType() string {
	return "scheduler.election_started"
}

// AggregateID 返回聚合根ID
func (e LeaderElectionStartedEvent) AggregateID() types.ID {
	return e.SchedulerID
}

// OccurredOn 返回事件发生时间
func (e LeaderElectionStartedEvent) OccurredOn() time.Time {
	return e.StartedAt
}

// LeaderElectedEvent 领导者选举成功事件
type LeaderElectedEvent struct {
	SchedulerID types.ID  `json:"scheduler_id"`
	InstanceID  string    `json:"instance_id"`
	ElectedAt   time.Time `json:"elected_at"`
}

// EventType 返回事件类型
func (e LeaderElectedEvent) EventType() string {
	return "scheduler.leader_elected"
}

// AggregateID 返回聚合根ID
func (e LeaderElectedEvent) AggregateID() types.ID {
	return e.SchedulerID
}

// OccurredOn 返回事件发生时间
func (e LeaderElectedEvent) OccurredOn() time.Time {
	return e.ElectedAt
}

// LeadershipLostEvent 失去领导权事件
type LeadershipLostEvent struct {
	SchedulerID types.ID  `json:"scheduler_id"`
	InstanceID  string    `json:"instance_id"`
	Reason      string    `json:"reason"`
	LostAt      time.Time `json:"lost_at"`
}

// EventType 返回事件类型
func (e LeadershipLostEvent) EventType() string {
	return "scheduler.leadership_lost"
}

// AggregateID 返回聚合根ID
func (e LeadershipLostEvent) AggregateID() types.ID {
	return e.SchedulerID
}

// OccurredOn 返回事件发生时间
func (e LeadershipLostEvent) OccurredOn() time.Time {
	return e.LostAt
}

// SchedulerConfigUpdatedEvent 调度器配置更新事件
type SchedulerConfigUpdatedEvent struct {
	SchedulerID types.ID      `json:"scheduler_id"`
	InstanceID  string        `json:"instance_id"`
	OldConfig   ClusterConfig `json:"old_config"`
	NewConfig   ClusterConfig `json:"new_config"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// EventType 返回事件类型
func (e SchedulerConfigUpdatedEvent) EventType() string {
	return "scheduler.config_updated"
}

// AggregateID 返回聚合根ID
func (e SchedulerConfigUpdatedEvent) AggregateID() types.ID {
	return e.SchedulerID
}

// OccurredOn 返回事件发生时间
func (e SchedulerConfigUpdatedEvent) OccurredOn() time.Time {
	return e.UpdatedAt
}

// SchedulerMetadataUpdatedEvent 调度器元数据更新事件
type SchedulerMetadataUpdatedEvent struct {
	SchedulerID types.ID          `json:"scheduler_id"`
	InstanceID  string            `json:"instance_id"`
	OldMetadata SchedulerMetadata `json:"old_metadata"`
	NewMetadata SchedulerMetadata `json:"new_metadata"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// EventType 返回事件类型
func (e SchedulerMetadataUpdatedEvent) EventType() string {
	return "scheduler.metadata_updated"
}

// AggregateID 返回聚合根ID
func (e SchedulerMetadataUpdatedEvent) AggregateID() types.ID {
	return e.SchedulerID
}

// OccurredOn 返回事件发生时间
func (e SchedulerMetadataUpdatedEvent) OccurredOn() time.Time {
	return e.UpdatedAt
}

// SchedulerInstanceStoppedEvent 调度器实例停止事件
type SchedulerInstanceStoppedEvent struct {
	SchedulerID types.ID  `json:"scheduler_id"`
	InstanceID  string    `json:"instance_id"`
	Reason      string    `json:"reason,omitempty"`
	StoppedAt   time.Time `json:"stopped_at"`
}

// EventType 返回事件类型
func (e SchedulerInstanceStoppedEvent) EventType() string {
	return "scheduler.instance_stopped"
}

// AggregateID 返回聚合根ID
func (e SchedulerInstanceStoppedEvent) AggregateID() types.ID {
	return e.SchedulerID
}

// OccurredOn 返回事件发生时间
func (e SchedulerInstanceStoppedEvent) OccurredOn() time.Time {
	return e.StoppedAt
}
