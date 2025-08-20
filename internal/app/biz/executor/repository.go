package executor

import (
	"context"
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// Repository 执行器仓储接口
// 遵循DDD原则，定义执行器聚合根的持久化操作
type Repository interface {
	// 基础CRUD操作
	Save(ctx context.Context, executor *Executor) error
	FindByID(ctx context.Context, id types.ID) (*Executor, error)
	Update(ctx context.Context, executor *Executor) error
	Delete(ctx context.Context, id types.ID) error

	// 查询操作
	FindByInstanceID(ctx context.Context, instanceID string) (*Executor, error)
	FindByName(ctx context.Context, name string) ([]*Executor, error)
	FindByStatus(ctx context.Context, status ExecutorStatus, pagination types.Pagination) ([]*Executor, error)
	FindAll(ctx context.Context, pagination types.Pagination) ([]*Executor, error)
	FindAvailableExecutors(ctx context.Context) ([]*Executor, error)
	FindHealthyExecutors(ctx context.Context) ([]*Executor, error)

	// 条件查询
	FindByFilters(ctx context.Context, filters ExecutorFilters, pagination types.Pagination) ([]*Executor, error)
	Count(ctx context.Context, filters ExecutorFilters) (int64, error)

	// 执行器选择相关
	FindExecutorsForTask(ctx context.Context, taskID types.ID) ([]*Executor, error)
	FindExecutorsByTags(ctx context.Context, tags []string) ([]*Executor, error)
	FindLeastLoadedExecutors(ctx context.Context, limit int) ([]*Executor, error)

	// 健康检查相关
	FindExecutorsNeedingHealthCheck(ctx context.Context, interval time.Duration) ([]*Executor, error)
	UpdateHealthStatus(ctx context.Context, executorID types.ID, isHealthy bool, checkTime time.Time) error

	// 批量操作
	BatchUpdateStatus(ctx context.Context, executorIDs []types.ID, status ExecutorStatus, reason string) error
	BatchMarkUnhealthy(ctx context.Context, executorIDs []types.ID) error

	// 统计操作
	GetStatusCounts(ctx context.Context) (map[ExecutorStatus]int64, error)
	GetHealthStatusCounts(ctx context.Context) (map[bool]int64, error)

	// 存在性检查
	ExistsByInstanceID(ctx context.Context, instanceID string) (bool, error)
	ExistsByID(ctx context.Context, id types.ID) (bool, error)
}

// QueryService 执行器查询服务接口
// 专门用于复杂查询操作，遵循CQRS原则
type QueryService interface {
	// 复杂查询
	GetExecutorOverview(ctx context.Context, executorID types.ID) (*ExecutorOverview, error)
	GetExecutorsWithTasks(ctx context.Context, filters ExecutorFilters, pagination types.Pagination) ([]*ExecutorWithTasks, error)
	GetExecutorHealthHistory(ctx context.Context, executorID types.ID, days int) ([]*HealthRecord, error)

	// 统计查询
	GetExecutorStatistics(ctx context.Context, timeRange TimeRange) (*ExecutorStatistics, error)
	GetTaskDistribution(ctx context.Context, executorID types.ID) (*TaskDistribution, error)
	GetLoadBalancingStats(ctx context.Context) (*LoadBalancingStats, error)

	// 搜索功能
	SearchExecutors(ctx context.Context, query string, pagination types.Pagination) ([]*Executor, error)
}

// ExecutorFilters 执行器过滤条件
type ExecutorFilters struct {
	Name          string          `json:"name,omitempty"`
	InstanceID    string          `json:"instance_id,omitempty"`
	Status        *ExecutorStatus `json:"status,omitempty"`
	IsHealthy     *bool           `json:"is_healthy,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
	CreatedAfter  *time.Time      `json:"created_after,omitempty"`
	CreatedBefore *time.Time      `json:"created_before,omitempty"`
}

// ExecutorOverview 执行器概览
type ExecutorOverview struct {
	Executor          *Executor    `json:"executor"`
	TaskCount         int          `json:"task_count"`
	RunningTasks      int          `json:"running_tasks"`
	CompletedTasks    int64        `json:"completed_tasks"`
	FailedTasks       int64        `json:"failed_tasks"`
	LastExecutionTime *time.Time   `json:"last_execution_time,omitempty"`
	HealthStats       *HealthStats `json:"health_stats"`
	LoadStats         *LoadStats   `json:"load_stats"`
}

// ExecutorWithTasks 执行器及其任务
type ExecutorWithTasks struct {
	Executor *Executor `json:"executor"`
	Tasks    []*Task   `json:"tasks"`
}

// HealthRecord 健康检查记录
type HealthRecord struct {
	ExecutorID   types.ID  `json:"executor_id"`
	CheckTime    time.Time `json:"check_time"`
	IsHealthy    bool      `json:"is_healthy"`
	ResponseTime int64     `json:"response_time"` // 毫秒
	ErrorMsg     string    `json:"error_msg,omitempty"`
}

// ExecutorStatistics 执行器统计
type ExecutorStatistics struct {
	TotalExecutors       int64                    `json:"total_executors"`
	OnlineExecutors      int64                    `json:"online_executors"`
	OfflineExecutors     int64                    `json:"offline_executors"`
	MaintenanceExecutors int64                    `json:"maintenance_executors"`
	HealthyExecutors     int64                    `json:"healthy_executors"`
	UnhealthyExecutors   int64                    `json:"unhealthy_executors"`
	StatusDistribution   map[ExecutorStatus]int64 `json:"status_distribution"`
	AvgResponseTime      float64                  `json:"avg_response_time"`
	TimeRange            TimeRange                `json:"time_range"`
}

// TaskDistribution 任务分布
type TaskDistribution struct {
	ExecutorID    types.ID       `json:"executor_id"`
	TotalTasks    int            `json:"total_tasks"`
	ActiveTasks   int            `json:"active_tasks"`
	TasksByStatus map[string]int `json:"tasks_by_status"`
}

// LoadBalancingStats 负载均衡统计
type LoadBalancingStats struct {
	TotalExecutions   int64              `json:"total_executions"`
	ExecutorLoad      map[types.ID]int64 `json:"executor_load"`
	StrategyStats     map[string]int64   `json:"strategy_stats"`
	LoadDistribution  *LoadDistribution  `json:"load_distribution"`
	BalanceEfficiency float64            `json:"balance_efficiency"`
}

// HealthStats 健康统计
type HealthStats struct {
	TotalChecks      int64      `json:"total_checks"`
	SuccessfulChecks int64      `json:"successful_checks"`
	FailedChecks     int64      `json:"failed_checks"`
	SuccessRate      float64    `json:"success_rate"`
	AvgResponseTime  float64    `json:"avg_response_time"`
	LastCheckTime    *time.Time `json:"last_check_time,omitempty"`
	UptimePercent    float64    `json:"uptime_percent"`
}

// LoadStats 负载统计
type LoadStats struct {
	CurrentLoad       int     `json:"current_load"`
	MaxCapacity       int     `json:"max_capacity"`
	LoadPercent       float64 `json:"load_percent"`
	QueuedTasks       int     `json:"queued_tasks"`
	ProcessingTasks   int     `json:"processing_tasks"`
	AvgProcessingTime float64 `json:"avg_processing_time"`
}

// LoadDistribution 负载分布
type LoadDistribution struct {
	Mean         float64 `json:"mean"`
	Median       float64 `json:"median"`
	StdDeviation float64 `json:"std_deviation"`
	MinLoad      int64   `json:"min_load"`
	MaxLoad      int64   `json:"max_load"`
	Imbalance    float64 `json:"imbalance"` // 负载不平衡度
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// Task 任务信息（查询服务使用）
type Task struct {
	ID                  types.ID `json:"id"`
	Name                string   `json:"name"`
	Status              string   `json:"status"`
	CronExpression      string   `json:"cron_expression"`
	ExecutionMode       string   `json:"execution_mode"`
	LoadBalanceStrategy string   `json:"load_balance_strategy"`
}
