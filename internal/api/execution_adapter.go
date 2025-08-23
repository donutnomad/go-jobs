package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/handler"
	"github.com/jobs/scheduler/internal/models"
)

// ExecutionAPIAdapter 执行记录API适配器，将新架构适配到现有的生成代码系统
type ExecutionAPIAdapter struct {
	handler handler.IExecutionHandler
}

// NewExecutionAPIAdapter 创建执行记录API适配器
func NewExecutionAPIAdapter(handler handler.IExecutionHandler) IExecutionAPI {
	return &ExecutionAPIAdapter{handler: handler}
}

// 实现原有的IExecutionAPI接口，保持完全兼容

func (a *ExecutionAPIAdapter) List(ctx *gin.Context, req ListExecutionReq) (ListExecutionResp, error) {
	resp, err := a.handler.List(ctx, handler.ListExecutionReq{
		Page:      req.Page,
		PageSize:  req.PageSize,
		TaskID:    req.TaskID,
		Status:    req.Status,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	})
	if err != nil {
		return ListExecutionResp{}, err
	}

	// 转换为原有格式
	data := make([]models.TaskExecution, len(resp.Data))
	for i, execResp := range resp.Data {
		data[i] = models.TaskExecution{
			ID:            execResp.ID,
			TaskID:        execResp.TaskID,
			ExecutorID:    execResp.ExecutorID,
			ScheduledTime: execResp.ScheduledTime,
			StartTime:     execResp.StartTime,
			EndTime:       execResp.EndTime,
			Status:        models.ExecutionStatus(execResp.Status),
			Result:        models.JSONMap(execResp.Result),
			Logs:          execResp.Logs,
			CreatedAt:     execResp.CreatedAt,
		}

		// 转换关联的Task
		if execResp.Task != nil {
			data[i].Task = &models.Task{
				ID:                  execResp.Task.ID,
				Name:                execResp.Task.Name,
				CronExpression:      execResp.Task.CronExpression,
				Parameters:          models.JSONMap(execResp.Task.Parameters),
				ExecutionMode:       models.ExecutionMode(execResp.Task.ExecutionMode),
				LoadBalanceStrategy: models.LoadBalanceStrategy(execResp.Task.LoadBalanceStrategy),
				MaxRetry:            execResp.Task.MaxRetry,
				TimeoutSeconds:      execResp.Task.TimeoutSeconds,
				Status:              models.TaskStatus(execResp.Task.Status),
				CreatedAt:           execResp.Task.CreatedAt,
				UpdatedAt:           execResp.Task.UpdatedAt,
			}
		}

		// 转换关联的Executor
		if execResp.Executor != nil {
			data[i].Executor = &models.Executor{
				ID:                  execResp.Executor.ID,
				Name:                execResp.Executor.Name,
				InstanceID:          execResp.Executor.InstanceID,
				BaseURL:             execResp.Executor.BaseURL,
				HealthCheckURL:      execResp.Executor.HealthCheckURL,
				Status:              models.ExecutorStatus(execResp.Executor.Status),
				IsHealthy:           execResp.Executor.IsHealthy,
				HealthCheckFailures: execResp.Executor.HealthCheckFailures,
				LastHealthCheck:     execResp.Executor.LastHealthCheck,
				Metadata:            execResp.Executor.Metadata,
				CreatedAt:           execResp.Executor.CreatedAt,
				UpdatedAt:           execResp.Executor.UpdatedAt,
			}
		}
	}

	return ListExecutionResp{
		Data:       data,
		Total:      resp.Total,
		Page:       resp.Page,
		PageSize:   resp.PageSize,
		TotalPages: resp.TotalPages,
	}, nil
}

func (a *ExecutionAPIAdapter) Get(ctx *gin.Context, id string) (*models.TaskExecution, error) {
	resp, err := a.handler.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	execution := &models.TaskExecution{
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

	// 转换关联的Task
	if resp.Task != nil {
		execution.Task = &models.Task{
			ID:                  resp.Task.ID,
			Name:                resp.Task.Name,
			CronExpression:      resp.Task.CronExpression,
			Parameters:          models.JSONMap(resp.Task.Parameters),
			ExecutionMode:       models.ExecutionMode(resp.Task.ExecutionMode),
			LoadBalanceStrategy: models.LoadBalanceStrategy(resp.Task.LoadBalanceStrategy),
			MaxRetry:            resp.Task.MaxRetry,
			TimeoutSeconds:      resp.Task.TimeoutSeconds,
			Status:              models.TaskStatus(resp.Task.Status),
			CreatedAt:           resp.Task.CreatedAt,
			UpdatedAt:           resp.Task.UpdatedAt,
		}
	}

	// 转换关联的Executor
	if resp.Executor != nil {
		execution.Executor = &models.Executor{
			ID:                  resp.Executor.ID,
			Name:                resp.Executor.Name,
			InstanceID:          resp.Executor.InstanceID,
			BaseURL:             resp.Executor.BaseURL,
			HealthCheckURL:      resp.Executor.HealthCheckURL,
			Status:              models.ExecutorStatus(resp.Executor.Status),
			IsHealthy:           resp.Executor.IsHealthy,
			HealthCheckFailures: resp.Executor.HealthCheckFailures,
			LastHealthCheck:     resp.Executor.LastHealthCheck,
			Metadata:            resp.Executor.Metadata,
			CreatedAt:           resp.Executor.CreatedAt,
			UpdatedAt:           resp.Executor.UpdatedAt,
		}
	}

	return execution, nil
}

func (a *ExecutionAPIAdapter) Stats(ctx *gin.Context, req ExecutionStatsReq) (ExecutionStatsResp, error) {
	resp, err := a.handler.Stats(ctx, handler.ExecutionStatsReq{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		TaskID:    req.TaskID,
	})
	if err != nil {
		return ExecutionStatsResp{}, err
	}

	return ExecutionStatsResp{
		Total:   resp.Total,
		Success: resp.Success,
		Failed:  resp.Failed,
		Running: resp.Running,
		Pending: resp.Pending,
	}, nil
}

func (a *ExecutionAPIAdapter) Callback(ctx *gin.Context, id string, req ExecutionCallbackRequest) (string, error) {
	return a.handler.Callback(ctx, id, handler.ExecutionCallbackRequest{
		ExecutionID: req.ExecutionID,
		Status:      req.Status,
		Result:      req.Result,
		Logs:        req.Logs,
	})
}

func (a *ExecutionAPIAdapter) Stop(ctx *gin.Context, id string) (string, error) {
	return a.handler.Stop(ctx, id)
}
