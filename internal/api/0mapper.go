package api

import (
	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/models"
)

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name                string                     `json:"name" binding:"required"`
	CronExpression      string                     `json:"cron_expression" binding:"required"`
	Parameters          models.JSONMap             `json:"parameters"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name                string                     `json:"name"`
	CronExpression      string                     `json:"cron_expression"`
	Parameters          models.JSONMap             `json:"parameters"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Status              models.TaskStatus          `json:"status"`
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

// generateID 生成UUID
func generateID() string {
	return uuid.New().String()
}

////// executor API  //////

type ListExecutorRequest struct {
	IncludeTasks bool `json:"include_tasks"`
}

type UpdateExecutorRequest struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	HealthCheckURL string `json:"health_check_url"`
}
