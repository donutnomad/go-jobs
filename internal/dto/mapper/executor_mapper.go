package mapper

import (
	"github.com/jobs/scheduler/internal/domain/entity"
	"github.com/jobs/scheduler/internal/dto/response"
)

// ExecutorMapper 执行器映射器
type ExecutorMapper struct{}

// NewExecutorMapper 创建执行器映射器
func NewExecutorMapper() *ExecutorMapper {
	return &ExecutorMapper{}
}

// ToExecutorResponse 将执行器实体转换为响应DTO
func (m *ExecutorMapper) ToExecutorResponse(executor *entity.Executor) response.ExecutorResponse {
	resp := response.ExecutorResponse{
		ID:                  executor.ID,
		Name:                executor.Name,
		InstanceID:          executor.InstanceID,
		BaseURL:             executor.BaseURL,
		HealthCheckURL:      executor.HealthCheckURL,
		Status:              executor.Status,
		IsHealthy:           executor.IsHealthy,
		HealthCheckFailures: executor.HealthCheckFailures,
		LastHealthCheck:     executor.LastHealthCheck,
		Metadata:            executor.Metadata,
		CreatedAt:           executor.CreatedAt,
		UpdatedAt:           executor.UpdatedAt,
	}

	// 转换关联的TaskExecutors
	if len(executor.TaskExecutors) > 0 {
		resp.TaskExecutors = make([]response.TaskExecutorResponse, len(executor.TaskExecutors))
		for i, te := range executor.TaskExecutors {
			resp.TaskExecutors[i] = response.TaskExecutorResponse{
				ID:           te.ID,
				TaskID:       te.TaskID,
				ExecutorName: te.ExecutorName,
				Priority:     te.Priority,
				Weight:       te.Weight,
			}

			// 转换关联的Task
			if te.Task != nil {
				taskResp := response.TaskResponse{
					ID:                  te.Task.ID,
					Name:                te.Task.Name,
					CronExpression:      te.Task.CronExpression,
					Parameters:          te.Task.Parameters,
					ExecutionMode:       te.Task.ExecutionMode,
					LoadBalanceStrategy: te.Task.LoadBalanceStrategy,
					MaxRetry:            te.Task.MaxRetry,
					TimeoutSeconds:      te.Task.TimeoutSeconds,
					Status:              te.Task.Status,
					CreatedAt:           te.Task.CreatedAt,
					UpdatedAt:           te.Task.UpdatedAt,
				}
				resp.TaskExecutors[i].Task = &taskResp
			}
		}
	}

	return resp
}

// ToExecutorListResponse 将执行器实体列表转换为响应DTO列表
func (m *ExecutorMapper) ToExecutorListResponse(executors []*entity.Executor) []*response.ExecutorResponse {
	responses := make([]*response.ExecutorResponse, len(executors))
	for i, executor := range executors {
		resp := m.ToExecutorResponse(executor)
		responses[i] = &resp
	}
	return responses
}
