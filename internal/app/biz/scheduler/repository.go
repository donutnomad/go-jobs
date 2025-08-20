package scheduler

import (
	"context"
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// Repository 调度器仓储接口
// 遵循DDD原则，定义调度器聚合根的持久化操作
type Repository interface {
	// 基础CRUD操作
	Save(ctx context.Context, scheduler *Scheduler) error
	FindByID(ctx context.Context, id types.ID) (*Scheduler, error)
	Update(ctx context.Context, scheduler *Scheduler) error
	Delete(ctx context.Context, id types.ID) error

	// 查询操作
	FindByInstanceID(ctx context.Context, instanceID string) (*Scheduler, error)
	FindByStatus(ctx context.Context, status SchedulerStatus, pagination types.Pagination) ([]*Scheduler, error)
	FindAll(ctx context.Context, pagination types.Pagination) ([]*Scheduler, error)
	FindOnlineSchedulers(ctx context.Context) ([]*Scheduler, error)

	// 领导者选举相关
	FindCurrentLeader(ctx context.Context) (*Scheduler, error)
	FindLeaderCandidates(ctx context.Context) ([]*Scheduler, error)
	FindByLeadershipStatus(ctx context.Context, status LeadershipStatus) ([]*Scheduler, error)

	// 健康检查相关
	FindHealthySchedulers(ctx context.Context, heartbeatTimeout time.Duration) ([]*Scheduler, error)
	FindStaleSchedulers(ctx context.Context, staleTreshold time.Duration) ([]*Scheduler, error)
	UpdateHeartbeat(ctx context.Context, schedulerID types.ID, heartbeatTime time.Time) error

	// 条件查询
	FindByFilters(ctx context.Context, filters SchedulerFilters, pagination types.Pagination) ([]*Scheduler, error)
	Count(ctx context.Context, filters SchedulerFilters) (int64, error)

	// 分布式锁操作
	AcquireLeadershipLock(ctx context.Context, schedulerID types.ID, lockKey string, lockTimeout time.Duration) (bool, error)
	ReleaseLeadershipLock(ctx context.Context, lockKey string) error
	CheckLockOwnership(ctx context.Context, schedulerID types.ID, lockKey string) (bool, error)
	RefreshLock(ctx context.Context, schedulerID types.ID, lockKey string, lockTimeout time.Duration) error

	// 批量操作
	BatchUpdateStatus(ctx context.Context, schedulerIDs []types.ID, status SchedulerStatus, reason string) error
	BatchMarkStale(ctx context.Context, olderThan time.Time) (int64, error)

	// 清理操作
	CleanupOfflineSchedulers(ctx context.Context, olderThan time.Time) (int64, error)

	// 统计操作
	GetStatusCounts(ctx context.Context) (map[SchedulerStatus]int64, error)
	GetLeadershipStatusCounts(ctx context.Context) (map[LeadershipStatus]int64, error)
	GetClusterHealth(ctx context.Context) (*ClusterHealth, error)

	// 存在性检查
	ExistsByInstanceID(ctx context.Context, instanceID string) (bool, error)
	ExistsByID(ctx context.Context, id types.ID) (bool, error)
}

// QueryService 调度器查询服务接口
// 专门用于复杂查询操作，遵循CQRS原则
type QueryService interface {
	// 复杂查询
	GetSchedulerOverview(ctx context.Context, schedulerID types.ID) (*SchedulerOverview, error)
	GetClusterTopology(ctx context.Context) (*ClusterTopology, error)
	GetLeadershipHistory(ctx context.Context, days int) ([]*LeadershipRecord, error)

	// 统计查询
	GetClusterStatistics(ctx context.Context, timeRange TimeRange) (*ClusterStatistics, error)
	GetSchedulerPerformance(ctx context.Context, schedulerID types.ID, timeRange TimeRange) (*SchedulerPerformance, error)
	GetLoadDistribution(ctx context.Context) (*LoadDistribution, error)

	// 监控查询
	GetClusterStatus(ctx context.Context) (*ClusterStatus, error)
	GetSchedulingMetrics(ctx context.Context, timeRange TimeRange) (*SchedulingMetrics, error)
	GetFailoverHistory(ctx context.Context, days int) ([]*FailoverRecord, error)

	// 搜索功能
	SearchSchedulers(ctx context.Context, query string, pagination types.Pagination) ([]*Scheduler, error)
}

// SchedulerFilters 调度器过滤条件
type SchedulerFilters struct {
	InstanceID       string            `json:"instance_id,omitempty"`
	Status           *SchedulerStatus  `json:"status,omitempty"`
	LeadershipStatus *LeadershipStatus `json:"leadership_status,omitempty"`
	IsHealthy        *bool             `json:"is_healthy,omitempty"`
	HeartbeatAfter   *time.Time        `json:"heartbeat_after,omitempty"`
	HeartbeatBefore  *time.Time        `json:"heartbeat_before,omitempty"`
	CreatedAfter     *time.Time        `json:"created_after,omitempty"`
	CreatedBefore    *time.Time        `json:"created_before,omitempty"`
}

// SchedulerOverview 调度器概览
type SchedulerOverview struct {
	Scheduler           *Scheduler          `json:"scheduler"`
	ScheduledTasksCount int64               `json:"scheduled_tasks_count"`
	RunningTasksCount   int64               `json:"running_tasks_count"`
	CompletedTasksCount int64               `json:"completed_tasks_count"`
	LeadershipDuration  *time.Duration      `json:"leadership_duration,omitempty"`
	PerformanceMetrics  *PerformanceMetrics `json:"performance_metrics"`
	ClusterRole         string              `json:"cluster_role"`
}

// ClusterTopology 集群拓扑
type ClusterTopology struct {
	TotalSchedulers   int                 `json:"total_schedulers"`
	OnlineSchedulers  int                 `json:"online_schedulers"`
	Leader            *Scheduler          `json:"leader,omitempty"`
	Followers         []*Scheduler        `json:"followers"`
	MaintenanceNodes  []*Scheduler        `json:"maintenance_nodes"`
	OfflineNodes      []*Scheduler        `json:"offline_nodes"`
	NetworkPartitions []*NetworkPartition `json:"network_partitions,omitempty"`
	ClusterHealth     *ClusterHealth      `json:"cluster_health"`
}

// LeadershipRecord 领导者记录
type LeadershipRecord struct {
	SchedulerID    types.ID       `json:"scheduler_id"`
	InstanceID     string         `json:"instance_id"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        *time.Time     `json:"end_time,omitempty"`
	Duration       *time.Duration `json:"duration,omitempty"`
	Reason         string         `json:"reason,omitempty"`
	TasksScheduled int64          `json:"tasks_scheduled"`
}

// ClusterStatistics 集群统计
type ClusterStatistics struct {
	TotalSchedulers       int64                     `json:"total_schedulers"`
	StatusDistribution    map[SchedulerStatus]int64 `json:"status_distribution"`
	LeadershipChanges     int64                     `json:"leadership_changes"`
	AvgLeadershipDuration float64                   `json:"avg_leadership_duration"`
	TotalScheduledTasks   int64                     `json:"total_scheduled_tasks"`
	SuccessfulScheduling  int64                     `json:"successful_scheduling"`
	FailedScheduling      int64                     `json:"failed_scheduling"`
	SchedulingSuccessRate float64                   `json:"scheduling_success_rate"`
	ClusterUptime         float64                   `json:"cluster_uptime"`
	TimeRange             TimeRange                 `json:"time_range"`
}

// SchedulerPerformance 调度器性能
type SchedulerPerformance struct {
	SchedulerID          types.ID       `json:"scheduler_id"`
	InstanceID           string         `json:"instance_id"`
	TasksScheduled       int64          `json:"tasks_scheduled"`
	SuccessfulExecutions int64          `json:"successful_executions"`
	FailedExecutions     int64          `json:"failed_executions"`
	AvgSchedulingLatency float64        `json:"avg_scheduling_latency"`
	ThroughputPerHour    float64        `json:"throughput_per_hour"`
	ErrorRate            float64        `json:"error_rate"`
	UptimePercent        float64        `json:"uptime_percent"`
	MemoryUsage          *ResourceUsage `json:"memory_usage,omitempty"`
	CPUUsage             *ResourceUsage `json:"cpu_usage,omitempty"`
}

// LoadDistribution 负载分布
type LoadDistribution struct {
	SchedulerLoads    map[types.ID]*SchedulerLoad `json:"scheduler_loads"`
	TotalLoad         int64                       `json:"total_load"`
	BalanceScore      float64                     `json:"balance_score"` // 0-1, 1为完全平衡
	LoadVariance      float64                     `json:"load_variance"`
	RecommendedAction string                      `json:"recommended_action"`
}

// SchedulerLoad 调度器负载
type SchedulerLoad struct {
	SchedulerID    types.ID `json:"scheduler_id"`
	RunningTasks   int64    `json:"running_tasks"`
	PendingTasks   int64    `json:"pending_tasks"`
	LoadPercentage float64  `json:"load_percentage"`
	IsOverloaded   bool     `json:"is_overloaded"`
}

// ClusterStatus 集群状态
type ClusterStatus struct {
	IsHealthy         bool            `json:"is_healthy"`
	HasLeader         bool            `json:"has_leader"`
	LeaderID          *types.ID       `json:"leader_id,omitempty"`
	OnlineCount       int             `json:"online_count"`
	TotalCount        int             `json:"total_count"`
	LastLeaderChange  *time.Time      `json:"last_leader_change,omitempty"`
	NetworkPartitions int             `json:"network_partitions"`
	ClusterHealth     *ClusterHealth  `json:"cluster_health"`
	RecentAlerts      []*ClusterAlert `json:"recent_alerts"`
}

// SchedulingMetrics 调度指标
type SchedulingMetrics struct {
	TotalSchedulingEvents    int64             `json:"total_scheduling_events"`
	SuccessfulScheduling     int64             `json:"successful_scheduling"`
	FailedScheduling         int64             `json:"failed_scheduling"`
	AverageSchedulingLatency float64           `json:"average_scheduling_latency"`
	SchedulingThroughput     float64           `json:"scheduling_throughput"`
	PeakLoad                 int64             `json:"peak_load"`
	LoadDistribution         *LoadDistribution `json:"load_distribution"`
	TimeRange                TimeRange         `json:"time_range"`
}

// FailoverRecord 故障转移记录
type FailoverRecord struct {
	ID               types.ID      `json:"id"`
	OldLeaderID      types.ID      `json:"old_leader_id"`
	NewLeaderID      types.ID      `json:"new_leader_id"`
	FailoverTime     time.Time     `json:"failover_time"`
	FailoverDuration time.Duration `json:"failover_duration"`
	Reason           string        `json:"reason"`
	ImpactedTasks    int64         `json:"impacted_tasks"`
	Recovery         bool          `json:"recovery"`
}

// ClusterHealth 集群健康状态
type ClusterHealth struct {
	OverallStatus       HealthStatus   `json:"overall_status"`
	HealthScore         float64        `json:"health_score"` // 0-100
	OnlineNodes         int            `json:"online_nodes"`
	HealthyNodes        int            `json:"healthy_nodes"`
	TotalNodes          int            `json:"total_nodes"`
	HasQuorum           bool           `json:"has_quorum"`
	LeaderStability     float64        `json:"leader_stability"`
	NetworkConnectivity float64        `json:"network_connectivity"`
	Issues              []*HealthIssue `json:"issues,omitempty"`
	LastCheckTime       time.Time      `json:"last_check_time"`
}

// NetworkPartition 网络分区
type NetworkPartition struct {
	ID            string         `json:"id"`
	AffectedNodes []types.ID     `json:"affected_nodes"`
	DetectedAt    time.Time      `json:"detected_at"`
	ResolvedAt    *time.Time     `json:"resolved_at,omitempty"`
	Duration      *time.Duration `json:"duration,omitempty"`
	Impact        string         `json:"impact"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	CPU                 float64 `json:"cpu"`
	Memory              float64 `json:"memory"`
	NetworkLatency      float64 `json:"network_latency"`
	DiskIO              float64 `json:"disk_io"`
	SchedulingLatency   float64 `json:"scheduling_latency"`
	ThroughputPerSecond float64 `json:"throughput_per_second"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	Current    float64 `json:"current"`
	Average    float64 `json:"average"`
	Peak       float64 `json:"peak"`
	Limit      float64 `json:"limit"`
	Percentage float64 `json:"percentage"`
}

// ClusterAlert 集群告警
type ClusterAlert struct {
	ID          string        `json:"id"`
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Message     string        `json:"message"`
	SchedulerID *types.ID     `json:"scheduler_id,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
}

// HealthIssue 健康问题
type HealthIssue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	SchedulerID *types.ID `json:"scheduler_id,omitempty"`
	DetectedAt  time.Time `json:"detected_at"`
}

// HealthStatus 健康状态枚举
type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
	HealthStatusUnknown  HealthStatus = "unknown"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeLeadershipChange AlertType = "leadership_change"
	AlertTypeNodeDown         AlertType = "node_down"
	AlertTypeNetworkPartition AlertType = "network_partition"
	AlertTypeHighLoad         AlertType = "high_load"
	AlertTypeSchedulingFailed AlertType = "scheduling_failed"
)

// AlertSeverity 告警严重性
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}
