package api

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/handler"
	"github.com/jobs/scheduler/internal/models"
)

// TaskAPIAdapter 任务API适配器，将新架构适配到现有的生成代码系统
type TaskAPIAdapter struct {
	handler handler.ITaskHandler
}

// NewTaskAPIAdapter 创建任务API适配器
func NewTaskAPIAdapter(handler handler.ITaskHandler) ITaskAPI {
	return &TaskAPIAdapter{handler: handler}
}

// 实现原有的ITaskAPI接口，保持完全兼容

func (a *TaskAPIAdapter) List(ctx *gin.Context, req GetTasksReq) ([]models.Task, error) {
	// 调用新的handler
	responses, err := a.handler.List(ctx, handler.GetTasksReq{Status: req.Status})
	if err != nil {
		return nil, err
	}

	// 转换为原有的models.Task格式以保持兼容性
	tasks := make([]models.Task, len(responses))
	for i, resp := range responses {
		tasks[i] = models.Task{
			ID:                  resp.ID,
			Name:                resp.Name,
			CronExpression:      resp.CronExpression,
			Parameters:          models.JSONMap(resp.Parameters),
			ExecutionMode:       models.ExecutionMode(resp.ExecutionMode),
			LoadBalanceStrategy: models.LoadBalanceStrategy(resp.LoadBalanceStrategy),
			MaxRetry:            resp.MaxRetry,
			TimeoutSeconds:      resp.TimeoutSeconds,
			Status:              models.TaskStatus(resp.Status),
			CreatedAt:           resp.CreatedAt,
			UpdatedAt:           resp.UpdatedAt,
		}

		// 恢复关联字段转换
		if len(resp.TaskExecutors) > 0 {
			tasks[i].TaskExecutors = make([]models.TaskExecutor, len(resp.TaskExecutors))
			for j, te := range resp.TaskExecutors {
				tasks[i].TaskExecutors[j] = models.TaskExecutor{
					ID:           te.ID,
					TaskID:       te.TaskID,
					ExecutorName: te.ExecutorName,
					Priority:     te.Priority,
					Weight:       te.Weight,
				}

				// 转换Executor
				if te.Executor != nil {
					tasks[i].TaskExecutors[j].Executor = &models.Executor{
						ID:                  te.Executor.ID,
						Name:                te.Executor.Name,
						InstanceID:          te.Executor.InstanceID,
						BaseURL:             te.Executor.BaseURL,
						HealthCheckURL:      te.Executor.HealthCheckURL,
						Status:              models.ExecutorStatus(te.Executor.Status),
						IsHealthy:           te.Executor.IsHealthy,
						HealthCheckFailures: te.Executor.HealthCheckFailures,
						LastHealthCheck:     te.Executor.LastHealthCheck,
						Metadata:            models.JSONMap(te.Executor.Metadata),
						CreatedAt:           te.Executor.CreatedAt,
						UpdatedAt:           te.Executor.UpdatedAt,
					}
				}
			}
		}
	}

	return tasks, nil
}

func (a *TaskAPIAdapter) Get(ctx *gin.Context, id string) (models.Task, error) {
	resp, err := a.handler.Get(ctx, id)
	if err != nil {
		return models.Task{}, err
	}

	fmt.Println("详情内容")
	spew.Dump(resp)

	task := models.Task{
		ID:                  resp.ID,
		Name:                resp.Name,
		CronExpression:      resp.CronExpression,
		Parameters:          models.JSONMap(resp.Parameters),
		ExecutionMode:       models.ExecutionMode(resp.ExecutionMode),
		LoadBalanceStrategy: models.LoadBalanceStrategy(resp.LoadBalanceStrategy),
		MaxRetry:            resp.MaxRetry,
		TimeoutSeconds:      resp.TimeoutSeconds,
		Status:              models.TaskStatus(resp.Status),
		CreatedAt:           resp.CreatedAt,
		UpdatedAt:           resp.UpdatedAt,
	}

	// 恢复关联字段的转换
	if len(resp.TaskExecutors) > 0 {
		task.TaskExecutors = make([]models.TaskExecutor, len(resp.TaskExecutors))
		for i, te := range resp.TaskExecutors {
			task.TaskExecutors[i] = models.TaskExecutor{
				ID:           te.ID,
				TaskID:       te.TaskID,
				ExecutorName: te.ExecutorName,
				Priority:     te.Priority,
				Weight:       te.Weight,
			}

			// 转换Executor
			if te.Executor != nil {
				task.TaskExecutors[i].Executor = &models.Executor{
					ID:                  te.Executor.ID,
					Name:                te.Executor.Name,
					InstanceID:          te.Executor.InstanceID,
					BaseURL:             te.Executor.BaseURL,
					HealthCheckURL:      te.Executor.HealthCheckURL,
					Status:              models.ExecutorStatus(te.Executor.Status),
					IsHealthy:           te.Executor.IsHealthy,
					HealthCheckFailures: te.Executor.HealthCheckFailures,
					LastHealthCheck:     te.Executor.LastHealthCheck,
					Metadata:            models.JSONMap(te.Executor.Metadata),
					CreatedAt:           te.Executor.CreatedAt,
					UpdatedAt:           te.Executor.UpdatedAt,
				}
			}
		}
	}

	return task, nil
}

func (a *TaskAPIAdapter) Create(ctx *gin.Context, req CreateTaskReq) (models.Task, error) {
	resp, err := a.handler.Create(ctx, handler.CreateTaskReq{
		Name:                req.Name,
		CronExpression:      req.CronExpression,
		Parameters:          models.JSONMap(req.Parameters),
		ExecutionMode:       models.ExecutionMode(req.ExecutionMode),
		LoadBalanceStrategy: models.LoadBalanceStrategy(req.LoadBalanceStrategy),
		MaxRetry:            req.MaxRetry,
		TimeoutSeconds:      req.TimeoutSeconds,
	})
	if err != nil {
		return models.Task{}, err
	}

	return models.Task{
		ID:                  resp.ID,
		Name:                resp.Name,
		CronExpression:      resp.CronExpression,
		Parameters:          models.JSONMap(resp.Parameters),
		ExecutionMode:       models.ExecutionMode(resp.ExecutionMode),
		LoadBalanceStrategy: models.LoadBalanceStrategy(resp.LoadBalanceStrategy),
		MaxRetry:            resp.MaxRetry,
		TimeoutSeconds:      resp.TimeoutSeconds,
		Status:              models.TaskStatus(resp.Status),
		CreatedAt:           resp.CreatedAt,
		UpdatedAt:           resp.UpdatedAt,
	}, nil
}

func (a *TaskAPIAdapter) Delete(ctx *gin.Context, id string) (string, error) {
	return a.handler.Delete(ctx, id)
}

func (a *TaskAPIAdapter) UpdateTask(ctx *gin.Context, id string, req UpdateTaskReq) (models.Task, error) {
	resp, err := a.handler.UpdateTask(ctx, id, handler.UpdateTaskReq{
		Name:                req.Name,
		CronExpression:      req.CronExpression,
		Parameters:          models.JSONMap(req.Parameters),
		ExecutionMode:       models.ExecutionMode(req.ExecutionMode),
		LoadBalanceStrategy: models.LoadBalanceStrategy(req.LoadBalanceStrategy),
		MaxRetry:            req.MaxRetry,
		TimeoutSeconds:      req.TimeoutSeconds,
		Status:              models.TaskStatus(req.Status),
	})
	if err != nil {
		return models.Task{}, err
	}

	return models.Task{
		ID:                  resp.ID,
		Name:                resp.Name,
		CronExpression:      resp.CronExpression,
		Parameters:          models.JSONMap(resp.Parameters),
		ExecutionMode:       models.ExecutionMode(resp.ExecutionMode),
		LoadBalanceStrategy: models.LoadBalanceStrategy(resp.LoadBalanceStrategy),
		MaxRetry:            resp.MaxRetry,
		TimeoutSeconds:      resp.TimeoutSeconds,
		Status:              models.TaskStatus(resp.Status),
		CreatedAt:           resp.CreatedAt,
		UpdatedAt:           resp.UpdatedAt,
	}, nil
}

func (a *TaskAPIAdapter) TriggerTask(ctx *gin.Context, id string, req TriggerTaskRequest) (models.TaskExecution, error) {
	resp, err := a.handler.TriggerTask(ctx, id, handler.TriggerTaskRequest{
		Parameters: req.Parameters,
	})
	if err != nil {
		return models.TaskExecution{}, err
	}

	execution := models.TaskExecution{
		ID:            resp.ID,
		TaskID:        resp.TaskID,
		ExecutorID:    resp.ExecutorID,
		ScheduledTime: resp.ScheduledTime,
		StartTime:     resp.StartTime,
		EndTime:       resp.EndTime,
		Status:        models.ExecutionStatus(resp.Status),
		Result:        models.JSONMap(resp.Result),
		Logs:          resp.Logs,
		CreatedAt:     resp.CreatedAt,
	}

	return execution, nil
}

func (a *TaskAPIAdapter) Pause(ctx *gin.Context, id string) (string, error) {
	return a.handler.Pause(ctx, id)
}

func (a *TaskAPIAdapter) Resume(ctx *gin.Context, id string) (string, error) {
	return a.handler.Resume(ctx, id)
}

func (a *TaskAPIAdapter) GetTaskExecutors(ctx *gin.Context, id string) ([]models.TaskExecutor, error) {
	responses, err := a.handler.GetTaskExecutors(ctx, id)
	if err != nil {
		return nil, err
	}

	taskExecutors := make([]models.TaskExecutor, len(responses))
	for i, resp := range responses {
		taskExecutors[i] = models.TaskExecutor{
			ID:           resp.ID,
			TaskID:       resp.TaskID,
			ExecutorName: resp.ExecutorName,
			Priority:     resp.Priority,
			Weight:       resp.Weight,
		}
	}

	return taskExecutors, nil
}

func (a *TaskAPIAdapter) AssignExecutor(ctx *gin.Context, id string, req AssignExecutorReq) (models.TaskExecutor, error) {
	resp, err := a.handler.AssignExecutor(ctx, id, handler.AssignExecutorReq{
		ExecutorID: req.ExecutorID,
		Priority:   req.Priority,
		Weight:     req.Weight,
	})
	if err != nil {
		return models.TaskExecutor{}, err
	}

	return models.TaskExecutor{
		ID:           resp.ID,
		TaskID:       resp.TaskID,
		ExecutorName: resp.ExecutorName,
		Priority:     resp.Priority,
		Weight:       resp.Weight,
	}, nil
}

func (a *TaskAPIAdapter) UpdateExecutorAssignment(ctx *gin.Context, id string, executorID string, req UpdateExecutorAssignmentReq) (models.TaskExecutor, error) {
	resp, err := a.handler.UpdateExecutorAssignment(ctx, id, executorID, handler.UpdateExecutorAssignmentReq{
		Priority: req.Priority,
		Weight:   req.Weight,
	})
	if err != nil {
		return models.TaskExecutor{}, err
	}

	return models.TaskExecutor{
		ID:           resp.ID,
		TaskID:       resp.TaskID,
		ExecutorName: resp.ExecutorName,
		Priority:     resp.Priority,
		Weight:       resp.Weight,
	}, nil
}

func (a *TaskAPIAdapter) UnassignExecutor(ctx *gin.Context, id string, executorID string) (string, error) {
	return a.handler.UnassignExecutor(ctx, id, executorID)
}

func (a *TaskAPIAdapter) GetTaskStats(ctx *gin.Context, taskID string) (TaskStatsResp, error) {
	resp, err := a.handler.GetTaskStats(ctx, taskID)
	if err != nil {
		return TaskStatsResp{}, err
	}

	// 转换为原有格式
	recentExecutions := make([]RecentExecutions, len(resp.RecentExecutions))
	for i, re := range resp.RecentExecutions {
		recentExecutions[i] = RecentExecutions{
			Date:        re.Date,
			Total:       re.Total,
			Success:     re.Success,
			Failed:      re.Failed,
			SuccessRate: re.SuccessRate,
		}
	}

	return TaskStatsResp{
		SuccessRate24h: resp.SuccessRate24h,
		Total24h:       resp.Total24h,
		Success24h:     resp.Success24h,
		Health90d: HealthStatus{
			HealthScore:        resp.Health90d.HealthScore,
			TotalCount:         resp.Health90d.TotalCount,
			SuccessCount:       resp.Health90d.SuccessCount,
			FailedCount:        resp.Health90d.FailedCount,
			TimeoutCount:       resp.Health90d.TimeoutCount,
			AvgDurationSeconds: resp.Health90d.AvgDurationSeconds,
			PeriodDays:         resp.Health90d.PeriodDays,
		},
		RecentExecutions: recentExecutions,
		DailyStats90d:    resp.DailyStats90d,
	}, nil
}
