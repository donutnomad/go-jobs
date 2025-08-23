package response

import (
	"time"

	"github.com/jobs/scheduler/internal/domain/entity"
)

// TaskExecutionResponse 任务执行记录响应（与现有models.TaskExecution兼容）
type TaskExecutionResponse struct {
	ID            string                 `json:"id"`
	TaskID        string                 `json:"task_id"`
	ExecutorID    *string                `json:"executor_id"`
	ScheduledTime time.Time              `json:"scheduled_time"`
	StartTime     *time.Time             `json:"start_time"`
	EndTime       *time.Time             `json:"end_time"`
	Status        entity.ExecutionStatus `json:"status"`
	Result        map[string]any         `json:"result"`
	Logs          string                 `json:"logs"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Task          *TaskResponse          `json:"task,omitempty"`
	Executor      *ExecutorResponse      `json:"executor,omitempty"`
}

// ListExecutionResponse 执行记录列表响应
type ListExecutionResponse struct {
	Data       []TaskExecutionResponse `json:"data"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

// ExecutionStatsResponse 执行统计响应
type ExecutionStatsResponse struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
	Running int64 `json:"running"`
	Pending int64 `json:"pending"`
}
