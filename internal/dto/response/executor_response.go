package response

import (
	"time"

	"github.com/jobs/scheduler/internal/domain/entity"
)

// ExecutorResponse 执行器响应（与现有models.Executor兼容）
type ExecutorResponse struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	InstanceID          string                 `json:"instance_id"`
	BaseURL             string                 `json:"base_url"`
	HealthCheckURL      string                 `json:"health_check_url"`
	Status              entity.ExecutorStatus  `json:"status"`
	IsHealthy           bool                   `json:"is_healthy"`
	HealthCheckFailures int                    `json:"health_check_failures"`
	LastHealthCheck     *time.Time             `json:"last_health_check"`
	Metadata            map[string]any         `json:"metadata"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	TaskExecutors       []TaskExecutorResponse `json:"task_executors,omitempty"`
}
