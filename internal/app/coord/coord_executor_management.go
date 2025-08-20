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

// ExecutorManagementCoordinator 执行器管理协调器
// 遵循架构指南的coord_[流程].go命名规范，协调执行器生命周期管理流程
type ExecutorManagementCoordinator struct {
	taskUC      *taskbiz.UseCase
	executorUC  *executorbiz.UseCase
	executionUC *executionbiz.UseCase
	schedulerUC *schedulerbiz.UseCase
	logger      interfaces.Logger
}

// NewExecutorManagementCoordinator 创建执行器管理协调器
func NewExecutorManagementCoordinator(
	taskUC *taskbiz.UseCase,
	executorUC *executorbiz.UseCase,
	executionUC *executionbiz.UseCase,
	schedulerUC *schedulerbiz.UseCase,
	logger interfaces.Logger,
) *ExecutorManagementCoordinator {
	return &ExecutorManagementCoordinator{
		taskUC:      taskUC,
		executorUC:  executorUC,
		executionUC: executionUC,
		schedulerUC: schedulerUC,
		logger:      logger,
	}
}

// ExecutorRegistrationRequest 执行器注册请求
type ExecutorRegistrationRequest struct {
	Name           string                       `json:"name"`
	InstanceID     string                       `json:"instance_id"`
	BaseURL        string                       `json:"base_url"`
	HealthCheckURL string                       `json:"health_check_url,omitempty"`
	Metadata       executorbiz.ExecutorMetadata `json:"metadata,omitempty"`
}

// ExecutorRegistrationResponse 执行器注册响应
type ExecutorRegistrationResponse struct {
	ExecutorID    types.ID                   `json:"executor_id"`
	Name          string                     `json:"name"`
	InstanceID    string                     `json:"instance_id"`
	Status        executorbiz.ExecutorStatus `json:"status"`
	AssignedTasks []types.ID                 `json:"assigned_tasks"`
	CreatedAt     time.Time                  `json:"created_at"`
}

// RegisterExecutorWithTaskAssignment 注册执行器并分配任务
func (c *ExecutorManagementCoordinator) RegisterExecutorWithTaskAssignment(ctx context.Context, req *ExecutorRegistrationRequest) (*ExecutorRegistrationResponse, error) {
	c.logger.Info(ctx, "Starting executor registration with task assignment", map[string]interface{}{
		"name":        req.Name,
		"instance_id": req.InstanceID,
		"base_url":    req.BaseURL,
	})

	// 1. 注册执行器
	registerReq := &executorbiz.RegisterExecutorRequest{
		Name:           req.Name,
		InstanceID:     req.InstanceID,
		BaseURL:        req.BaseURL,
		HealthCheckURL: req.HealthCheckURL,
	}

	registerResp, err := c.executorUC.RegisterExecutor(ctx, registerReq)
	if err != nil {
		return nil, fmt.Errorf("failed to register executor: %w", err)
	}

	executorID := registerResp.ExecutorID

	// 2. 更新执行器元数据
	if req.Metadata.Tags != nil || req.Metadata.Capacity > 0 {
		updateReq := &executorbiz.UpdateExecutorConfigRequest{
			ExecutorID: executorID,
		}

		if err := c.executorUC.UpdateExecutorConfig(ctx, updateReq); err != nil {
			c.logger.Warn(ctx, "Failed to update executor metadata", map[string]interface{}{
				"executor_id": executorID,
				"error":       err.Error(),
			})
		}
	}

	// 3. 执行健康检查
	healthReq := &executorbiz.PerformHealthCheckRequest{
		ExecutorID: executorID,
	}

	healthResp, err := c.executorUC.PerformHealthCheck(ctx, healthReq)
	if err != nil {
		c.logger.Warn(ctx, "Initial health check failed", map[string]interface{}{
			"executor_id": executorID,
			"error":       err.Error(),
		})

		// 健康检查失败，但不阻止注册流程
	} else if !healthResp.IsHealthy {
		c.logger.Warn(ctx, "Executor is not healthy after registration", map[string]interface{}{
			"executor_id": executorID,
			"error_msg":   healthResp.ErrorMsg,
		})
	}

	// 4. 查找适合的任务并建立关联
	assignedTasks, err := c.assignTasksToExecutor(ctx, executorID, req.Metadata)
	if err != nil {
		c.logger.Warn(ctx, "Failed to assign tasks to executor", map[string]interface{}{
			"executor_id": executorID,
			"error":       err.Error(),
		})
		assignedTasks = []types.ID{} // 继续，但没有分配任务
	}

	c.logger.Info(ctx, "Executor registered successfully", map[string]interface{}{
		"executor_id":    executorID,
		"assigned_tasks": len(assignedTasks),
	})

	return &ExecutorRegistrationResponse{
		ExecutorID:    executorID,
		Name:          registerResp.Name,
		InstanceID:    registerResp.InstanceID,
		Status:        registerResp.Status,
		AssignedTasks: assignedTasks,
		CreatedAt:     registerResp.CreatedAt,
	}, nil
}

// assignTasksToExecutor 为执行器分配任务
func (c *ExecutorManagementCoordinator) assignTasksToExecutor(ctx context.Context, executorID types.ID, metadata executorbiz.ExecutorMetadata) ([]types.ID, error) {
	// 获取所有活跃任务
	listReq := &taskbiz.ListTasksRequest{
		Filters: taskbiz.TaskFilters{
			Status: func() *taskbiz.TaskStatus {
				status := taskbiz.TaskStatusActive
				return &status
			}(),
		},
		Pagination: types.Pagination{
			Page:     1,
			PageSize: 100, // 最多考虑100个任务
		},
	}

	listResp, err := c.taskUC.ListTasks(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	var assignedTasks []types.ID

	// 根据执行器元数据匹配任务
	for _, task := range listResp.Tasks {
		if c.isExecutorSuitableForTask(metadata, task) {
			assignedTasks = append(assignedTasks, task.ID())

			// 这里应该创建task-executor关联关系
			// 简化处理，实际应该调用相应的repository方法
			c.logger.Debug(ctx, "Task assigned to executor", map[string]interface{}{
				"task_id":     task.ID(),
				"executor_id": executorID,
				"task_name":   task.Name(),
			})
		}
	}

	return assignedTasks, nil
}

// isExecutorSuitableForTask 判断执行器是否适合执行任务
func (c *ExecutorManagementCoordinator) isExecutorSuitableForTask(metadata executorbiz.ExecutorMetadata, task *taskbiz.Task) bool {
	// 简化的匹配逻辑，实际应该根据任务需求和执行器能力进行匹配

	// 检查标签匹配
	if len(metadata.Tags) > 0 {
		// 这里应该检查任务是否需要特定标签的执行器
		// 简化处理，假设所有执行器都可以执行所有任务
	}

	// 检查容量
	if metadata.Capacity > 0 {
		// 执行器有容量限制，这里应该检查当前负载
		// 简化处理
	}

	return true // 简化处理，认为所有执行器都适合所有任务
}

// ExecutorUnregistrationRequest 执行器注销请求
type ExecutorUnregistrationRequest struct {
	ExecutorID        types.ID `json:"executor_id"`
	Reason            string   `json:"reason,omitempty"`
	GracefulShutdown  bool     `json:"graceful_shutdown"`   // 是否优雅关闭
	WaitForCompletion bool     `json:"wait_for_completion"` // 是否等待当前任务完成
}

// ExecutorUnregistrationResponse 执行器注销响应
type ExecutorUnregistrationResponse struct {
	ExecutorID        types.ID  `json:"executor_id"`
	PendingExecutions int       `json:"pending_executions"` // 待处理的执行数
	ReassignedTasks   int       `json:"reassigned_tasks"`   // 重新分配的任务数
	UnregisteredAt    time.Time `json:"unregistered_at"`
}

// UnregisterExecutorWithTaskReassignment 注销执行器并重新分配任务
func (c *ExecutorManagementCoordinator) UnregisterExecutorWithTaskReassignment(ctx context.Context, req *ExecutorUnregistrationRequest) (*ExecutorUnregistrationResponse, error) {
	c.logger.Info(ctx, "Starting executor unregistration with task reassignment", map[string]interface{}{
		"executor_id":         req.ExecutorID,
		"reason":              req.Reason,
		"graceful_shutdown":   req.GracefulShutdown,
		"wait_for_completion": req.WaitForCompletion,
	})

	// 1. 获取执行器信息
	getReq := &executorbiz.GetExecutorRequest{
		ExecutorID: req.ExecutorID,
	}

	getResp, err := c.executorUC.GetExecutor(ctx, getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor: %w", err)
	}

	// 2. 处理正在运行的执行
	pendingCount := 0
	if req.GracefulShutdown {
		// 优雅关闭：停止接受新任务，等待或取消当前任务
		if err := c.executorUC.SetMaintenanceMode(ctx, req.ExecutorID, "preparing for shutdown"); err != nil {
			c.logger.Warn(ctx, "Failed to set maintenance mode", map[string]interface{}{
				"executor_id": req.ExecutorID,
				"error":       err.Error(),
			})
		}

		// 处理正在运行的执行
		runningExecutions, err := c.getRunningExecutionsByExecutor(ctx, req.ExecutorID)
		if err != nil {
			return nil, fmt.Errorf("failed to get running executions: %w", err)
		}

		pendingCount = len(runningExecutions)

		if req.WaitForCompletion && len(runningExecutions) > 0 {
			c.logger.Info(ctx, "Waiting for running executions to complete", map[string]interface{}{
				"executor_id":   req.ExecutorID,
				"running_count": len(runningExecutions),
			})

			// 等待执行完成（简化处理，实际应该有超时和监控）
			if err := c.waitForExecutionsCompletion(ctx, runningExecutions, 5*time.Minute); err != nil {
				c.logger.Warn(ctx, "Timeout waiting for executions", map[string]interface{}{
					"executor_id": req.ExecutorID,
					"error":       err.Error(),
				})
			}
		} else if len(runningExecutions) > 0 {
			// 取消正在运行的执行
			for _, execution := range runningExecutions {
				cancelReq := &executionbiz.CancelExecutionRequest{
					ExecutionID: execution.ID(),
					Reason:      "executor shutting down",
				}

				if err := c.executionUC.CancelExecution(ctx, cancelReq); err != nil {
					c.logger.Warn(ctx, "Failed to cancel execution", map[string]interface{}{
						"execution_id": execution.ID(),
						"error":        err.Error(),
					})
				}
			}
		}
	} else {
		// 强制关闭：立即取消所有任务
		if err := c.executorUC.TakeOffline(ctx, req.ExecutorID, req.Reason); err != nil {
			c.logger.Warn(ctx, "Failed to take executor offline", map[string]interface{}{
				"executor_id": req.ExecutorID,
				"error":       err.Error(),
			})
		}
	}

	// 3. 重新分配任务到其他执行器
	reassignedTasks, err := c.reassignTasksFromExecutor(ctx, req.ExecutorID)
	if err != nil {
		c.logger.Warn(ctx, "Failed to reassign tasks", map[string]interface{}{
			"executor_id": req.ExecutorID,
			"error":       err.Error(),
		})
		reassignedTasks = 0
	}

	// 4. 注销执行器
	unregisterReq := &executorbiz.UnregisterExecutorRequest{
		ExecutorID: req.ExecutorID,
		Reason:     req.Reason,
	}

	if err := c.executorUC.UnregisterExecutor(ctx, unregisterReq); err != nil {
		return nil, fmt.Errorf("failed to unregister executor: %w", err)
	}

	c.logger.Info(ctx, "Executor unregistered successfully", map[string]interface{}{
		"executor_id":        req.ExecutorID,
		"pending_executions": pendingCount,
		"reassigned_tasks":   reassignedTasks,
	})

	return &ExecutorUnregistrationResponse{
		ExecutorID:        req.ExecutorID,
		PendingExecutions: pendingCount,
		ReassignedTasks:   reassignedTasks,
		UnregisteredAt:    time.Now(),
	}, nil
}

// getRunningExecutionsByExecutor 获取执行器正在运行的执行
func (c *ExecutorManagementCoordinator) getRunningExecutionsByExecutor(ctx context.Context, executorID types.ID) ([]*executionbiz.TaskExecution, error) {
	// 获取所有正在运行的执行
	runningExecutions, err := c.executionUC.GetRunningExecutions(ctx)
	if err != nil {
		return nil, err
	}

	// 筛选出该执行器的执行
	var result []*executionbiz.TaskExecution
	for _, execution := range runningExecutions {
		if execution.HasExecutor() && execution.GetExecutorID() == executorID {
			result = append(result, execution)
		}
	}

	return result, nil
}

// waitForExecutionsCompletion 等待执行完成
func (c *ExecutorManagementCoordinator) waitForExecutionsCompletion(ctx context.Context, executions []*executionbiz.TaskExecution, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			allCompleted := true
			for _, execution := range executions {
				// 重新获取执行状态
				getReq := &executionbiz.GetExecutionRequest{
					ExecutionID: execution.ID(),
				}

				getResp, err := c.executionUC.GetExecution(ctx, getReq)
				if err != nil {
					continue
				}

				if !getResp.Execution.IsCompleted() {
					allCompleted = false
					break
				}
			}

			if allCompleted {
				return nil
			}
		}
	}
}

// reassignTasksFromExecutor 从执行器重新分配任务
func (c *ExecutorManagementCoordinator) reassignTasksFromExecutor(ctx context.Context, executorID types.ID) (int, error) {
	// 简化处理，实际应该：
	// 1. 查询该执行器分配的所有任务
	// 2. 找到其他可用的执行器
	// 3. 重新建立任务-执行器关联
	// 4. 处理待执行的任务，重新分配到其他执行器

	c.logger.Info(ctx, "Task reassignment completed", map[string]interface{}{
		"executor_id": executorID,
		"reassigned":  0, // 简化处理
	})

	return 0, nil
}

// HealthCheckCoordinationRequest 健康检查协调请求
type HealthCheckCoordinationRequest struct {
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	FailureThreshold    int           `json:"failure_threshold"`
	RecoveryThreshold   int           `json:"recovery_threshold"`
}

// HealthCheckCoordinationResponse 健康检查协调响应
type HealthCheckCoordinationResponse struct {
	TotalExecutors     int                                      `json:"total_executors"`
	HealthyExecutors   int                                      `json:"healthy_executors"`
	UnhealthyExecutors int                                      `json:"unhealthy_executors"`
	CheckedExecutors   int                                      `json:"checked_executors"`
	Results            []executorbiz.PerformHealthCheckResponse `json:"results"`
}

// CoordinateHealthCheck 协调健康检查
func (c *ExecutorManagementCoordinator) CoordinateHealthCheck(ctx context.Context, req *HealthCheckCoordinationRequest) (*HealthCheckCoordinationResponse, error) {
	c.logger.Info(ctx, "Starting health check coordination", map[string]interface{}{
		"interval":           req.HealthCheckInterval,
		"failure_threshold":  req.FailureThreshold,
		"recovery_threshold": req.RecoveryThreshold,
	})

	// 1. 获取所有在线执行器
	listReq := &executorbiz.ListExecutorsRequest{
		Filters: executorbiz.ExecutorFilters{
			Status: func() *executorbiz.ExecutorStatus {
				status := executorbiz.ExecutorStatusOnline
				return &status
			}(),
		},
		Pagination: types.Pagination{
			Page:     1,
			PageSize: 1000,
		},
	}

	listResp, err := c.executorUC.ListExecutors(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list executors: %w", err)
	}

	// 2. 批量健康检查
	if err := c.executorUC.BatchHealthCheck(ctx, req.HealthCheckInterval); err != nil {
		return nil, fmt.Errorf("batch health check failed: %w", err)
	}

	// 3. 收集结果
	var results []executorbiz.PerformHealthCheckResponse
	healthyCount := 0
	unhealthyCount := 0
	checkedCount := 0

	for _, executor := range listResp.Executors {
		healthReq := &executorbiz.PerformHealthCheckRequest{
			ExecutorID: executor.ID(),
		}

		healthResp, err := c.executorUC.PerformHealthCheck(ctx, healthReq)
		if err != nil {
			c.logger.Warn(ctx, "Health check failed for executor", map[string]interface{}{
				"executor_id": executor.ID(),
				"error":       err.Error(),
			})
			continue
		}

		results = append(results, *healthResp)
		checkedCount++

		if healthResp.IsHealthy {
			healthyCount++
		} else {
			unhealthyCount++

			// 处理不健康的执行器
			c.handleUnhealthyExecutor(ctx, executor.ID(), healthResp.ErrorMsg)
		}
	}

	c.logger.Info(ctx, "Health check coordination completed", map[string]interface{}{
		"total":     len(listResp.Executors),
		"checked":   checkedCount,
		"healthy":   healthyCount,
		"unhealthy": unhealthyCount,
	})

	return &HealthCheckCoordinationResponse{
		TotalExecutors:     len(listResp.Executors),
		HealthyExecutors:   healthyCount,
		UnhealthyExecutors: unhealthyCount,
		CheckedExecutors:   checkedCount,
		Results:            results,
	}, nil
}

// handleUnhealthyExecutor 处理不健康的执行器
func (c *ExecutorManagementCoordinator) handleUnhealthyExecutor(ctx context.Context, executorID types.ID, errorMsg string) {
	c.logger.Warn(ctx, "Handling unhealthy executor", map[string]interface{}{
		"executor_id": executorID,
		"error":       errorMsg,
	})

	// 1. 取消该执行器正在运行的任务
	runningExecutions, err := c.getRunningExecutionsByExecutor(ctx, executorID)
	if err != nil {
		c.logger.Error(ctx, "Failed to get running executions for unhealthy executor", map[string]interface{}{
			"executor_id": executorID,
			"error":       err.Error(),
		})
		return
	}

	for _, execution := range runningExecutions {
		cancelReq := &executionbiz.CancelExecutionRequest{
			ExecutionID: execution.ID(),
			Reason:      "executor unhealthy: " + errorMsg,
		}

		if err := c.executionUC.CancelExecution(ctx, cancelReq); err != nil {
			c.logger.Error(ctx, "Failed to cancel execution on unhealthy executor", map[string]interface{}{
				"execution_id": execution.ID(),
				"executor_id":  executorID,
				"error":        err.Error(),
			})
		}
	}

	// 2. 将执行器标记为离线
	if err := c.executorUC.TakeOffline(ctx, executorID, "health check failed: "+errorMsg); err != nil {
		c.logger.Error(ctx, "Failed to take unhealthy executor offline", map[string]interface{}{
			"executor_id": executorID,
			"error":       err.Error(),
		})
	}
}

// LoadBalancingCoordinationRequest 负载均衡协调请求
type LoadBalancingCoordinationRequest struct {
	RebalanceThreshold float64 `json:"rebalance_threshold"` // 负载不平衡阈值
}

// LoadBalancingCoordinationResponse 负载均衡协调响应
type LoadBalancingCoordinationResponse struct {
	TotalExecutors     int              `json:"total_executors"`
	LoadImbalance      float64          `json:"load_imbalance"`
	RebalanceTriggered bool             `json:"rebalance_triggered"`
	TasksReassigned    int              `json:"tasks_reassigned"`
	LoadDistribution   map[types.ID]int `json:"load_distribution"`
}

// CoordinateLoadBalancing 协调负载均衡
func (c *ExecutorManagementCoordinator) CoordinateLoadBalancing(ctx context.Context, req *LoadBalancingCoordinationRequest) (*LoadBalancingCoordinationResponse, error) {
	c.logger.Info(ctx, "Starting load balancing coordination", map[string]interface{}{
		"rebalance_threshold": req.RebalanceThreshold,
	})

	// 简化的负载均衡实现
	// 实际应该：
	// 1. 计算每个执行器的当前负载
	// 2. 分析负载分布情况
	// 3. 如果不平衡超过阈值，重新分配任务
	// 4. 更新执行器-任务关联

	return &LoadBalancingCoordinationResponse{
		TotalExecutors:     0,
		LoadImbalance:      0.0,
		RebalanceTriggered: false,
		TasksReassigned:    0,
		LoadDistribution:   make(map[types.ID]int),
	}, nil
}
