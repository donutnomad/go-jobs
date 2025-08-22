package api

import (
	"github.com/jobs/scheduler/internal/models"
)

type CreateTaskReq struct {
	Name                string                     `json:"name" binding:"required"`
	CronExpression      string                     `json:"cron_expression" binding:"required"`
	Parameters          models.JSONMap             `json:"parameters"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
}

type UpdateTaskReq struct {
	Name                string                     `json:"name"`
	CronExpression      string                     `json:"cron_expression"`
	Parameters          models.JSONMap             `json:"parameters"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Status              models.TaskStatus          `json:"status"`
}

type AssignExecutorReq struct {
	ExecutorID string `json:"executor_id" binding:"required"`
	Priority   int    `json:"priority"`
	Weight     int    `json:"weight"`
}

type UpdateExecutorAssignmentReq struct {
	Priority int `json:"priority"`
	Weight   int `json:"weight"`
}

////// executor API  //////

type ListExecutorReq struct {
	IncludeTasks bool `json:"include_tasks"`
}

type UpdateExecutorReq struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	HealthCheckURL string `json:"health_check_url"`
}

///// execution API //////

type ExecutionStatsResp struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
	Running int64 `json:"running"`
	Pending int64 `json:"pending"`
}

type ListExecutionReq struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	TaskID    string `form:"task_id"`
	Status    string `form:"status"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

type ListExecutionResp struct {
	Data       []models.TaskExecution `json:"data"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}
