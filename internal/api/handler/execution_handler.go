package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/domain/entity"
	"github.com/jobs/scheduler/internal/domain/repository"
	"github.com/jobs/scheduler/internal/dto/mapper"
	"github.com/jobs/scheduler/internal/dto/response"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/service"
)

// IExecutionHandler 执行记录处理器接口（与原有IExecutionAPI保持兼容）
type IExecutionHandler interface {
	// List 获取执行历史列表
	// 获取所有的执行历史列表
	// @GET(api/v1/executions)
	List(ctx *gin.Context, req ListExecutionReq) (response.ListExecutionResponse, error)

	// Get 获取执行历史详情
	// 获取指定id的执行历史详情
	// @GET(api/v1/executions/{id})
	Get(ctx *gin.Context, id string) (*response.TaskExecutionResponse, error)

	// Stats 获取执行统计
	// 获取指定任务的执行统计
	// @GET(api/v1/executions/stats)
	Stats(ctx *gin.Context, req ExecutionStatsReq) (response.ExecutionStatsResponse, error)

	// Callback 执行回调
	// 执行指定id的执行回调
	// @POST(api/v1/executions/{id}/callback)
	Callback(ctx *gin.Context, id string, req ExecutionCallbackRequest) (string, error)

	// Stop 停止执行
	// 停止指定id的执行
	// @POST(api/v1/executions/{id}/stop)
	Stop(ctx *gin.Context, id string) (string, error)
}

// 保持与原有API兼容的请求类型
type ListExecutionReq struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	TaskID    string `form:"task_id"`
	Status    string `form:"status"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

type ExecutionStatsReq struct {
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	TaskID    string `form:"task_id"`
}

type ExecutionCallbackRequest struct {
	ExecutionID string                 `json:"execution_id" binding:"required"`
	Status      models.ExecutionStatus `json:"status" binding:"required"`
	Result      map[string]any         `json:"result"`
	Logs        string                 `json:"logs"`
}

type ExecutionHandler struct {
	executionService service.IExecutionService
	executionMapper  *mapper.ExecutionMapper
}

// NewExecutionHandler 创建执行记录处理器
func NewExecutionHandler(executionService service.IExecutionService, executionMapper *mapper.ExecutionMapper) IExecutionHandler {
	return &ExecutionHandler{
		executionService: executionService,
		executionMapper:  executionMapper,
	}
}

func (h *ExecutionHandler) List(ctx *gin.Context, req ListExecutionReq) (response.ListExecutionResponse, error) {
	// 构建过滤器
	filter := repository.ExecutionFilter{
		TaskID:   req.TaskID,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	if req.Status != "" {
		filter.Status = entity.ExecutionStatus(req.Status)
	}

	// 解析时间
	if req.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			filter.StartTime = &startTime
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			filter.EndTime = &endTime
		}
	}

	executions, total, err := h.executionService.ListExecutions(ctx, filter)
	if err != nil {
		return response.ListExecutionResponse{}, err
	}

	// 设置默认分页参数
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	return h.executionMapper.ToExecutionListResponse(executions, total, page, pageSize), nil
}

func (h *ExecutionHandler) Get(ctx *gin.Context, id string) (*response.TaskExecutionResponse, error) {
	execution, err := h.executionService.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := h.executionMapper.ToExecutionResponse(execution)
	return &resp, nil
}

func (h *ExecutionHandler) Stats(ctx *gin.Context, req ExecutionStatsReq) (response.ExecutionStatsResponse, error) {
	// 构建过滤器
	filter := repository.ExecutionStatsFilter{
		TaskID: req.TaskID,
	}

	// 解析时间
	if req.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			filter.StartTime = &startTime
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			filter.EndTime = &endTime
		}
	}

	stats, err := h.executionService.GetExecutionStats(ctx, filter)
	if err != nil {
		return response.ExecutionStatsResponse{}, err
	}

	return h.executionMapper.ToExecutionStatsResponse(stats), nil
}

func (h *ExecutionHandler) Callback(ctx *gin.Context, id string, req ExecutionCallbackRequest) (string, error) {
	err := h.executionService.ProcessCallback(
		ctx,
		id,
		entity.ExecutionStatus(req.Status),
		req.Result,
		req.Logs,
	)
	if err != nil {
		return "", err
	}

	return "callback processed", nil
}

func (h *ExecutionHandler) Stop(ctx *gin.Context, id string) (string, error) {
	if err := h.executionService.StopExecution(ctx, id); err != nil {
		return "", err
	}
	return "stop request sent to executor", nil
}
