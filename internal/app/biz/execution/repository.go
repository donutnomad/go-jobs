package execution

import (
	"context"
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// Repository 任务执行仓储接口
// 遵循DDD原则，定义任务执行聚合根的持久化操作
type Repository interface {
	// 基础CRUD操作
	Save(ctx context.Context, execution *TaskExecution) error
	FindByID(ctx context.Context, id types.ID) (*TaskExecution, error)
	Update(ctx context.Context, execution *TaskExecution) error
	Delete(ctx context.Context, id types.ID) error

	// 查询操作
	FindByTaskID(ctx context.Context, taskID types.ID, pagination types.Pagination) ([]*TaskExecution, error)
	FindByExecutorID(ctx context.Context, executorID types.ID, pagination types.Pagination) ([]*TaskExecution, error)
	FindByStatus(ctx context.Context, status ExecutionStatus, pagination types.Pagination) ([]*TaskExecution, error)
	FindAll(ctx context.Context, pagination types.Pagination) ([]*TaskExecution, error)

	// 状态查询
	FindRunningExecutions(ctx context.Context) ([]*TaskExecution, error)
	FindPendingExecutions(ctx context.Context) ([]*TaskExecution, error)
	FindTimeoutExecutions(ctx context.Context, timeoutDuration time.Duration) ([]*TaskExecution, error)
	FindRetryableExecutions(ctx context.Context) ([]*TaskExecution, error)

	// 条件查询
	FindByFilters(ctx context.Context, filters ExecutionFilters, pagination types.Pagination) ([]*TaskExecution, error)
	Count(ctx context.Context, filters ExecutionFilters) (int64, error)

	// 时间范围查询
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time, pagination types.Pagination) ([]*TaskExecution, error)
	FindLatestExecutions(ctx context.Context, limit int) ([]*TaskExecution, error)
	FindExecutionHistory(ctx context.Context, taskID types.ID, days int) ([]*TaskExecution, error)

	// 特定业务查询
	FindConcurrentExecutions(ctx context.Context, taskID types.ID) ([]*TaskExecution, error)
	FindLastExecutionByTask(ctx context.Context, taskID types.ID) (*TaskExecution, error)
	FindExecutionsByCallbackStatus(ctx context.Context, needsCallback bool) ([]*TaskExecution, error)

	// 批量操作
	BatchUpdateStatus(ctx context.Context, executionIDs []types.ID, status ExecutionStatus, reason string) error
	BatchCancel(ctx context.Context, executionIDs []types.ID, reason string) error
	BatchRetry(ctx context.Context, executionIDs []types.ID) error

	// 清理操作
	CleanupOldExecutions(ctx context.Context, olderThan time.Time) (int64, error)
	CleanupCompletedExecutions(ctx context.Context, keepDays int) (int64, error)

	// 统计操作
	GetStatusCounts(ctx context.Context) (map[ExecutionStatus]int64, error)
	GetExecutionStatsByTask(ctx context.Context, taskID types.ID) (*TaskExecutionStats, error)
	GetExecutionStatsByExecutor(ctx context.Context, executorID types.ID) (*ExecutorExecutionStats, error)

	// 存在性检查
	ExistsByID(ctx context.Context, id types.ID) (bool, error)
	HasRunningExecutions(ctx context.Context, taskID types.ID) (bool, error)
}

// QueryService 任务执行查询服务接口
// 专门用于复杂查询操作，遵循CQRS原则
type QueryService interface {
	// 复杂查询
	GetExecutionOverview(ctx context.Context, executionID types.ID) (*ExecutionOverview, error)
	GetExecutionTree(ctx context.Context, taskID types.ID, timeRange TimeRange) ([]*ExecutionNode, error)
	GetExecutionTrend(ctx context.Context, taskID types.ID, days int) (*ExecutionTrend, error)

	// 统计查询
	GetExecutionStatistics(ctx context.Context, timeRange TimeRange, filters ExecutionFilters) (*ExecutionStatistics, error)
	GetPerformanceMetrics(ctx context.Context, taskID types.ID, timeRange TimeRange) (*PerformanceMetrics, error)
	GetRetryAnalysis(ctx context.Context, taskID types.ID, timeRange TimeRange) (*RetryAnalysis, error)

	// 监控查询
	GetRunningExecutionSummary(ctx context.Context) (*RunningExecutionSummary, error)
	GetFailureAnalysis(ctx context.Context, timeRange TimeRange) (*FailureAnalysis, error)
	GetExecutorPerformanceComparison(ctx context.Context, timeRange TimeRange) ([]*ExecutorPerformance, error)

	// 搜索功能
	SearchExecutions(ctx context.Context, query string, pagination types.Pagination) ([]*TaskExecution, error)
}

// ExecutionFilters 执行过滤条件
type ExecutionFilters struct {
	TaskID          *types.ID        `json:"task_id,omitempty"`
	ExecutorID      *types.ID        `json:"executor_id,omitempty"`
	Status          *ExecutionStatus `json:"status,omitempty"`
	ScheduledAfter  *time.Time       `json:"scheduled_after,omitempty"`
	ScheduledBefore *time.Time       `json:"scheduled_before,omitempty"`
	StartedAfter    *time.Time       `json:"started_after,omitempty"`
	StartedBefore   *time.Time       `json:"started_before,omitempty"`
	CompletedAfter  *time.Time       `json:"completed_after,omitempty"`
	CompletedBefore *time.Time       `json:"completed_before,omitempty"`
	MinDuration     *time.Duration   `json:"min_duration,omitempty"`
	MaxDuration     *time.Duration   `json:"max_duration,omitempty"`
	HasError        *bool            `json:"has_error,omitempty"`
	RetryCount      *int             `json:"retry_count,omitempty"`
}

// ExecutionOverview 执行概览
type ExecutionOverview struct {
	Execution    *TaskExecution   `json:"execution"`
	TaskInfo     *TaskInfo        `json:"task_info"`
	ExecutorInfo *ExecutorInfo    `json:"executor_info"`
	RetryHistory []*TaskExecution `json:"retry_history"`
	RelatedRuns  []*TaskExecution `json:"related_runs"`
}

// ExecutionNode 执行节点（用于树形展示）
type ExecutionNode struct {
	Execution *TaskExecution   `json:"execution"`
	Children  []*ExecutionNode `json:"children,omitempty"`
	Level     int              `json:"level"`
}

// ExecutionTrend 执行趋势
type ExecutionTrend struct {
	TaskID     types.ID          `json:"task_id"`
	TimeRange  TimeRange         `json:"time_range"`
	DataPoints []*TrendDataPoint `json:"data_points"`
	Summary    *TrendSummary     `json:"summary"`
}

// TrendDataPoint 趋势数据点
type TrendDataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Executions  int64     `json:"executions"`
	Successes   int64     `json:"successes"`
	Failures    int64     `json:"failures"`
	AvgDuration float64   `json:"avg_duration"`
	SuccessRate float64   `json:"success_rate"`
}

// TrendSummary 趋势摘要
type TrendSummary struct {
	TotalExecutions    int64   `json:"total_executions"`
	OverallSuccessRate float64 `json:"overall_success_rate"`
	AvgDuration        float64 `json:"avg_duration"`
	Trend              string  `json:"trend"` // "improving", "stable", "degrading"
}

// ExecutionStatistics 执行统计
type ExecutionStatistics struct {
	TotalExecutions    int64                     `json:"total_executions"`
	StatusDistribution map[ExecutionStatus]int64 `json:"status_distribution"`
	SuccessRate        float64                   `json:"success_rate"`
	AvgDuration        float64                   `json:"avg_duration"`
	MedianDuration     float64                   `json:"median_duration"`
	MinDuration        float64                   `json:"min_duration"`
	MaxDuration        float64                   `json:"max_duration"`
	TotalRetries       int64                     `json:"total_retries"`
	AvgRetries         float64                   `json:"avg_retries"`
	TimeRange          TimeRange                 `json:"time_range"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	TaskID               types.ID         `json:"task_id"`
	ExecutionCount       int64            `json:"execution_count"`
	SuccessRate          float64          `json:"success_rate"`
	AvgResponseTime      float64          `json:"avg_response_time"`
	P50Duration          float64          `json:"p50_duration"`
	P95Duration          float64          `json:"p95_duration"`
	P99Duration          float64          `json:"p99_duration"`
	ErrorRate            float64          `json:"error_rate"`
	TimeoutRate          float64          `json:"timeout_rate"`
	ThroughputPerHour    float64          `json:"throughput_per_hour"`
	DurationDistribution map[string]int64 `json:"duration_distribution"`
}

// RetryAnalysis 重试分析
type RetryAnalysis struct {
	TaskID             types.ID      `json:"task_id"`
	TotalExecutions    int64         `json:"total_executions"`
	RetriedExecutions  int64         `json:"retried_executions"`
	RetryRate          float64       `json:"retry_rate"`
	RetryDistribution  map[int]int64 `json:"retry_distribution"`
	SuccessAfterRetry  int64         `json:"success_after_retry"`
	RetryEffectiveness float64       `json:"retry_effectiveness"`
	TopFailureReasons  []string      `json:"top_failure_reasons"`
}

// RunningExecutionSummary 运行中执行摘要
type RunningExecutionSummary struct {
	TotalRunning         int64                   `json:"total_running"`
	TaskDistribution     map[types.ID]int64      `json:"task_distribution"`
	ExecutorDistribution map[types.ID]int64      `json:"executor_distribution"`
	LongRunningTasks     []*LongRunningExecution `json:"long_running_tasks"`
	AvgRuntime           float64                 `json:"avg_runtime"`
}

// LongRunningExecution 长时间运行的执行
type LongRunningExecution struct {
	ExecutionID      types.ID       `json:"execution_id"`
	TaskID           types.ID       `json:"task_id"`
	TaskName         string         `json:"task_name"`
	ExecutorID       types.ID       `json:"executor_id"`
	StartTime        time.Time      `json:"start_time"`
	Duration         time.Duration  `json:"duration"`
	ExpectedDuration *time.Duration `json:"expected_duration,omitempty"`
}

// FailureAnalysis 失败分析
type FailureAnalysis struct {
	TotalFailures    int64              `json:"total_failures"`
	FailureRate      float64            `json:"failure_rate"`
	TaskFailures     map[types.ID]int64 `json:"task_failures"`
	ExecutorFailures map[types.ID]int64 `json:"executor_failures"`
	ErrorMessages    map[string]int64   `json:"error_messages"`
	FailurePatterns  []*FailurePattern  `json:"failure_patterns"`
	TimeRange        TimeRange          `json:"time_range"`
}

// FailurePattern 失败模式
type FailurePattern struct {
	Pattern       string     `json:"pattern"`
	Count         int64      `json:"count"`
	Percentage    float64    `json:"percentage"`
	FirstSeen     time.Time  `json:"first_seen"`
	LastSeen      time.Time  `json:"last_seen"`
	AffectedTasks []types.ID `json:"affected_tasks"`
}

// ExecutorPerformance 执行器性能
type ExecutorPerformance struct {
	ExecutorID      types.ID `json:"executor_id"`
	ExecutorName    string   `json:"executor_name"`
	TotalExecutions int64    `json:"total_executions"`
	SuccessRate     float64  `json:"success_rate"`
	AvgDuration     float64  `json:"avg_duration"`
	ErrorRate       float64  `json:"error_rate"`
	Reliability     float64  `json:"reliability"`
	Throughput      float64  `json:"throughput"`
}

// TaskExecutionStats 任务执行统计
type TaskExecutionStats struct {
	TaskID          types.ID `json:"task_id"`
	TotalExecutions int64    `json:"total_executions"`
	SuccessfulRuns  int64    `json:"successful_runs"`
	FailedRuns      int64    `json:"failed_runs"`
	SkippedRuns     int64    `json:"skipped_runs"`
	SuccessRate     float64  `json:"success_rate"`
	AvgDuration     float64  `json:"avg_duration"`
	TotalRetries    int64    `json:"total_retries"`
}

// ExecutorExecutionStats 执行器执行统计
type ExecutorExecutionStats struct {
	ExecutorID      types.ID `json:"executor_id"`
	TotalExecutions int64    `json:"total_executions"`
	SuccessfulRuns  int64    `json:"successful_runs"`
	FailedRuns      int64    `json:"failed_runs"`
	SuccessRate     float64  `json:"success_rate"`
	AvgDuration     float64  `json:"avg_duration"`
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// TaskInfo 任务信息（查询服务使用）
type TaskInfo struct {
	ID                  types.ID `json:"id"`
	Name                string   `json:"name"`
	CronExpression      string   `json:"cron_expression"`
	ExecutionMode       string   `json:"execution_mode"`
	LoadBalanceStrategy string   `json:"load_balance_strategy"`
}

// ExecutorInfo 执行器信息（查询服务使用）
type ExecutorInfo struct {
	ID         types.ID `json:"id"`
	Name       string   `json:"name"`
	InstanceID string   `json:"instance_id"`
	Status     string   `json:"status"`
	BaseURL    string   `json:"base_url"`
}
