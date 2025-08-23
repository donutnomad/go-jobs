package response

import (
	"time"

	"github.com/jobs/scheduler/internal/domain/entity"
)

// TaskResponse 任务响应（与现有models.Task兼容）
type TaskResponse struct {
	ID                  string                     `json:"id"`
	Name                string                     `json:"name"`
	CronExpression      string                     `json:"cron_expression"`
	Parameters          map[string]any             `json:"parameters"`
	ExecutionMode       entity.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy entity.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Status              entity.TaskStatus          `json:"status"`
	CreatedAt           time.Time                  `json:"created_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
	TaskExecutors       []TaskExecutorResponse     `json:"task_executors,omitempty"`
}

// TaskExecutorResponse 任务执行器关联响应
type TaskExecutorResponse struct {
	ID           string            `json:"id"`
	TaskID       string            `json:"task_id"`
	ExecutorName string            `json:"executor_name"`
	Priority     int               `json:"priority"`
	Weight       int               `json:"weight"`
	Executor     *ExecutorResponse `json:"executor,omitempty"`
	Task         *TaskResponse     `json:"task,omitempty"`
}

// TaskStatsResponse 任务统计响应（保持与现有API兼容）
type TaskStatsResponse struct {
	SuccessRate24h   float64                    `json:"success_rate_24h"`
	Total24h         int64                      `json:"total_24h"`
	Success24h       int64                      `json:"success_24h"`
	Health90d        HealthStatusResponse       `json:"health_90d"`
	RecentExecutions []RecentExecutionsResponse `json:"recent_executions"`
	DailyStats90d    []map[string]any           `json:"daily_stats_90d"`
}

// HealthStatusResponse 健康状态响应
type HealthStatusResponse struct {
	HealthScore        float64 `json:"health_score"`
	TotalCount         int64   `json:"total_count"`
	SuccessCount       int64   `json:"success_count"`
	FailedCount        int64   `json:"failed_count"`
	TimeoutCount       int64   `json:"timeout_count"`
	AvgDurationSeconds float64 `json:"avg_duration_seconds"`
	PeriodDays         int     `json:"period_days"`
}

// RecentExecutionsResponse 最近执行统计响应
type RecentExecutionsResponse struct {
	Date        string  `json:"date"`
	Total       int     `json:"total"`
	Success     int     `json:"success"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}
