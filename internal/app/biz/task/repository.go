package task

import (
	"context"
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// Repository 任务仓储接口
// 遵循DDD原则，定义任务聚合根的持久化操作
type Repository interface {
	// 基础CRUD操作
	Save(ctx context.Context, task *Task) error
	FindByID(ctx context.Context, id types.ID) (*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id types.ID) error

	// 查询操作
	FindByName(ctx context.Context, name string) (*Task, error)
	FindByStatus(ctx context.Context, status TaskStatus, pagination types.Pagination) ([]*Task, error)
	FindAll(ctx context.Context, pagination types.Pagination) ([]*Task, error)
	FindActiveSchedulableTasks(ctx context.Context) ([]*Task, error)

	// 条件查询
	FindByFilters(ctx context.Context, filters TaskFilters, pagination types.Pagination) ([]*Task, error)
	Count(ctx context.Context, filters TaskFilters) (int64, error)

	// 任务调度相关
	FindTasksReadyForScheduling(ctx context.Context, currentTime time.Time, limit int) ([]*Task, error)
	FindTasksByExecutionMode(ctx context.Context, mode ExecutionMode) ([]*Task, error)

	// 批量操作
	BatchUpdateStatus(ctx context.Context, taskIDs []types.ID, status TaskStatus) error
	BatchDelete(ctx context.Context, taskIDs []types.ID) error

	// 统计操作
	GetStatusCounts(ctx context.Context) (map[TaskStatus]int64, error)
	GetExecutionModeStats(ctx context.Context) (map[ExecutionMode]int64, error)

	// 存在性检查
	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsByID(ctx context.Context, id types.ID) (bool, error)
}

// QueryService 任务查询服务接口
// 专门用于复杂查询操作，遵循CQRS原则
type QueryService interface {
	// 复杂查询
	GetTaskOverview(ctx context.Context, taskID types.ID) (*TaskOverview, error)
	GetTasksWithExecutors(ctx context.Context, filters TaskFilters, pagination types.Pagination) ([]*TaskWithExecutors, error)
	GetTaskScheduleHistory(ctx context.Context, taskID types.ID, days int) ([]*ScheduleRecord, error)

	// 统计查询
	GetTaskStatistics(ctx context.Context, timeRange TimeRange) (*TaskStatistics, error)
	GetExecutorDistribution(ctx context.Context, taskID types.ID) (*ExecutorDistribution, error)

	// 搜索功能
	SearchTasks(ctx context.Context, query string, pagination types.Pagination) ([]*Task, error)
}

// TaskFilters 任务过滤条件
type TaskFilters struct {
	Name                string               `json:"name,omitempty"`
	Status              *TaskStatus          `json:"status,omitempty"`
	ExecutionMode       *ExecutionMode       `json:"execution_mode,omitempty"`
	LoadBalanceStrategy *LoadBalanceStrategy `json:"load_balance_strategy,omitempty"`
	CreatedAfter        *time.Time           `json:"created_after,omitempty"`
	CreatedBefore       *time.Time           `json:"created_before,omitempty"`
	UpdatedAfter        *time.Time           `json:"updated_after,omitempty"`
	UpdatedBefore       *time.Time           `json:"updated_before,omitempty"`
}

// TaskOverview 任务概览
type TaskOverview struct {
	Task              *Task           `json:"task"`
	ExecutorCount     int             `json:"executor_count"`
	LastExecutionTime *time.Time      `json:"last_execution_time,omitempty"`
	NextExecutionTime *time.Time      `json:"next_execution_time,omitempty"`
	ExecutionStats    *ExecutionStats `json:"execution_stats"`
}

// TaskWithExecutors 任务及其执行器
type TaskWithExecutors struct {
	Task      *Task       `json:"task"`
	Executors []*Executor `json:"executors"`
}

// ScheduleRecord 调度记录
type ScheduleRecord struct {
	TaskID        types.ID  `json:"task_id"`
	ScheduledTime time.Time `json:"scheduled_time"`
	ExecutorID    types.ID  `json:"executor_id"`
	Status        string    `json:"status"`
	Duration      int64     `json:"duration"` // 毫秒
}

// TaskStatistics 任务统计
type TaskStatistics struct {
	TotalTasks         int64                   `json:"total_tasks"`
	ActiveTasks        int64                   `json:"active_tasks"`
	PausedTasks        int64                   `json:"paused_tasks"`
	DeletedTasks       int64                   `json:"deleted_tasks"`
	StatusDistribution map[TaskStatus]int64    `json:"status_distribution"`
	ModeDistribution   map[ExecutionMode]int64 `json:"mode_distribution"`
	TotalExecutions    int64                   `json:"total_executions"`
	SuccessRate        float64                 `json:"success_rate"`
	TimeRange          TimeRange               `json:"time_range"`
}

// ExecutorDistribution 执行器分布
type ExecutorDistribution struct {
	TaskID       types.ID         `json:"task_id"`
	TotalCount   int              `json:"total_count"`
	ActiveCount  int              `json:"active_count"`
	Distribution map[types.ID]int `json:"distribution"`
}

// ExecutionStats 执行统计
type ExecutionStats struct {
	TotalExecutions   int64      `json:"total_executions"`
	SuccessfulRuns    int64      `json:"successful_runs"`
	FailedRuns        int64      `json:"failed_runs"`
	SkippedRuns       int64      `json:"skipped_runs"`
	SuccessRate       float64    `json:"success_rate"`
	AvgDuration       int64      `json:"avg_duration"` // 毫秒
	LastExecutionTime *time.Time `json:"last_execution_time,omitempty"`
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// Executor 执行器信息（查询服务使用）
type Executor struct {
	ID         types.ID `json:"id"`
	Name       string   `json:"name"`
	InstanceID string   `json:"instance_id"`
	Status     string   `json:"status"`
	BaseURL    string   `json:"base_url"`
}

// UpdateTaskRequest 更新任务请求（实体方法使用）
type UpdateTaskRequest struct {
	Name                string              `json:"name,omitempty"`
	CronExpression      string              `json:"cron_expression,omitempty"`
	Parameters          types.JSONMap       `json:"parameters,omitempty"`
	ExecutionMode       ExecutionMode       `json:"execution_mode,omitempty"`
	LoadBalanceStrategy LoadBalanceStrategy `json:"load_balance_strategy,omitempty"`
	MaxRetry            int                 `json:"max_retry,omitempty"`
	TimeoutSeconds      int                 `json:"timeout_seconds,omitempty"`
	Status              TaskStatus          `json:"status,omitempty"`
}

// Validate 验证更新请求
func (r *UpdateTaskRequest) Validate() error {
	// 只有在非空字符串时才验证
	if r.Name == "" && r.CronExpression == "" && r.ExecutionMode == "" &&
		r.LoadBalanceStrategy == "" && r.MaxRetry <= 0 && r.TimeoutSeconds <= 0 &&
		r.Status == 0 && r.Parameters == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}

	// 验证执行模式
	if r.ExecutionMode != "" && !r.ExecutionMode.IsValid() {
		return fmt.Errorf("invalid execution mode: %s", r.ExecutionMode)
	}

	// 验证负载均衡策略
	if r.LoadBalanceStrategy != "" && !r.LoadBalanceStrategy.IsValid() {
		return fmt.Errorf("invalid load balance strategy: %s", r.LoadBalanceStrategy)
	}

	// 验证配置值
	if r.MaxRetry < 0 {
		return fmt.Errorf("max retry cannot be negative")
	}
	if r.TimeoutSeconds < 0 {
		return fmt.Errorf("timeout seconds cannot be negative")
	}

	return nil
}
