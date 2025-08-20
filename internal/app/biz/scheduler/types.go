package scheduler

import (
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// SchedulerStatus 调度器状态
type SchedulerStatus int

const (
	SchedulerStatusActive SchedulerStatus = iota + 1
	SchedulerStatusInactive
	SchedulerStatusMaintenance
)

func (s SchedulerStatus) String() string {
	switch s {
	case SchedulerStatusActive:
		return "active"
	case SchedulerStatusInactive:
		return "inactive"
	case SchedulerStatusMaintenance:
		return "maintenance"
	default:
		return "unknown"
	}
}

// IsActive 判断是否为活跃状态
func (s SchedulerStatus) IsActive() bool {
	return s == SchedulerStatusActive
}

// CanScheduleTasks 判断是否可以调度任务
func (s SchedulerStatus) CanScheduleTasks() bool {
	return s == SchedulerStatusActive
}

// LeadershipStatus 领导权状态
type LeadershipStatus int

const (
	LeadershipStatusFollower LeadershipStatus = iota + 1
	LeadershipStatusLeader
	LeadershipStatusCandidate
)

func (s LeadershipStatus) String() string {
	switch s {
	case LeadershipStatusFollower:
		return "follower"
	case LeadershipStatusLeader:
		return "leader"
	case LeadershipStatusCandidate:
		return "candidate"
	default:
		return "unknown"
	}
}

// IsLeader 判断是否为领导者
func (s LeadershipStatus) IsLeader() bool {
	return s == LeadershipStatusLeader
}

// CanBecomeLeader 判断是否可以成为领导者
func (s LeadershipStatus) CanBecomeLeader() bool {
	return s == LeadershipStatusFollower || s == LeadershipStatusCandidate
}

// ClusterConfig 集群配置值对象
type ClusterConfig struct {
	InstanceID        string        `json:"instance_id"`
	Host              string        `json:"host"`
	Port              int           `json:"port"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	ElectionTimeout   time.Duration `json:"election_timeout"`
}

// NewClusterConfig 创建集群配置
func NewClusterConfig(instanceID, host string, port int) (ClusterConfig, error) {
	if instanceID == "" {
		return ClusterConfig{}, fmt.Errorf("instance ID is required")
	}
	if host == "" {
		return ClusterConfig{}, fmt.Errorf("host is required")
	}
	if port <= 0 || port > 65535 {
		return ClusterConfig{}, fmt.Errorf("invalid port: %d", port)
	}

	return ClusterConfig{
		InstanceID:        instanceID,
		Host:              host,
		Port:              port,
		HeartbeatInterval: 10 * time.Second,
		ElectionTimeout:   30 * time.Second,
	}, nil
}

// IsValid 验证配置是否有效
func (c ClusterConfig) IsValid() bool {
	return c.InstanceID != "" && c.Host != "" &&
		c.Port > 0 && c.Port <= 65535 &&
		c.HeartbeatInterval > 0 && c.ElectionTimeout > 0
}

// GetAddress 获取完整地址
func (c ClusterConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ShouldSendHeartbeat 判断是否应该发送心跳
func (c ClusterConfig) ShouldSendHeartbeat(lastHeartbeat time.Time) bool {
	return time.Since(lastHeartbeat) >= c.HeartbeatInterval
}

// IsElectionTimeout 判断是否选举超时
func (c ClusterConfig) IsElectionTimeout(lastContact time.Time) bool {
	return time.Since(lastContact) >= c.ElectionTimeout
}

// LockConfiguration 锁配置值对象
type LockConfiguration struct {
	Key     string        `json:"key"`
	Timeout time.Duration `json:"timeout"`
	TTL     time.Duration `json:"ttl"`
}

// NewLockConfiguration 创建锁配置
func NewLockConfiguration(key string) LockConfiguration {
	if key == "" {
		key = "scheduler_leader_lock"
	}
	return LockConfiguration{
		Key:     key,
		Timeout: 5 * time.Second,
		TTL:     30 * time.Second,
	}
}

// IsValid 验证锁配置是否有效
func (c LockConfiguration) IsValid() bool {
	return c.Key != "" && c.Timeout > 0 && c.TTL > 0
}

// ShouldRenew 判断是否应该续约锁
func (c LockConfiguration) ShouldRenew(lastRenewal time.Time) bool {
	// 在TTL过期前续约（提前一些时间）
	renewTime := c.TTL / 2
	return time.Since(lastRenewal) >= renewTime
}

// SchedulerInstance 调度器实例信息
type SchedulerInstance struct {
	InstanceID       string           `json:"instance_id"`
	Host             string           `json:"host"`
	Port             int              `json:"port"`
	Status           SchedulerStatus  `json:"status"`
	LeadershipStatus LeadershipStatus `json:"leadership_status"`
	StartTime        time.Time        `json:"start_time"`
	LastHeartbeat    time.Time        `json:"last_heartbeat"`
}

// NewSchedulerInstance 创建调度器实例
func NewSchedulerInstance(instanceID, host string, port int) SchedulerInstance {
	now := time.Now()
	return SchedulerInstance{
		InstanceID:       instanceID,
		Host:             host,
		Port:             port,
		Status:           SchedulerStatusActive,
		LeadershipStatus: LeadershipStatusFollower,
		StartTime:        now,
		LastHeartbeat:    now,
	}
}

// GetAddress 获取地址
func (s SchedulerInstance) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// IsHealthy 判断是否健康
func (s SchedulerInstance) IsHealthy(timeout time.Duration) bool {
	return time.Since(s.LastHeartbeat) <= timeout
}

// BecomeLeader 成为领导者
func (s *SchedulerInstance) BecomeLeader() {
	s.LeadershipStatus = LeadershipStatusLeader
	s.UpdateHeartbeat()
}

// BecomeFollower 成为跟随者
func (s *SchedulerInstance) BecomeFollower() {
	s.LeadershipStatus = LeadershipStatusFollower
	s.UpdateHeartbeat()
}

// UpdateHeartbeat 更新心跳
func (s *SchedulerInstance) UpdateHeartbeat() {
	s.LastHeartbeat = time.Now()
}

// GetUptime 获取运行时间
func (s SchedulerInstance) GetUptime() time.Duration {
	return time.Since(s.StartTime)
}

// ClusterInfo 集群信息
type ClusterInfo struct {
	TotalInstances  int                 `json:"total_instances"`
	ActiveInstances int                 `json:"active_instances"`
	Leader          *SchedulerInstance  `json:"leader"`
	Instances       []SchedulerInstance `json:"instances"`
	HasLeader       bool                `json:"has_leader"`
}

// NewClusterInfo 创建集群信息
func NewClusterInfo(instances []SchedulerInstance) ClusterInfo {
	info := ClusterInfo{
		TotalInstances: len(instances),
		Instances:      instances,
	}

	// 统计活跃实例和查找领导者
	for _, instance := range instances {
		if instance.Status.IsActive() {
			info.ActiveInstances++
		}
		if instance.LeadershipStatus.IsLeader() {
			info.Leader = &instance
			info.HasLeader = true
		}
	}

	return info
}

// IsHealthy 判断集群是否健康
func (c ClusterInfo) IsHealthy() bool {
	return c.HasLeader && c.ActiveInstances > 0
}

// NeedsElection 判断是否需要选举
func (c ClusterInfo) NeedsElection() bool {
	return !c.HasLeader && c.ActiveInstances > 0
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	TotalTasks        int `json:"total_tasks"`
	ActiveTasks       int `json:"active_tasks"`
	PausedTasks       int `json:"paused_tasks"`
	TotalExecutors    int `json:"total_executors"`
	OnlineExecutors   int `json:"online_executors"`
	HealthyExecutors  int `json:"healthy_executors"`
	RunningExecutions int `json:"running_executions"`
	PendingExecutions int `json:"pending_executions"`
}

// GetTaskActiveRate 获取任务活跃率
func (m SystemMetrics) GetTaskActiveRate() float64 {
	if m.TotalTasks == 0 {
		return 0
	}
	return float64(m.ActiveTasks) / float64(m.TotalTasks) * 100
}

// GetExecutorHealthRate 获取执行器健康率
func (m SystemMetrics) GetExecutorHealthRate() float64 {
	if m.TotalExecutors == 0 {
		return 0
	}
	return float64(m.HealthyExecutors) / float64(m.TotalExecutors) * 100
}

// IsSystemHealthy 判断系统是否健康
func (m SystemMetrics) IsSystemHealthy() bool {
	// 健康标准：至少有一个在线执行器，执行器健康率超过50%
	return m.OnlineExecutors > 0 && m.GetExecutorHealthRate() >= 50
}

// StartSchedulerRequest 启动调度器请求
type StartSchedulerRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required"`
}

// Validate 验证请求
func (r StartSchedulerRequest) Validate() error {
	if r.InstanceID == "" {
		return fmt.Errorf("instance ID is required")
	}
	if r.Host == "" {
		return fmt.Errorf("host is required")
	}
	if r.Port <= 0 || r.Port > 65535 {
		return fmt.Errorf("invalid port: %d", r.Port)
	}
	return nil
}

// SchedulerFilter 调度器过滤器
type SchedulerFilter struct {
	types.Filter
	Status         []string `json:"status"`
	Leadership     []string `json:"leadership"`
	IncludeMetrics bool     `json:"include_metrics"`
}

// NewSchedulerFilter 创建调度器过滤器
func NewSchedulerFilter() SchedulerFilter {
	return SchedulerFilter{
		Filter: types.NewFilter(),
	}
}
