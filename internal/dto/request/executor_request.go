package request

import "github.com/jobs/scheduler/internal/domain/entity"

// RegisterExecutorRequest 注册执行器请求
type RegisterExecutorRequest struct {
	ExecutorID     string                  `json:"executor_id" binding:"required"`   // 执行器唯一ID
	ExecutorName   string                  `json:"executor_name" binding:"required"` // 执行器名称
	ExecutorURL    string                  `json:"executor_url" binding:"required"`  // 执行器URL
	HealthCheckURL string                  `json:"health_check_url"`                 // 健康检查URL（可选）
	Tasks          []TaskDefinitionRequest `json:"tasks"`                            // 任务定义列表
	Metadata       map[string]any          `json:"metadata"`                         // 元数据
}

// TaskDefinitionRequest 任务定义请求
type TaskDefinitionRequest struct {
	Name                string                     `json:"name" binding:"required"`
	ExecutionMode       entity.ExecutionMode       `json:"execution_mode" binding:"required"`
	CronExpression      string                     `json:"cron_expression" binding:"required"`
	LoadBalanceStrategy entity.LoadBalanceStrategy `json:"load_balance_strategy" binding:"required"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Parameters          map[string]any             `json:"parameters"`
	Status              entity.TaskStatus          `json:"status"` // 初始状态，可以是 active 或 paused
}

// UpdateExecutorRequest 更新执行器请求
type UpdateExecutorRequest struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	HealthCheckURL string `json:"health_check_url"`
}

// UpdateExecutorStatusRequest 更新执行器状态请求
type UpdateExecutorStatusRequest struct {
	Status entity.ExecutorStatus `json:"status" binding:"required"`
	Reason string                `json:"reason"`
}

// ListExecutorRequest 获取执行器列表请求
type ListExecutorRequest struct {
	IncludeTasks bool `json:"include_tasks"`
}
