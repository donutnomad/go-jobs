package repository

import (
	"context"
	"time"

	"github.com/jobs/scheduler/internal/domain/entity"
)

// ExecutionRepository 执行记录仓储接口
type ExecutionRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, execution *entity.TaskExecution) error
	GetByID(ctx context.Context, id string) (*entity.TaskExecution, error)
	Update(ctx context.Context, execution *entity.TaskExecution) error
	Delete(ctx context.Context, id string) error

	// 查询操作
	List(ctx context.Context, filter ExecutionFilter) ([]*entity.TaskExecution, int64, error)
	Count(ctx context.Context, filter ExecutionFilter) (int64, error)

	// 统计查询
	GetStats(ctx context.Context, filter ExecutionStatsFilter) (*ExecutionStats, error)
	GetTaskStats(ctx context.Context, taskID string, days int) (*TaskHealthStats, error)
	GetDailyStats(ctx context.Context, taskID string, days int) ([]*DailyStats, error)
	GetRecentExecutions(ctx context.Context, taskID string, days int) ([]*RecentExecutionStats, error)

	// 业务查询
	ListRunning(ctx context.Context) ([]*entity.TaskExecution, error)
	ListPending(ctx context.Context) ([]*entity.TaskExecution, error)
	ListByTaskID(ctx context.Context, taskID string, limit int) ([]*entity.TaskExecution, error)
	CountRunningByExecutor(ctx context.Context, executorID string) (int64, error)
}

// ExecutionFilter 执行记录查询过滤器
type ExecutionFilter struct {
	TaskID    string
	Status    entity.ExecutionStatus
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
}

// ExecutionStatsFilter 执行统计查询过滤器
type ExecutionStatsFilter struct {
	TaskID    string
	StartTime *time.Time
	EndTime   *time.Time
}

// ExecutionStats 执行统计
type ExecutionStats struct {
	Total   int64
	Success int64
	Failed  int64
	Running int64
	Pending int64
}

// TaskHealthStats 任务健康统计
type TaskHealthStats struct {
	HealthScore        float64
	TotalCount         int64
	SuccessCount       int64
	FailedCount        int64
	TimeoutCount       int64
	AvgDurationSeconds float64
	PeriodDays         int
}

// DailyStats 每日统计
type DailyStats struct {
	Date        string
	SuccessRate float64
	Total       int64
}

// RecentExecutionStats 最近执行统计
type RecentExecutionStats struct {
	Date        string
	Total       int
	Success     int
	Failed      int
	SuccessRate float64
}
