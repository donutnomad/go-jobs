package coordination

import (
	"context"
	"fmt"
	"time"

	executionbiz "github.com/jobs/scheduler/internal/app/biz/execution"
	executorbiz "github.com/jobs/scheduler/internal/app/biz/executor"
	schedulerbiz "github.com/jobs/scheduler/internal/app/biz/scheduler"
	taskbiz "github.com/jobs/scheduler/internal/app/biz/task"
	"github.com/jobs/scheduler/internal/app/infra/interfaces"
	"github.com/jobs/scheduler/internal/app/types"
)

// TaskSchedulingCoordinator 任务调度协调器
// 遵循架构指南的coord_[流程].go命名规范，协调跨领域的任务调度流程
type TaskSchedulingCoordinator struct {
	taskUC      *taskbiz.UseCase
	executorUC  *executorbiz.UseCase
	executionUC *executionbiz.UseCase
	schedulerUC *schedulerbiz.UseCase
	logger      interfaces.Logger
}

// NewTaskSchedulingCoordinator 创建任务调度协调器
func NewTaskSchedulingCoordinator(
	taskUC *taskbiz.UseCase,
	executorUC *executorbiz.UseCase,
	executionUC *executionbiz.UseCase,
	schedulerUC *schedulerbiz.UseCase,
	logger interfaces.Logger,
) *TaskSchedulingCoordinator {
	return &TaskSchedulingCoordinator{
		taskUC:      taskUC,
		executorUC:  executorUC,
		executionUC: executionUC,
		schedulerUC: schedulerUC,
		logger:      logger,
	}
}

// ScheduleTaskRequest 调度任务请求
type ScheduleTaskRequest struct {
	TaskID        types.ID      `json:"task_id"`
	ScheduledTime time.Time     `json:"scheduled_time"`
	Parameters    types.JSONMap `json:"parameters,omitempty"`
	ForceExecute  bool          `json:"force_execute"` // 是否强制执行，忽略并发限制
}

// ScheduleTaskResponse 调度任务响应
type ScheduleTaskResponse struct {
	ExecutionID   types.ID                     `json:"execution_id"`
	TaskID        types.ID                     `json:"task_id"`
	ExecutorID    *types.ID                    `json:"executor_id,omitempty"`
	Status        executionbiz.ExecutionStatus `json:"status"`
	ScheduledTime time.Time                    `json:"scheduled_time"`
	Reason        string                       `json:"reason,omitempty"`
}

// ScheduleTask 调度任务执行
// 这是核心的任务调度协调流程，涉及多个领域的协作
func (c *TaskSchedulingCoordinator) ScheduleTask(ctx context.Context, req *ScheduleTaskRequest) (*ScheduleTaskResponse, error) {
	c.logger.Info(ctx, "Starting task scheduling", map[string]interface{}{
		"task_id":        req.TaskID,
		"scheduled_time": req.ScheduledTime,
		"force_execute":  req.ForceExecute,
	})

	// 1. 获取任务信息并验证
	taskResp, err := c.taskUC.GetTask(ctx, &taskbiz.GetTaskRequest{
		TaskID: req.TaskID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	task := taskResp.Task

	// 检查任务是否可以被调度
	if !task.CanBeScheduled() {
		return &ScheduleTaskResponse{
			TaskID: req.TaskID,
			Status: executionbiz.ExecutionStatusSkipped,
			Reason: fmt.Sprintf("task cannot be scheduled in current status: %s", task.Status()),
		}, nil
	}

	// 2. 检查并发执行限制
	if !req.ForceExecute && !task.AllowsConcurrentExecution() {
		runningExecutions, err := c.executionUC.GetRunningExecutions(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check running executions: %w", err)
		}

		// 检查是否有同一任务正在运行
		for _, execution := range runningExecutions {
			if execution.TaskID() == req.TaskID {
				if task.ShouldSkipOnRunning() {
					c.logger.Info(ctx, "Skipping task due to concurrent execution", map[string]interface{}{
						"task_id":    req.TaskID,
						"running_id": execution.ID(),
					})

					return &ScheduleTaskResponse{
						TaskID: req.TaskID,
						Status: executionbiz.ExecutionStatusSkipped,
						Reason: "task is already running and concurrent execution is disabled",
					}, nil
				}
				// 对于串行模式，等待当前执行完成（这里简化处理，实际可能需要队列机制）
			}
		}
	}

	// 3. 创建任务执行
	createExecReq := &executionbiz.CreateExecutionRequest{
		TaskID:        req.TaskID,
		Parameters:    req.Parameters,
		ScheduledTime: req.ScheduledTime,
	}

	createExecResp, err := c.executionUC.CreateExecution(ctx, createExecReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	executionID := createExecResp.ExecutionID

	// 4. 选择执行器
	selectedExecutor, err := c.selectExecutor(ctx, task, executionID)
	if err != nil {
		// 如果没有可用执行器，标记为待执行
		c.logger.Warn(ctx, "No available executor found", map[string]interface{}{
			"task_id":      req.TaskID,
			"execution_id": executionID,
			"error":        err.Error(),
		})

		return &ScheduleTaskResponse{
			ExecutionID:   executionID,
			TaskID:        req.TaskID,
			Status:        executionbiz.ExecutionStatusPending,
			ScheduledTime: req.ScheduledTime,
			Reason:        "no available executor",
		}, nil
	}

	// 5. 启动执行
	startReq := &executionbiz.StartExecutionRequest{
		ExecutionID: executionID,
		ExecutorID:  selectedExecutor.ID(),
	}

	if err := c.executionUC.StartExecution(ctx, startReq); err != nil {
		c.logger.Error(ctx, "Failed to start execution", map[string]interface{}{
			"task_id":      req.TaskID,
			"execution_id": executionID,
			"executor_id":  selectedExecutor.ID(),
			"error":        err.Error(),
		})

		return nil, fmt.Errorf("failed to start execution: %w", err)
	}

	c.logger.Info(ctx, "Task scheduled successfully", map[string]interface{}{
		"task_id":      req.TaskID,
		"execution_id": executionID,
		"executor_id":  selectedExecutor.ID(),
	})

	return &ScheduleTaskResponse{
		ExecutionID:   executionID,
		TaskID:        req.TaskID,
		ExecutorID:    &selectedExecutor.ID(),
		Status:        executionbiz.ExecutionStatusRunning,
		ScheduledTime: req.ScheduledTime,
	}, nil
}

// selectExecutor 选择执行器
func (c *TaskSchedulingCoordinator) selectExecutor(ctx context.Context, task *taskbiz.Task, executionID types.ID) (*executorbiz.Executor, error) {
	// 获取可用执行器
	availableExecutors, err := c.executorUC.GetAvailableExecutors(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available executors: %w", err)
	}

	if len(availableExecutors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	// 根据负载均衡策略选择执行器
	strategy := task.LoadBalanceStrategy()

	switch strategy {
	case taskbiz.LoadBalanceRoundRobin:
		return c.selectByRoundRobin(availableExecutors), nil
	case taskbiz.LoadBalanceWeighted:
		return c.selectByWeight(availableExecutors), nil
	case taskbiz.LoadBalanceRandom:
		return c.selectByRandom(availableExecutors), nil
	case taskbiz.LoadBalanceLeastLoaded:
		return c.selectByLeastLoaded(ctx, availableExecutors)
	case taskbiz.LoadBalanceSticky:
		return c.selectBySticky(ctx, task.ID(), availableExecutors)
	default:
		// 默认使用轮询
		return c.selectByRoundRobin(availableExecutors), nil
	}
}

// selectByRoundRobin 轮询选择
func (c *TaskSchedulingCoordinator) selectByRoundRobin(executors []*executorbiz.Executor) *executorbiz.Executor {
	// 简化实现，实际应该维护轮询状态
	return executors[0]
}

// selectByWeight 加权选择
func (c *TaskSchedulingCoordinator) selectByWeight(executors []*executorbiz.Executor) *executorbiz.Executor {
	// 简化实现，实际应该根据执行器权重选择
	return executors[0]
}

// selectByRandom 随机选择
func (c *TaskSchedulingCoordinator) selectByRandom(executors []*executorbiz.Executor) *executorbiz.Executor {
	// 简化实现，实际应该随机选择
	return executors[0]
}

// selectByLeastLoaded 最少负载选择
func (c *TaskSchedulingCoordinator) selectByLeastLoaded(ctx context.Context, executors []*executorbiz.Executor) (*executorbiz.Executor, error) {
	// 简化实现，实际应该查询每个执行器的负载情况
	return executors[0], nil
}

// selectBySticky 粘性选择
func (c *TaskSchedulingCoordinator) selectBySticky(ctx context.Context, taskID types.ID, executors []*executorbiz.Executor) (*executorbiz.Executor, error) {
	// 简化实现，实际应该查询任务历史执行记录，优先选择之前使用的执行器
	return executors[0], nil
}

// BatchScheduleTasksRequest 批量调度任务请求
type BatchScheduleTasksRequest struct {
	Tasks []ScheduleTaskRequest `json:"tasks"`
}

// BatchScheduleTasksResponse 批量调度任务响应
type BatchScheduleTasksResponse struct {
	Results []ScheduleTaskResponse `json:"results"`
	Success int                    `json:"success"`
	Failed  int                    `json:"failed"`
}

// BatchScheduleTasks 批量调度任务
func (c *TaskSchedulingCoordinator) BatchScheduleTasks(ctx context.Context, req *BatchScheduleTasksRequest) (*BatchScheduleTasksResponse, error) {
	results := make([]ScheduleTaskResponse, len(req.Tasks))
	success := 0
	failed := 0

	for i, taskReq := range req.Tasks {
		result, err := c.ScheduleTask(ctx, &taskReq)
		if err != nil {
			results[i] = ScheduleTaskResponse{
				TaskID: taskReq.TaskID,
				Status: executionbiz.ExecutionStatusFailed,
				Reason: err.Error(),
			}
			failed++
		} else {
			results[i] = *result
			if result.Status == executionbiz.ExecutionStatusRunning {
				success++
			} else {
				failed++
			}
		}
	}

	return &BatchScheduleTasksResponse{
		Results: results,
		Success: success,
		Failed:  failed,
	}, nil
}

// ProcessPendingExecutionsRequest 处理待执行任务请求
type ProcessPendingExecutionsRequest struct {
	MaxExecutions int `json:"max_executions,omitempty"`
}

// ProcessPendingExecutionsResponse 处理待执行任务响应
type ProcessPendingExecutionsResponse struct {
	ProcessedCount int                    `json:"processed_count"`
	SuccessCount   int                    `json:"success_count"`
	FailedCount    int                    `json:"failed_count"`
	Results        []ScheduleTaskResponse `json:"results"`
}

// ProcessPendingExecutions 处理待执行的任务
func (c *TaskSchedulingCoordinator) ProcessPendingExecutions(ctx context.Context, req *ProcessPendingExecutionsRequest) (*ProcessPendingExecutionsResponse, error) {
	// 获取待执行的任务
	pendingExecutions, err := c.executionUC.GetPendingExecutions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending executions: %w", err)
	}

	maxExecutions := req.MaxExecutions
	if maxExecutions <= 0 {
		maxExecutions = len(pendingExecutions) // 处理所有待执行任务
	}

	if len(pendingExecutions) > maxExecutions {
		pendingExecutions = pendingExecutions[:maxExecutions]
	}

	results := make([]ScheduleTaskResponse, len(pendingExecutions))
	successCount := 0
	failedCount := 0

	for i, execution := range pendingExecutions {
		// 尝试为待执行任务分配执行器
		taskResp, err := c.taskUC.GetTask(ctx, &taskbiz.GetTaskRequest{
			TaskID: execution.TaskID(),
		})
		if err != nil {
			results[i] = ScheduleTaskResponse{
				ExecutionID: execution.ID(),
				TaskID:      execution.TaskID(),
				Status:      executionbiz.ExecutionStatusFailed,
				Reason:      fmt.Sprintf("failed to get task: %v", err),
			}
			failedCount++
			continue
		}

		selectedExecutor, err := c.selectExecutor(ctx, taskResp.Task, execution.ID())
		if err != nil {
			results[i] = ScheduleTaskResponse{
				ExecutionID: execution.ID(),
				TaskID:      execution.TaskID(),
				Status:      executionbiz.ExecutionStatusPending,
				Reason:      "no available executor",
			}
			failedCount++
			continue
		}

		// 启动执行
		startReq := &executionbiz.StartExecutionRequest{
			ExecutionID: execution.ID(),
			ExecutorID:  selectedExecutor.ID(),
		}

		if err := c.executionUC.StartExecution(ctx, startReq); err != nil {
			results[i] = ScheduleTaskResponse{
				ExecutionID: execution.ID(),
				TaskID:      execution.TaskID(),
				Status:      executionbiz.ExecutionStatusFailed,
				Reason:      fmt.Sprintf("failed to start execution: %v", err),
			}
			failedCount++
			continue
		}

		results[i] = ScheduleTaskResponse{
			ExecutionID:   execution.ID(),
			TaskID:        execution.TaskID(),
			ExecutorID:    &selectedExecutor.ID(),
			Status:        executionbiz.ExecutionStatusRunning,
			ScheduledTime: execution.GetScheduledTime(),
		}
		successCount++

		c.logger.Info(ctx, "Pending execution processed", map[string]interface{}{
			"execution_id": execution.ID(),
			"task_id":      execution.TaskID(),
			"executor_id":  selectedExecutor.ID(),
		})
	}

	return &ProcessPendingExecutionsResponse{
		ProcessedCount: len(pendingExecutions),
		SuccessCount:   successCount,
		FailedCount:    failedCount,
		Results:        results,
	}, nil
}

// CronSchedulingRequest Cron调度请求
type CronSchedulingRequest struct {
	CheckTime time.Time `json:"check_time"`
}

// CronSchedulingResponse Cron调度响应
type CronSchedulingResponse struct {
	ScheduledCount int                    `json:"scheduled_count"`
	SkippedCount   int                    `json:"skipped_count"`
	Results        []ScheduleTaskResponse `json:"results"`
}

// ProcessCronScheduling 处理Cron调度
// 检查所有活跃任务，看哪些需要在当前时间执行
func (c *TaskSchedulingCoordinator) ProcessCronScheduling(ctx context.Context, req *CronSchedulingRequest) (*CronSchedulingResponse, error) {
	checkTime := req.CheckTime
	if checkTime.IsZero() {
		checkTime = time.Now()
	}

	c.logger.Info(ctx, "Processing cron scheduling", map[string]interface{}{
		"check_time": checkTime,
	})

	// 获取所有活跃的可调度任务
	listReq := &taskbiz.ListTasksRequest{
		Filters: taskbiz.TaskFilters{
			Status: func() *taskbiz.TaskStatus {
				status := taskbiz.TaskStatusActive
				return &status
			}(),
		},
		Pagination: types.Pagination{
			Page:     1,
			PageSize: 1000, // 假设最多1000个活跃任务
		},
	}

	listResp, err := c.taskUC.ListTasks(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list active tasks: %w", err)
	}

	var results []ScheduleTaskResponse
	scheduledCount := 0
	skippedCount := 0

	for _, task := range listResp.Tasks {
		// 检查任务是否需要在这个时间执行
		nextExecTime, err := task.GetNextExecutionTime(checkTime.Add(-1 * time.Minute)) // 检查前一分钟到现在的时间窗口
		if err != nil {
			c.logger.Warn(ctx, "Failed to get next execution time", map[string]interface{}{
				"task_id": task.ID(),
				"error":   err.Error(),
			})
			continue
		}

		// 如果下次执行时间在检查时间窗口内，就需要调度
		if nextExecTime.Before(checkTime) || nextExecTime.Equal(checkTime) {
			scheduleReq := ScheduleTaskRequest{
				TaskID:        task.ID(),
				ScheduledTime: nextExecTime,
				Parameters:    task.Parameters(),
			}

			result, err := c.ScheduleTask(ctx, &scheduleReq)
			if err != nil {
				c.logger.Error(ctx, "Failed to schedule cron task", map[string]interface{}{
					"task_id": task.ID(),
					"error":   err.Error(),
				})
				results = append(results, ScheduleTaskResponse{
					TaskID: task.ID(),
					Status: executionbiz.ExecutionStatusFailed,
					Reason: err.Error(),
				})
				skippedCount++
			} else {
				results = append(results, *result)
				if result.Status == executionbiz.ExecutionStatusRunning {
					scheduledCount++
				} else {
					skippedCount++
				}
			}
		}
	}

	c.logger.Info(ctx, "Cron scheduling completed", map[string]interface{}{
		"scheduled_count": scheduledCount,
		"skipped_count":   skippedCount,
		"total_results":   len(results),
	})

	return &CronSchedulingResponse{
		ScheduledCount: scheduledCount,
		SkippedCount:   skippedCount,
		Results:        results,
	}, nil
}
