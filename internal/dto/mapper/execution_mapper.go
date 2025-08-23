package mapper

import (
	"github.com/jobs/scheduler/internal/domain/entity"
	"github.com/jobs/scheduler/internal/domain/repository"
	"github.com/jobs/scheduler/internal/dto/response"
)

// ExecutionMapper 执行记录映射器
type ExecutionMapper struct {
	taskMapper     *TaskMapper
	executorMapper *ExecutorMapper
}

// NewExecutionMapper 创建执行记录映射器
func NewExecutionMapper(taskMapper *TaskMapper, executorMapper *ExecutorMapper) *ExecutionMapper {
	return &ExecutionMapper{
		taskMapper:     taskMapper,
		executorMapper: executorMapper,
	}
}

// ToExecutionResponse 将执行记录实体转换为响应DTO
func (m *ExecutionMapper) ToExecutionResponse(execution *entity.TaskExecution) response.TaskExecutionResponse {
	resp := response.TaskExecutionResponse{
		ID:            execution.ID,
		TaskID:        execution.TaskID,
		ExecutorID:    execution.ExecutorID,
		ScheduledTime: execution.ScheduledTime,
		StartTime:     execution.StartTime,
		EndTime:       execution.EndTime,
		Status:        execution.Status,
		Result:        execution.Result,
		Logs:          execution.Logs,
		CreatedAt:     execution.CreatedAt,
		UpdatedAt:     execution.UpdatedAt,
	}

	// 转换关联的Task
	if execution.Task != nil {
		taskResp := m.taskMapper.ToTaskResponse(execution.Task)
		resp.Task = &taskResp
	}

	// 转换关联的Executor
	if execution.Executor != nil {
		executorResp := m.executorMapper.ToExecutorResponse(execution.Executor)
		resp.Executor = &executorResp
	}

	return resp
}

// ToExecutionListResponse 将执行记录实体列表转换为列表响应DTO
func (m *ExecutionMapper) ToExecutionListResponse(executions []*entity.TaskExecution, total int64, page, pageSize int) response.ListExecutionResponse {
	data := make([]response.TaskExecutionResponse, len(executions))
	for i, execution := range executions {
		data[i] = m.ToExecutionResponse(execution)
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return response.ListExecutionResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// ToExecutionStatsResponse 将执行统计转换为响应DTO
func (m *ExecutionMapper) ToExecutionStatsResponse(stats *repository.ExecutionStats) response.ExecutionStatsResponse {
	return response.ExecutionStatsResponse{
		Total:   stats.Total,
		Success: stats.Success,
		Failed:  stats.Failed,
		Running: stats.Running,
		Pending: stats.Pending,
	}
}
