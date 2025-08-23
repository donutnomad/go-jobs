package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/domain/entity"
	"github.com/jobs/scheduler/internal/domain/repository"
	"github.com/jobs/scheduler/internal/dto/mapper"
	"github.com/jobs/scheduler/internal/dto/response"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/service"
)

// IExecutorHandler 执行器处理器接口（与原有IExecutorAPI保持兼容）
type IExecutorHandler interface {
	// List 获取执行器列表
	// 获取所有的执行器列表
	// @GET(api/v1/executors)
	List(ctx *gin.Context, req ListExecutorReq) ([]*response.ExecutorResponse, error)

	// Get 获取执行器详情
	// 获取指定id的执行器详情
	// @GET(api/v1/executors/{id})
	Get(ctx *gin.Context, id string) (*response.ExecutorResponse, error)

	// Register 注册执行器
	// 注册一个新执行器
	// @POST(api/v1/executors/register)
	Register(ctx *gin.Context, req RegisterExecutorReq) (*response.ExecutorResponse, error)

	// Update 更新执行器
	// 更新指定id的执行器
	// @PUT(api/v1/executors/{id})
	Update(ctx *gin.Context, id string, req UpdateExecutorReq) (response.ExecutorResponse, error)

	// UpdateStatus 更新执行器状态
	// 更新指定id的执行器状态
	// @PUT(api/v1/executors/{id}/status)
	UpdateStatus(ctx *gin.Context, id string, req UpdateExecutorStatusReq) (string, error)

	// Delete 删除执行器
	// 删除指定id的执行器
	// @DELETE(api/v1/executors/{id})
	Delete(ctx *gin.Context, id string) (string, error)
}

// 保持与原有API兼容的请求类型
type ListExecutorReq struct {
	IncludeTasks bool `form:"include_tasks" json:"include_tasks"`
}

type UpdateExecutorReq struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	HealthCheckURL string `json:"health_check_url"`
}

type RegisterExecutorReq struct {
	ExecutorID     string           `json:"executor_id" binding:"required"`   // 执行器唯一ID
	ExecutorName   string           `json:"executor_name" binding:"required"` // 执行器名称
	ExecutorURL    string           `json:"executor_url" binding:"required"`  // 执行器URL
	HealthCheckURL string           `json:"health_check_url"`                 // 健康检查URL（可选）
	Tasks          []TaskDefinition `json:"tasks"`                            // 任务定义列表
	Metadata       map[string]any   `json:"metadata"`                         // 元数据
}

type TaskDefinition struct {
	Name                string                     `json:"name" binding:"required"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode" binding:"required"`
	CronExpression      string                     `json:"cron_expression" binding:"required"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy" binding:"required"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Parameters          map[string]any             `json:"parameters"`
	Status              models.TaskStatus          `json:"status"` // 初始状态，可以是 active 或 paused
}

type UpdateExecutorStatusReq struct {
	Status models.ExecutorStatus `json:"status" binding:"required"`
	Reason string                `json:"reason"`
}

type ExecutorHandler struct {
	executorService service.IExecutorService
	executorMapper  *mapper.ExecutorMapper
}

// NewExecutorHandler 创建执行器处理器
func NewExecutorHandler(executorService service.IExecutorService, executorMapper *mapper.ExecutorMapper) IExecutorHandler {
	return &ExecutorHandler{
		executorService: executorService,
		executorMapper:  executorMapper,
	}
}

func (h *ExecutorHandler) List(ctx *gin.Context, req ListExecutorReq) ([]*response.ExecutorResponse, error) {
	filter := repository.ExecutorFilter{
		IncludeTasks: req.IncludeTasks,
	}

	executors, err := h.executorService.ListExecutors(ctx, filter)
	if err != nil {
		return nil, err
	}

	return h.executorMapper.ToExecutorListResponse(executors), nil
}

func (h *ExecutorHandler) Get(ctx *gin.Context, id string) (*response.ExecutorResponse, error) {
	executor, err := h.executorService.GetExecutor(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := h.executorMapper.ToExecutorResponse(executor)
	return &resp, nil
}

func (h *ExecutorHandler) Register(ctx *gin.Context, req RegisterExecutorReq) (*response.ExecutorResponse, error) {
	// 转换任务定义
	tasks := make([]service.TaskDefinitionRequest, len(req.Tasks))
	for i, taskDef := range req.Tasks {
		tasks[i] = service.TaskDefinitionRequest{
			Name:                taskDef.Name,
			ExecutionMode:       entity.ExecutionMode(taskDef.ExecutionMode),
			CronExpression:      taskDef.CronExpression,
			LoadBalanceStrategy: entity.LoadBalanceStrategy(taskDef.LoadBalanceStrategy),
			MaxRetry:            taskDef.MaxRetry,
			TimeoutSeconds:      taskDef.TimeoutSeconds,
			Parameters:          taskDef.Parameters,
			Status:              entity.TaskStatus(taskDef.Status),
		}
	}

	serviceReq := &service.RegisterExecutorRequest{
		ExecutorID:     req.ExecutorID,
		ExecutorName:   req.ExecutorName,
		ExecutorURL:    req.ExecutorURL,
		HealthCheckURL: req.HealthCheckURL,
		Tasks:          tasks,
		Metadata:       req.Metadata,
	}

	executor, err := h.executorService.RegisterExecutor(ctx, serviceReq)
	if err != nil {
		return nil, err
	}

	resp := h.executorMapper.ToExecutorResponse(executor)
	return &resp, nil
}

func (h *ExecutorHandler) Update(ctx *gin.Context, id string, req UpdateExecutorReq) (response.ExecutorResponse, error) {
	executor, err := h.executorService.UpdateExecutor(ctx, id, req.Name, req.BaseURL, req.HealthCheckURL)
	if err != nil {
		return response.ExecutorResponse{}, err
	}

	return h.executorMapper.ToExecutorResponse(executor), nil
}

func (h *ExecutorHandler) UpdateStatus(ctx *gin.Context, id string, req UpdateExecutorStatusReq) (string, error) {
	if err := h.executorService.UpdateExecutorStatus(ctx, id, entity.ExecutorStatus(req.Status)); err != nil {
		return "", err
	}
	return "status updated", nil
}

func (h *ExecutorHandler) Delete(ctx *gin.Context, id string) (string, error) {
	if err := h.executorService.DeleteExecutor(ctx, id); err != nil {
		return "", err
	}
	return "executor deleted", nil
}
