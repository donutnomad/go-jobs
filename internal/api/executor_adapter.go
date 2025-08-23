package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/handler"
	"github.com/jobs/scheduler/internal/models"
)

// ExecutorAPIAdapter 执行器API适配器，将新架构适配到现有的生成代码系统
type ExecutorAPIAdapter struct {
	handler handler.IExecutorHandler
}

// NewExecutorAPIAdapter 创建执行器API适配器
func NewExecutorAPIAdapter(handler handler.IExecutorHandler) IExecutorAPI {
	return &ExecutorAPIAdapter{handler: handler}
}

// 实现原有的IExecutorAPI接口，保持完全兼容

func (a *ExecutorAPIAdapter) List(ctx *gin.Context, req ListExecutorReq) ([]*models.Executor, error) {
	responses, err := a.handler.List(ctx, handler.ListExecutorReq{})
	if err != nil {
		return nil, err
	}

	// 转换为原有的models.Executor格式
	executors := make([]*models.Executor, len(responses))
	for i, resp := range responses {
		executors[i] = &models.Executor{
			ID:                  resp.ID,
			Name:                resp.Name,
			InstanceID:          resp.InstanceID,
			BaseURL:             resp.BaseURL,
			HealthCheckURL:      resp.HealthCheckURL,
			Status:              models.ExecutorStatus(resp.Status),
			IsHealthy:           resp.IsHealthy,
			HealthCheckFailures: resp.HealthCheckFailures,
			LastHealthCheck:     resp.LastHealthCheck,
			Metadata:            resp.Metadata,
			CreatedAt:           resp.CreatedAt,
			UpdatedAt:           resp.UpdatedAt,
		}

		// 恢复关联字段转换
		if len(resp.TaskExecutors) > 0 {
			executors[i].TaskExecutors = make([]models.TaskExecutor, len(resp.TaskExecutors))
			for j, te := range resp.TaskExecutors {
				executors[i].TaskExecutors[j] = models.TaskExecutor{
					ID:           te.ID,
					TaskID:       te.TaskID,
					ExecutorName: te.ExecutorName,
					Priority:     te.Priority,
					Weight:       te.Weight,
				}
				
				// 转换关联的Task
				if te.Task != nil {
					executors[i].TaskExecutors[j].Task = &models.Task{
						ID:                  te.Task.ID,
						Name:                te.Task.Name,
						CronExpression:      te.Task.CronExpression,
						Parameters:          models.JSONMap(te.Task.Parameters),
						ExecutionMode:       models.ExecutionMode(te.Task.ExecutionMode),
						LoadBalanceStrategy: models.LoadBalanceStrategy(te.Task.LoadBalanceStrategy),
						MaxRetry:            te.Task.MaxRetry,
						TimeoutSeconds:      te.Task.TimeoutSeconds,
						Status:              models.TaskStatus(te.Task.Status),
						CreatedAt:           te.Task.CreatedAt,
						UpdatedAt:           te.Task.UpdatedAt,
					}
				}
			}
		}
	}

	return executors, nil
}

func (a *ExecutorAPIAdapter) Get(ctx *gin.Context, id string) (*models.Executor, error) {
	resp, err := a.handler.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	executor := &models.Executor{
		ID:                  resp.ID,
		Name:                resp.Name,
		InstanceID:          resp.InstanceID,
		BaseURL:             resp.BaseURL,
		HealthCheckURL:      resp.HealthCheckURL,
		Status:              models.ExecutorStatus(resp.Status),
		IsHealthy:           resp.IsHealthy,
		HealthCheckFailures: resp.HealthCheckFailures,
		LastHealthCheck:     resp.LastHealthCheck,
		Metadata:            resp.Metadata,
		CreatedAt:           resp.CreatedAt,
		UpdatedAt:           resp.UpdatedAt,
	}

	// 转换关联的TaskExecutors
	if len(resp.TaskExecutors) > 0 {
		executor.TaskExecutors = make([]models.TaskExecutor, len(resp.TaskExecutors))
		for j, te := range resp.TaskExecutors {
			executor.TaskExecutors[j] = models.TaskExecutor{
				ID:           te.ID,
				TaskID:       te.TaskID,
				ExecutorName: te.ExecutorName,
				Priority:     te.Priority,
				Weight:       te.Weight,
			}
			
			// 转换关联的Task
			if te.Task != nil {
				executor.TaskExecutors[j].Task = &models.Task{
					ID:                  te.Task.ID,
					Name:                te.Task.Name,
					CronExpression:      te.Task.CronExpression,
					Parameters:          models.JSONMap(te.Task.Parameters),
					ExecutionMode:       models.ExecutionMode(te.Task.ExecutionMode),
					LoadBalanceStrategy: models.LoadBalanceStrategy(te.Task.LoadBalanceStrategy),
					MaxRetry:            te.Task.MaxRetry,
					TimeoutSeconds:      te.Task.TimeoutSeconds,
					Status:              models.TaskStatus(te.Task.Status),
					CreatedAt:           te.Task.CreatedAt,
					UpdatedAt:           te.Task.UpdatedAt,
				}
			}
		}
	}

	return executor, nil
}

func (a *ExecutorAPIAdapter) Register(ctx *gin.Context, req RegisterExecutorReq) (*models.Executor, error) {
	// 转换任务定义
	tasks := make([]handler.TaskDefinition, len(req.Tasks))
	for i, taskDef := range req.Tasks {
		tasks[i] = handler.TaskDefinition{
			Name:                taskDef.Name,
			ExecutionMode:       taskDef.ExecutionMode,
			CronExpression:      taskDef.CronExpression,
			LoadBalanceStrategy: taskDef.LoadBalanceStrategy,
			MaxRetry:            taskDef.MaxRetry,
			TimeoutSeconds:      taskDef.TimeoutSeconds,
			Parameters:          taskDef.Parameters,
			Status:              taskDef.Status,
		}
	}

	resp, err := a.handler.Register(ctx, handler.RegisterExecutorReq{
		ExecutorID:     req.ExecutorID,
		ExecutorName:   req.ExecutorName,
		ExecutorURL:    req.ExecutorURL,
		HealthCheckURL: req.HealthCheckURL,
		Tasks:          tasks,
		Metadata:       req.Metadata,
	})
	if err != nil {
		return nil, err
	}

	return &models.Executor{
		ID:                  resp.ID,
		Name:                resp.Name,
		InstanceID:          resp.InstanceID,
		BaseURL:             resp.BaseURL,
		HealthCheckURL:      resp.HealthCheckURL,
		Status:              models.ExecutorStatus(resp.Status),
		IsHealthy:           resp.IsHealthy,
		HealthCheckFailures: resp.HealthCheckFailures,
		LastHealthCheck:     resp.LastHealthCheck,
		Metadata:            resp.Metadata,
		CreatedAt:           resp.CreatedAt,
		UpdatedAt:           resp.UpdatedAt,
	}, nil
}

func (a *ExecutorAPIAdapter) Update(ctx *gin.Context, id string, req UpdateExecutorReq) (models.Executor, error) {
	resp, err := a.handler.Update(ctx, id, handler.UpdateExecutorReq{
		Name:           req.Name,
		BaseURL:        req.BaseURL,
		HealthCheckURL: req.HealthCheckURL,
	})
	if err != nil {
		return models.Executor{}, err
	}

	return models.Executor{
		ID:                  resp.ID,
		Name:                resp.Name,
		InstanceID:          resp.InstanceID,
		BaseURL:             resp.BaseURL,
		HealthCheckURL:      resp.HealthCheckURL,
		Status:              models.ExecutorStatus(resp.Status),
		IsHealthy:           resp.IsHealthy,
		HealthCheckFailures: resp.HealthCheckFailures,
		LastHealthCheck:     resp.LastHealthCheck,
		Metadata:            resp.Metadata,
		CreatedAt:           resp.CreatedAt,
		UpdatedAt:           resp.UpdatedAt,
	}, nil
}

func (a *ExecutorAPIAdapter) UpdateStatus(ctx *gin.Context, id string, req UpdateExecutorStatusReq) (string, error) {
	return a.handler.UpdateStatus(ctx, id, handler.UpdateExecutorStatusReq{
		Status: req.Status,
		Reason: req.Reason,
	})
}

func (a *ExecutorAPIAdapter) Delete(ctx *gin.Context, id string) (string, error) {
	return a.handler.Delete(ctx, id)
}
