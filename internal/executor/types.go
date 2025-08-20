package executor

import "github.com/jobs/scheduler/internal/models"

// TaskDefinition 任务定义
type TaskDefinition struct {
	Name                string                     `json:"name" binding:"required"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode" binding:"required"`
	CronExpression      string                     `json:"cron_expression" binding:"required"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy" binding:"required"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Parameters          map[string]interface{}     `json:"parameters"`
	Status              models.TaskStatus          `json:"status"` // 初始状态，可以是 active 或 paused
}

// RegisterRequest 执行器注册请求
type RegisterRequest struct {
	ExecutorID     string                 `json:"executor_id" binding:"required"`   // 执行器唯一ID
	ExecutorName   string                 `json:"executor_name" binding:"required"` // 执行器名称
	ExecutorURL    string                 `json:"executor_url" binding:"required"`  // 执行器URL
	HealthCheckURL string                 `json:"health_check_url"`                 // 健康检查URL（可选）
	Tasks          []TaskDefinition       `json:"tasks"`                            // 任务定义列表
	Metadata       map[string]interface{} `json:"metadata"`                         // 元数据
}

// UpdateStatusRequest 更新执行器状态请求
type UpdateStatusRequest struct {
	Status models.ExecutorStatus `json:"status" binding:"required"`
	Reason string                `json:"reason"`
}

// ExecutionCallbackRequest 执行回调请求
type ExecutionCallbackRequest struct {
	ExecutionID string                 `json:"execution_id" binding:"required"`
	Status      models.ExecutionStatus `json:"status" binding:"required"`
	Result      map[string]interface{} `json:"result"`
	Logs        string                 `json:"logs"`
}

// TriggerTaskRequest 触发任务请求
type TriggerTaskRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
}
