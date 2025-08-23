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

// ITaskHandler 任务处理器接口（与原有ITaskAPI保持兼容）
type ITaskHandler interface {
	// List 获取任务列表
	// 获取所有的任务列表
	// @GET(api/v1/tasks)
	List(ctx *gin.Context, req GetTasksReq) ([]response.TaskResponse, error)

	// Get 获取任务详情
	// 获取指定id的任务详情
	// @GET(api/v1/tasks/{id})
	Get(ctx *gin.Context, id string) (response.TaskResponse, error)

	// Create 创建任务
	// 创建一个新任务
	// @POST(api/v1/tasks)
	Create(ctx *gin.Context, req CreateTaskReq) (response.TaskResponse, error)

	// Delete 删除任务
	// 删除指定id的任务
	// @DELETE(api/v1/tasks/{id})
	Delete(ctx *gin.Context, id string) (string, error)

	// UpdateTask 更新任务
	// 更新指定id的任务
	// @PUT(api/v1/tasks/{id})
	UpdateTask(ctx *gin.Context, id string, req UpdateTaskReq) (response.TaskResponse, error)

	// TriggerTask 手动触发任务
	// 手动触发指定id的任务
	// @POST(api/v1/tasks/{id}/trigger)
	TriggerTask(ctx *gin.Context, id string, req TriggerTaskRequest) (response.TaskExecutionResponse, error)

	// Pause 暂停任务
	// 暂停指定id的任务
	// @POST(api/v1/tasks/{id}/pause)
	Pause(ctx *gin.Context, id string) (string, error)

	// Resume 恢复任务
	// 恢复指定id的任务
	// @POST(api/v1/tasks/{id}/resume)
	Resume(ctx *gin.Context, id string) (string, error)

	// GetTaskExecutors 获取任务的执行器列表
	// 获取指定id的任务的执行器列表
	// @GET(api/v1/tasks/{id}/executors)
	GetTaskExecutors(ctx *gin.Context, id string) ([]response.TaskExecutorResponse, error)

	// AssignExecutor 为任务分配执行器
	// 为指定id的任务分配执行器
	// @POST(api/v1/tasks/{id}/executors)
	AssignExecutor(ctx *gin.Context, id string, req AssignExecutorReq) (response.TaskExecutorResponse, error)

	// UpdateExecutorAssignment 更新任务执行器分配
	// 更新指定id的任务执行器分配
	// @PUT(api/v1/tasks/{id}/executors/{executor_id})
	UpdateExecutorAssignment(ctx *gin.Context, id string, executorID string, req UpdateExecutorAssignmentReq) (response.TaskExecutorResponse, error)

	// UnassignExecutor 取消任务执行器分配
	// 取消指定id的任务执行器分配
	// @DELETE(api/v1/tasks/{id}/executors/{executor_id})
	UnassignExecutor(ctx *gin.Context, id string, executorID string) (string, error)

	// GetTaskStats 获取任务统计
	// 获取指定id的任务统计
	// @GET(api/v1/tasks/{id}/stats)
	GetTaskStats(ctx *gin.Context, id string) (response.TaskStatsResponse, error)
}

// 保持与原有API兼容的请求类型
type GetTasksReq struct {
	Status models.TaskStatus `form:"status" binding:"omitempty"`
}

type CreateTaskReq struct {
	Name                string                     `json:"name" binding:"required"`
	CronExpression      string                     `json:"cron_expression" binding:"required"`
	Parameters          models.JSONMap             `json:"parameters"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
}

type UpdateTaskReq struct {
	Name                string                     `json:"name"`
	CronExpression      string                     `json:"cron_expression"`
	Parameters          models.JSONMap             `json:"parameters"`
	ExecutionMode       models.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                        `json:"max_retry"`
	TimeoutSeconds      int                        `json:"timeout_seconds"`
	Status              models.TaskStatus          `json:"status"`
}

type TriggerTaskRequest struct {
	Parameters map[string]any `json:"parameters"`
}

type AssignExecutorReq struct {
	ExecutorID string `json:"executor_id" binding:"required"`
	Priority   int    `json:"priority"`
	Weight     int    `json:"weight"`
}

type UpdateExecutorAssignmentReq struct {
	Priority int `json:"priority"`
	Weight   int `json:"weight"`
}

type TaskHandler struct {
	taskService     service.ITaskService
	executorService service.IExecutorService
	taskMapper      *mapper.TaskMapper
	execMapper      *mapper.ExecutionMapper
	executorMapper  *mapper.ExecutorMapper
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(taskService service.ITaskService, executorService service.IExecutorService, taskMapper *mapper.TaskMapper, execMapper *mapper.ExecutionMapper, executorMapper *mapper.ExecutorMapper) ITaskHandler {
	return &TaskHandler{
		taskService:     taskService,
		executorService: executorService,
		taskMapper:      taskMapper,
		execMapper:      execMapper,
		executorMapper:  executorMapper,
	}
}

func (h *TaskHandler) List(ctx *gin.Context, req GetTasksReq) ([]response.TaskResponse, error) {
	filter := repository.TaskFilter{}
	if req.Status != "" {
		filter.Status = entity.TaskStatus(req.Status)
	}

	tasks, err := h.taskService.ListTasks(ctx, filter)
	if err != nil {
		return nil, err
	}

	responses := h.taskMapper.ToTaskListResponse(tasks)

	// 为每个任务的TaskExecutor获取所有同名的Executor实例，并展开为多个条目
	for i := range responses {
		if len(responses[i].TaskExecutors) > 0 {
			var expandedTaskExecutors []response.TaskExecutorResponse

			for _, te := range responses[i].TaskExecutors {
				// 根据ExecutorName获取所有同名的执行器实例
				executors, err := h.executorService.ListExecutors(ctx, repository.ExecutorFilter{Name: te.ExecutorName})
				if err != nil || len(executors) == 0 {
					// 如果获取失败或没有执行器，保留原来的条目但不关联Executor
					expandedTaskExecutors = append(expandedTaskExecutors, te)
				} else {
					// 为每个同名的执行器实例创建一个TaskExecutorResponse
					for _, executor := range executors {
						newTE := response.TaskExecutorResponse{
							ID:           te.ID + "-" + executor.ID, // 组合原始ID和执行器ID，确保唯一性
							TaskID:       te.TaskID,
							ExecutorName: te.ExecutorName,
							Priority:     te.Priority,
							Weight:       te.Weight,
						}

						// 关联具体的执行器实例
						executorResp := h.executorMapper.ToExecutorResponse(executor)
						newTE.Executor = &executorResp

						expandedTaskExecutors = append(expandedTaskExecutors, newTE)
					}
				}
			}

			responses[i].TaskExecutors = expandedTaskExecutors
		}
	}

	return responses, nil
}

func (h *TaskHandler) Get(ctx *gin.Context, id string) (response.TaskResponse, error) {
	task, err := h.taskService.GetTask(ctx, id)
	if err != nil {
		return response.TaskResponse{}, err
	}

	// 转换为响应DTO
	resp := h.taskMapper.ToTaskResponse(task)

	// 为每个TaskExecutor获取所有同名的Executor实例，并为每个实例创建一个TaskExecutorResponse
	if len(resp.TaskExecutors) > 0 {
		var expandedTaskExecutors []response.TaskExecutorResponse

		for _, te := range resp.TaskExecutors {
			// 根据ExecutorName获取所有同名的执行器实例
			executors, err := h.executorService.ListExecutors(ctx, repository.ExecutorFilter{Name: te.ExecutorName})
			if err != nil || len(executors) == 0 {
				// 如果获取失败或没有执行器，保留原来的条目但不关联Executor
				expandedTaskExecutors = append(expandedTaskExecutors, te)
			} else {
				// 为每个同名的执行器实例创建一个TaskExecutorResponse
				for _, executor := range executors {
					newTE := response.TaskExecutorResponse{
						ID:           te.ID + "-" + executor.ID, // 组合原始ID和执行器ID，确保唯一性
						TaskID:       te.TaskID,
						ExecutorName: te.ExecutorName,
						Priority:     te.Priority,
						Weight:       te.Weight,
					}

					// 关联具体的执行器实例
					executorResp := h.executorMapper.ToExecutorResponse(executor)
					newTE.Executor = &executorResp

					expandedTaskExecutors = append(expandedTaskExecutors, newTE)
				}
			}
		}

		resp.TaskExecutors = expandedTaskExecutors
	}

	return resp, nil
}

func (h *TaskHandler) Create(ctx *gin.Context, req CreateTaskReq) (response.TaskResponse, error) {
	task, err := h.taskService.CreateTask(
		ctx,
		req.Name,
		req.CronExpression,
		map[string]any(req.Parameters),
		entity.ExecutionMode(req.ExecutionMode),
		entity.LoadBalanceStrategy(req.LoadBalanceStrategy),
		req.MaxRetry,
		req.TimeoutSeconds,
	)
	if err != nil {
		return response.TaskResponse{}, err
	}

	return h.taskMapper.ToTaskResponse(task), nil
}

func (h *TaskHandler) Delete(ctx *gin.Context, id string) (string, error) {
	if err := h.taskService.DeleteTask(ctx, id); err != nil {
		return "", err
	}
	return "task deleted successfully", nil
}

func (h *TaskHandler) UpdateTask(ctx *gin.Context, id string, req UpdateTaskReq) (response.TaskResponse, error) {
	task, err := h.taskService.UpdateTask(
		ctx,
		id,
		req.Name,
		req.CronExpression,
		map[string]any(req.Parameters),
		entity.ExecutionMode(req.ExecutionMode),
		entity.LoadBalanceStrategy(req.LoadBalanceStrategy),
		req.MaxRetry,
		req.TimeoutSeconds,
		entity.TaskStatus(req.Status),
	)
	if err != nil {
		return response.TaskResponse{}, err
	}

	return h.taskMapper.ToTaskResponse(task), nil
}

func (h *TaskHandler) TriggerTask(ctx *gin.Context, id string, req TriggerTaskRequest) (response.TaskExecutionResponse, error) {
	execution, err := h.taskService.TriggerTask(ctx, id, req.Parameters)
	if err != nil {
		return response.TaskExecutionResponse{}, err
	}

	return h.execMapper.ToExecutionResponse(execution), nil
}

func (h *TaskHandler) Pause(ctx *gin.Context, id string) (string, error) {
	if err := h.taskService.PauseTask(ctx, id); err != nil {
		return "", err
	}
	return "task paused successfully", nil
}

func (h *TaskHandler) Resume(ctx *gin.Context, id string) (string, error) {
	if err := h.taskService.ResumeTask(ctx, id); err != nil {
		return "", err
	}
	return "task resumed successfully", nil
}

func (h *TaskHandler) GetTaskExecutors(ctx *gin.Context, id string) ([]response.TaskExecutorResponse, error) {
	taskExecutors, err := h.taskService.GetTaskExecutors(ctx, id)
	if err != nil {
		return nil, err
	}

	return h.taskMapper.ToTaskExecutorListResponse(taskExecutors), nil
}

func (h *TaskHandler) AssignExecutor(ctx *gin.Context, id string, req AssignExecutorReq) (response.TaskExecutorResponse, error) {
	taskExecutor, err := h.taskService.AssignExecutor(ctx, id, req.ExecutorID, req.Priority, req.Weight)
	if err != nil {
		return response.TaskExecutorResponse{}, err
	}

	return h.taskMapper.ToTaskExecutorResponse(taskExecutor), nil
}

func (h *TaskHandler) UpdateExecutorAssignment(ctx *gin.Context, id string, executorID string, req UpdateExecutorAssignmentReq) (response.TaskExecutorResponse, error) {
	taskExecutor, err := h.taskService.UpdateExecutorAssignment(ctx, id, executorID, req.Priority, req.Weight)
	if err != nil {
		return response.TaskExecutorResponse{}, err
	}

	return h.taskMapper.ToTaskExecutorResponse(taskExecutor), nil
}

func (h *TaskHandler) UnassignExecutor(ctx *gin.Context, id string, executorID string) (string, error) {
	if err := h.taskService.UnassignExecutor(ctx, id, executorID); err != nil {
		return "", err
	}
	return "executor unassigned", nil
}

func (h *TaskHandler) GetTaskStats(ctx *gin.Context, id string) (response.TaskStatsResponse, error) {
	stats, err := h.taskService.GetTaskStats(ctx, id)
	if err != nil {
		return response.TaskStatsResponse{}, err
	}

	return h.taskMapper.ToTaskStatsResponse(stats), nil
}
