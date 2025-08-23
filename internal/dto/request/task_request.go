package request

import "github.com/jobs/scheduler/internal/domain/entity"

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name                string                     `json:"name" binding:"required"`
	CronExpression      string                     `json:"cron_expression" binding:"required"`
	Parameters          map[string]any             `json:"parameters"`
	ExecutionMode       entity.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy entity.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name                string                     `json:"name"`
	CronExpression      string                     `json:"cron_expression"`
	Parameters          map[string]any             `json:"parameters"`
	ExecutionMode       entity.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy entity.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Status              entity.TaskStatus          `json:"status"`
}

// GetTasksRequest 获取任务列表请求
type GetTasksRequest struct {
	Status entity.TaskStatus `form:"status" binding:"omitempty"`
}

// TriggerTaskRequest 触发任务请求
type TriggerTaskRequest struct {
	Parameters map[string]any `json:"parameters"`
}

// AssignExecutorRequest 分配执行器请求
type AssignExecutorRequest struct {
	ExecutorID string `json:"executor_id" binding:"required"`
	Priority   int    `json:"priority"`
	Weight     int    `json:"weight"`
}

// UpdateExecutorAssignmentRequest 更新执行器分配请求
type UpdateExecutorAssignmentRequest struct {
	Priority int `json:"priority"`
	Weight   int `json:"weight"`
}
