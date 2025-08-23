package request

import "github.com/jobs/scheduler/internal/domain/entity"

// ExecutionCallbackRequest 执行回调请求
type ExecutionCallbackRequest struct {
	ExecutionID string                 `json:"execution_id" binding:"required"`
	Status      entity.ExecutionStatus `json:"status" binding:"required"`
	Result      map[string]any         `json:"result"`
	Logs        string                 `json:"logs"`
}

// ListExecutionRequest 获取执行记录列表请求
type ListExecutionRequest struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	TaskID    string `form:"task_id"`
	Status    string `form:"status"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

// ExecutionStatsRequest 执行统计请求
type ExecutionStatsRequest struct {
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	TaskID    string `form:"task_id"`
}
