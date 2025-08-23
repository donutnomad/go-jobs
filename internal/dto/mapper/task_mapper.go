package mapper

import (
	"github.com/jobs/scheduler/internal/domain/entity"
	"github.com/jobs/scheduler/internal/dto/response"
	"github.com/jobs/scheduler/internal/service"
)

// TaskMapper 任务映射器
type TaskMapper struct{}

// NewTaskMapper 创建任务映射器
func NewTaskMapper() *TaskMapper {
	return &TaskMapper{}
}

// ToTaskResponse 将任务实体转换为响应DTO
func (m *TaskMapper) ToTaskResponse(task *entity.Task) response.TaskResponse {
	resp := response.TaskResponse{
		ID:                  task.ID,
		Name:                task.Name,
		CronExpression:      task.CronExpression,
		Parameters:          task.Parameters,
		ExecutionMode:       task.ExecutionMode,
		LoadBalanceStrategy: task.LoadBalanceStrategy,
		MaxRetry:            task.MaxRetry,
		TimeoutSeconds:      task.TimeoutSeconds,
		Status:              task.Status,
		CreatedAt:           task.CreatedAt,
		UpdatedAt:           task.UpdatedAt,
	}

	// 转换关联的TaskExecutors
	if len(task.TaskExecutors) > 0 {
		resp.TaskExecutors = make([]response.TaskExecutorResponse, len(task.TaskExecutors))
		for i, te := range task.TaskExecutors {
			resp.TaskExecutors[i] = response.TaskExecutorResponse{
				ID:           te.ID,
				TaskID:       te.TaskID,
				ExecutorName: te.ExecutorName,
				Priority:     te.Priority,
				Weight:       te.Weight,
				// TODO: 需要在service层或handler层单独获取Executor信息
			}
		}
	}

	return resp
}

// ToTaskListResponse 将任务实体列表转换为响应DTO列表
func (m *TaskMapper) ToTaskListResponse(tasks []*entity.Task) []response.TaskResponse {
	responses := make([]response.TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = m.ToTaskResponse(task)
	}
	return responses
}

// ToTaskExecutorResponse 将任务执行器实体转换为响应DTO
func (m *TaskMapper) ToTaskExecutorResponse(te *entity.TaskExecutor) response.TaskExecutorResponse {
	return response.TaskExecutorResponse{
		ID:           te.ID,
		TaskID:       te.TaskID,
		ExecutorName: te.ExecutorName,
		Priority:     te.Priority,
		Weight:       te.Weight,
	}
}

// ToTaskExecutorListResponse 将任务执行器实体列表转换为响应DTO列表
func (m *TaskMapper) ToTaskExecutorListResponse(taskExecutors []*entity.TaskExecutor) []response.TaskExecutorResponse {
	responses := make([]response.TaskExecutorResponse, len(taskExecutors))
	for i, te := range taskExecutors {
		responses[i] = m.ToTaskExecutorResponse(te)
	}
	return responses
}

// ToTaskStatsResponse 将任务统计结果转换为响应DTO
func (m *TaskMapper) ToTaskStatsResponse(stats *service.TaskStatsResult) response.TaskStatsResponse {
	resp := response.TaskStatsResponse{
		SuccessRate24h: stats.SuccessRate24h,
		Total24h:       stats.Total24h,
		Success24h:     stats.Success24h,
	}

	// 转换健康度统计
	if stats.Health90d != nil {
		resp.Health90d = response.HealthStatusResponse{
			HealthScore:        stats.Health90d.HealthScore,
			TotalCount:         stats.Health90d.TotalCount,
			SuccessCount:       stats.Health90d.SuccessCount,
			FailedCount:        stats.Health90d.FailedCount,
			TimeoutCount:       stats.Health90d.TimeoutCount,
			AvgDurationSeconds: stats.Health90d.AvgDurationSeconds,
			PeriodDays:         stats.Health90d.PeriodDays,
		}
	}

	// 转换最近执行统计
	if len(stats.RecentExecutions) > 0 {
		resp.RecentExecutions = make([]response.RecentExecutionsResponse, len(stats.RecentExecutions))
		for i, re := range stats.RecentExecutions {
			resp.RecentExecutions[i] = response.RecentExecutionsResponse{
				Date:        re.Date,
				Total:       re.Total,
				Success:     re.Success,
				Failed:      re.Failed,
				SuccessRate: re.SuccessRate,
			}
		}
	}

	// 转换每日统计
	if len(stats.DailyStats90d) > 0 {
		resp.DailyStats90d = make([]map[string]any, len(stats.DailyStats90d))
		for i, ds := range stats.DailyStats90d {
			resp.DailyStats90d[i] = map[string]any{
				"date":        ds.Date,
				"successRate": ds.SuccessRate,
				"total":       ds.Total,
			}
		}
	}

	return resp
}
