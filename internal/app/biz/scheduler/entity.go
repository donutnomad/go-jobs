package scheduler

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/app/types"
)

// Scheduler 调度器领域实体
type Scheduler struct {
	// 基础属性
	id               types.ID
	instanceID       string
	clusterConfig    ClusterConfig
	status           SchedulerStatus
	leadershipStatus LeadershipStatus
	leadershipLock   string
	metadata         SchedulerMetadata

	// 时间戳
	createdAt       time.Time
	updatedAt       time.Time
	lastHeartbeat   time.Time
	leaderElectedAt *time.Time

	// 领域事件
	domainEvents []interface{}
}

// NewScheduler 创建新调度器（工厂方法）
func NewScheduler(instanceID string, config ClusterConfig) (*Scheduler, error) {
	// 验证参数
	if instanceID == "" {
		return nil, fmt.Errorf("instance ID cannot be empty")
	}
	if !config.IsValid() {
		return nil, fmt.Errorf("invalid cluster config")
	}

	// 生成唯一ID
	id := types.ID(uuid.New().String())

	now := time.Now()
	scheduler := &Scheduler{
		id:               id,
		instanceID:       instanceID,
		clusterConfig:    config,
		status:           SchedulerStatusOnline,
		leadershipStatus: LeadershipStatusFollower,
		leadershipLock:   "scheduler_leader_lock",
		metadata:         SchedulerMetadata{},
		createdAt:        now,
		updatedAt:        now,
		lastHeartbeat:    now,
		domainEvents:     make([]interface{}, 0),
	}

	// 添加领域事件
	scheduler.addDomainEvent(SchedulerInstanceStartedEvent{
		SchedulerID: id,
		InstanceID:  instanceID,
		StartedAt:   now,
	})

	return scheduler, nil
}

// ID 获取调度器ID
func (s *Scheduler) ID() types.ID {
	return s.id
}

// InstanceID 获取实例ID
func (s *Scheduler) InstanceID() string {
	return s.instanceID
}

// ClusterConfig 获取集群配置
func (s *Scheduler) ClusterConfig() ClusterConfig {
	return s.clusterConfig
}

// Status 获取状态
func (s *Scheduler) Status() SchedulerStatus {
	return s.status
}

// LeadershipStatus 获取领导者状态
func (s *Scheduler) LeadershipStatus() LeadershipStatus {
	return s.leadershipStatus
}

// LeadershipLock 获取领导者锁
func (s *Scheduler) LeadershipLock() string {
	return s.leadershipLock
}

// Metadata 获取元数据
func (s *Scheduler) Metadata() SchedulerMetadata {
	return s.metadata
}

// CreatedAt 获取创建时间
func (s *Scheduler) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt 获取更新时间
func (s *Scheduler) UpdatedAt() time.Time {
	return s.updatedAt
}

// LastHeartbeat 获取最后心跳时间
func (s *Scheduler) LastHeartbeat() time.Time {
	return s.lastHeartbeat
}

// LeaderElectedAt 获取领导者选举时间
func (s *Scheduler) LeaderElectedAt() *time.Time {
	return s.leaderElectedAt
}

// 业务方法

// IsOnline 判断是否在线
func (s *Scheduler) IsOnline() bool {
	return s.status == SchedulerStatusOnline
}

// IsOffline 判断是否离线
func (s *Scheduler) IsOffline() bool {
	return s.status == SchedulerStatusOffline
}

// IsInMaintenance 判断是否在维护中
func (s *Scheduler) IsInMaintenance() bool {
	return s.status == SchedulerStatusMaintenance
}

// IsLeader 判断是否为领导者
func (s *Scheduler) IsLeader() bool {
	return s.leadershipStatus == LeadershipStatusLeader
}

// IsFollower 判断是否为跟随者
func (s *Scheduler) IsFollower() bool {
	return s.leadershipStatus == LeadershipStatusFollower
}

// IsElecting 判断是否在选举中
func (s *Scheduler) IsElecting() bool {
	return s.leadershipStatus == LeadershipStatusElecting
}

// CanScheduleTasks 判断是否可以调度任务
func (s *Scheduler) CanScheduleTasks() bool {
	return s.IsOnline() && s.IsLeader()
}

// IsHealthy 判断是否健康（基于心跳）
func (s *Scheduler) IsHealthy(heartbeatTimeout time.Duration) bool {
	return time.Since(s.lastHeartbeat) < heartbeatTimeout
}

// NeedsLeaderElection 判断是否需要重新选举领导者
func (s *Scheduler) NeedsLeaderElection(leaderTimeout time.Duration) bool {
	if s.IsLeader() {
		return false
	}
	// 如果没有活跃的领导者超过一定时间，需要重新选举
	return s.leaderElectedAt == nil || time.Since(*s.leaderElectedAt) > leaderTimeout
}

// GetLockTimeout 获取锁超时时间
func (s *Scheduler) GetLockTimeout() time.Duration {
	return s.clusterConfig.GetLockTimeout()
}

// GetHeartbeatInterval 获取心跳间隔
func (s *Scheduler) GetHeartbeatInterval() time.Duration {
	return s.clusterConfig.GetHeartbeatInterval()
}

// ShouldTryBecomeLeader 判断是否应该尝试成为领导者
func (s *Scheduler) ShouldTryBecomeLeader() bool {
	return s.IsOnline() && s.IsFollower()
}

// 状态变更方法

// GoOnline 上线
func (s *Scheduler) GoOnline() error {
	if s.status == SchedulerStatusOnline {
		return nil // 已经在线
	}

	previousStatus := s.status
	s.status = SchedulerStatusOnline
	s.updatedAt = time.Now()
	s.lastHeartbeat = s.updatedAt

	// 添加领域事件
	s.addDomainEvent(SchedulerStatusChangedEvent{
		SchedulerID:    s.id,
		InstanceID:     s.instanceID,
		PreviousStatus: previousStatus,
		CurrentStatus:  s.status,
		ChangedAt:      s.updatedAt,
	})

	return nil
}

// GoOffline 离线
func (s *Scheduler) GoOffline(reason string) error {
	if s.status == SchedulerStatusOffline {
		return nil // 已经离线
	}

	previousStatus := s.status
	s.status = SchedulerStatusOffline
	s.updatedAt = time.Now()

	// 如果是领导者，需要释放领导权
	if s.IsLeader() {
		s.resignLeadership("scheduler going offline")
	}

	// 添加领域事件
	s.addDomainEvent(SchedulerStatusChangedEvent{
		SchedulerID:    s.id,
		InstanceID:     s.instanceID,
		PreviousStatus: previousStatus,
		CurrentStatus:  s.status,
		ChangedAt:      s.updatedAt,
		Reason:         reason,
	})

	return nil
}

// EnterMaintenance 进入维护模式
func (s *Scheduler) EnterMaintenance(reason string) error {
	if s.status == SchedulerStatusMaintenance {
		return nil // 已经在维护中
	}

	previousStatus := s.status
	s.status = SchedulerStatusMaintenance
	s.updatedAt = time.Now()

	// 如果是领导者，需要释放领导权
	if s.IsLeader() {
		s.resignLeadership("scheduler entering maintenance")
	}

	// 添加领域事件
	s.addDomainEvent(SchedulerStatusChangedEvent{
		SchedulerID:    s.id,
		InstanceID:     s.instanceID,
		PreviousStatus: previousStatus,
		CurrentStatus:  s.status,
		ChangedAt:      s.updatedAt,
		Reason:         reason,
	})

	return nil
}

// 领导者选举方法

// StartElection 开始选举
func (s *Scheduler) StartElection() error {
	if !s.IsOnline() {
		return fmt.Errorf("scheduler is not online")
	}
	if s.IsLeader() {
		return fmt.Errorf("scheduler is already a leader")
	}

	previousStatus := s.leadershipStatus
	s.leadershipStatus = LeadershipStatusElecting
	s.updatedAt = time.Now()

	// 添加领域事件
	s.addDomainEvent(LeaderElectionStartedEvent{
		SchedulerID: s.id,
		InstanceID:  s.instanceID,
		StartedAt:   s.updatedAt,
	})

	return nil
}

// BecomeLeader 成为领导者
func (s *Scheduler) BecomeLeader() error {
	if !s.IsOnline() {
		return fmt.Errorf("scheduler is not online")
	}
	if s.IsLeader() {
		return nil // 已经是领导者
	}

	previousStatus := s.leadershipStatus
	s.leadershipStatus = LeadershipStatusLeader
	s.updatedAt = time.Now()
	electedAt := s.updatedAt
	s.leaderElectedAt = &electedAt

	// 添加领域事件
	s.addDomainEvent(LeaderElectedEvent{
		SchedulerID: s.id,
		InstanceID:  s.instanceID,
		ElectedAt:   s.updatedAt,
	})

	return nil
}

// BecomeFollower 成为跟随者
func (s *Scheduler) BecomeFollower(reason string) error {
	if s.IsFollower() {
		return nil // 已经是跟随者
	}

	previousStatus := s.leadershipStatus
	s.leadershipStatus = LeadershipStatusFollower
	s.updatedAt = time.Now()

	// 如果之前是领导者，清除领导者选举时间
	if previousStatus == LeadershipStatusLeader {
		s.leaderElectedAt = nil
	}

	// 添加领域事件
	s.addDomainEvent(LeadershipLostEvent{
		SchedulerID: s.id,
		InstanceID:  s.instanceID,
		Reason:      reason,
		LostAt:      s.updatedAt,
	})

	return nil
}

// resignLeadership 辞去领导权（内部方法）
func (s *Scheduler) resignLeadership(reason string) {
	if s.IsLeader() {
		s.BecomeFollower(reason)
	}
}

// UpdateHeartbeat 更新心跳
func (s *Scheduler) UpdateHeartbeat() error {
	if !s.IsOnline() {
		return fmt.Errorf("scheduler is not online")
	}

	s.lastHeartbeat = time.Now()
	s.updatedAt = s.lastHeartbeat

	return nil
}

// 更新方法

// UpdateClusterConfig 更新集群配置
func (s *Scheduler) UpdateClusterConfig(config ClusterConfig) error {
	if !config.IsValid() {
		return fmt.Errorf("invalid cluster config")
	}

	if config == s.clusterConfig {
		return nil // 没有变化
	}

	oldConfig := s.clusterConfig
	s.clusterConfig = config
	s.updatedAt = time.Now()

	// 添加领域事件
	s.addDomainEvent(SchedulerConfigUpdatedEvent{
		SchedulerID: s.id,
		InstanceID:  s.instanceID,
		OldConfig:   oldConfig,
		NewConfig:   config,
		UpdatedAt:   s.updatedAt,
	})

	return nil
}

// UpdateMetadata 更新元数据
func (s *Scheduler) UpdateMetadata(metadata SchedulerMetadata) error {
	oldMetadata := s.metadata
	s.metadata = metadata
	s.updatedAt = time.Now()

	// 添加领域事件
	s.addDomainEvent(SchedulerMetadataUpdatedEvent{
		SchedulerID: s.id,
		InstanceID:  s.instanceID,
		OldMetadata: oldMetadata,
		NewMetadata: metadata,
		UpdatedAt:   s.updatedAt,
	})

	return nil
}

// 领域事件处理

// GetDomainEvents 获取领域事件
func (s *Scheduler) GetDomainEvents() []interface{} {
	events := make([]interface{}, len(s.domainEvents))
	copy(events, s.domainEvents)
	return events
}

// ClearDomainEvents 清除领域事件
func (s *Scheduler) ClearDomainEvents() {
	s.domainEvents = make([]interface{}, 0)
}

// addDomainEvent 添加领域事件
func (s *Scheduler) addDomainEvent(event interface{}) {
	s.domainEvents = append(s.domainEvents, event)
}

// AddDomainEvent 添加领域事件（公共方法，供UseCase使用）
func (s *Scheduler) AddDomainEvent(event interface{}) {
	s.addDomainEvent(event)
}

// 验证方法

// Validate 验证调度器的完整性
func (s *Scheduler) Validate() error {
	if s.id.IsZero() {
		return fmt.Errorf("scheduler ID cannot be empty")
	}
	if s.instanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}
	if !s.clusterConfig.IsValid() {
		return fmt.Errorf("invalid cluster config")
	}
	if s.leadershipLock == "" {
		return fmt.Errorf("leadership lock cannot be empty")
	}
	return nil
}
