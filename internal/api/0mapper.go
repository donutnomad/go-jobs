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

type TaskStatsResp struct {
	SuccessRate24h   float64            `json:"success_rate_24h"`
	Total24h         int64              `json:"total_24h"`
	Success24h       int64              `json:"success_24h"`
	Health90d        HealthStatus       `json:"health_90d"`
	RecentExecutions []RecentExecutions `json:"recent_executions"`
	DailyStats90d    []map[string]any   `json:"daily_stats_90d"`
}

type RecentExecutions struct {
	Date        string  `json:"date"`
	Total       int     `json:"total"`
	Success     int     `json:"success"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

type TriggerTaskRequest struct {
	Parameters map[string]any `json:"parameters"`
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

type RegisterExecutorReq struct {
	ExecutorID     string           `json:"executor_id" binding:"required"`   // 执行器唯一ID
	ExecutorName   string           `json:"executor_name" binding:"required"` // 执行器名称
	ExecutorURL    string           `json:"executor_url" binding:"required"`  // 执行器URL
	HealthCheckURL string           `json:"health_check_url"`                 // 健康检查URL（可选）
	Tasks          []TaskDefinition `json:"tasks"`                            // 任务定义列表
	Metadata       map[string]any   `json:"metadata"`                         // 元数据
}

type TaskDefinition struct {
	Name                string                     `json:"name" binding:"required"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode" binding:"required"`
	CronExpression      string                     `json:"cron_expression" binding:"required"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy" binding:"required"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Parameters          map[string]any             `json:"parameters"`
	Status              models.TaskStatus          `json:"status"` // 初始状态，可以是 active 或 paused
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
