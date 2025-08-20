package task

import (
	"context"
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/app/infra/interfaces"
	"github.com/jobs/scheduler/internal/app/types"
)

// UseCase Task业务用例
// 遵循架构指南的usecase_[功能].go命名规范，组织业务功能
type UseCase struct {
	taskRepo       Repository
	queryService   QueryService
	eventPublisher interfaces.EventPublisher
	txManager      interfaces.TransactionManager
}

// NewUseCase 创建Task UseCase
func NewUseCase(
	taskRepo Repository,
	queryService QueryService,
	eventPublisher interfaces.EventPublisher,
	txManager interfaces.TransactionManager,
) *UseCase {
	return &UseCase{
		taskRepo:       taskRepo,
		queryService:   queryService,
		eventPublisher: eventPublisher,
		txManager:      txManager,
	}
}

// 任务创建相关

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name                string              `json:"name" validate:"required,min=1,max=100"`
	CronExpression      string              `json:"cron_expression" validate:"required"`
	Parameters          types.JSONMap       `json:"parameters,omitempty"`
	ExecutionMode       ExecutionMode       `json:"execution_mode,omitempty"`
	LoadBalanceStrategy LoadBalanceStrategy `json:"load_balance_strategy,omitempty"`
	MaxRetry            int                 `json:"max_retry,omitempty"`
	TimeoutSeconds      int                 `json:"timeout_seconds,omitempty"`
}

// Validate 验证请求
func (r *CreateTaskRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("task name too long")
	}
	if r.CronExpression == "" {
		return fmt.Errorf("cron expression is required")
	}

	// 验证执行模式
	if r.ExecutionMode != "" && !r.ExecutionMode.IsValid() {
		return fmt.Errorf("invalid execution mode: %s", r.ExecutionMode)
	}

	// 验证负载均衡策略
	if r.LoadBalanceStrategy != "" && !r.LoadBalanceStrategy.IsValid() {
		return fmt.Errorf("invalid load balance strategy: %s", r.LoadBalanceStrategy)
	}

	// 验证配置值
	if r.MaxRetry < 0 {
		return fmt.Errorf("max retry cannot be negative")
	}
	if r.TimeoutSeconds < 0 {
		return fmt.Errorf("timeout seconds cannot be negative")
	}

	return nil
}

// CreateTaskResponse 创建任务响应
type CreateTaskResponse struct {
	TaskID    types.ID   `json:"task_id"`
	Name      string     `json:"name"`
	Status    TaskStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateTask 创建任务
func (uc *UseCase) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 检查任务名称是否已存在
	exists, err := uc.taskRepo.ExistsByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check task existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("task with name '%s' already exists", req.Name)
	}

	// 在事务中执行
	var task *Task
	err = uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 创建任务实体
		task, err = NewTask(req.Name, req.CronExpression)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		// 设置可选参数
		if req.Parameters != nil {
			if err := task.UpdateParameters(req.Parameters); err != nil {
				return fmt.Errorf("failed to set parameters: %w", err)
			}
		}

		if req.ExecutionMode != "" {
			if err := task.UpdateExecutionMode(req.ExecutionMode); err != nil {
				return fmt.Errorf("failed to set execution mode: %w", err)
			}
		}

		if req.LoadBalanceStrategy != "" {
			if err := task.UpdateLoadBalanceStrategy(req.LoadBalanceStrategy); err != nil {
				return fmt.Errorf("failed to set load balance strategy: %w", err)
			}
		}

		if req.MaxRetry > 0 || req.TimeoutSeconds > 0 {
			maxRetry := 3         // 默认值
			timeoutSeconds := 300 // 默认值

			if req.MaxRetry > 0 {
				maxRetry = req.MaxRetry
			}
			if req.TimeoutSeconds > 0 {
				timeoutSeconds = req.TimeoutSeconds
			}

			config := NewTaskConfiguration(maxRetry, timeoutSeconds)
			if err := task.UpdateConfiguration(config); err != nil {
				return fmt.Errorf("failed to set configuration: %w", err)
			}
		}

		// 保存到仓储
		if err := uc.taskRepo.Save(ctx, task); err != nil {
			return fmt.Errorf("failed to save task: %w", err)
		}

		// 发布领域事件
		events := task.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		// 清除事件
		task.ClearDomainEvents()

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &CreateTaskResponse{
		TaskID:    task.ID(),
		Name:      task.Name(),
		Status:    task.Status(),
		CreatedAt: task.CreatedAt(),
	}, nil
}

// 任务查询相关

// GetTaskRequest 获取任务请求
type GetTaskRequest struct {
	TaskID types.ID `json:"task_id" validate:"required"`
}

// GetTaskResponse 获取任务响应
type GetTaskResponse struct {
	Task              *Task      `json:"task"`
	ExecutorCount     int        `json:"executor_count"`
	LastExecutionTime *time.Time `json:"last_execution_time,omitempty"`
	NextExecutionTime *time.Time `json:"next_execution_time,omitempty"`
}

// GetTask 获取任务详情
func (uc *UseCase) GetTask(ctx context.Context, req *GetTaskRequest) (*GetTaskResponse, error) {
	if req.TaskID.IsZero() {
		return nil, fmt.Errorf("task ID is required")
	}

	// 获取任务概览
	overview, err := uc.queryService.GetTaskOverview(ctx, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task overview: %w", err)
	}

	return &GetTaskResponse{
		Task:              overview.Task,
		ExecutorCount:     overview.ExecutorCount,
		LastExecutionTime: overview.LastExecutionTime,
		NextExecutionTime: overview.NextExecutionTime,
	}, nil
}

// ListTasksRequest 列表任务请求
type ListTasksRequest struct {
	Filters    TaskFilters      `json:"filters"`
	Pagination types.Pagination `json:"pagination"`
}

// ListTasksResponse 列表任务响应
type ListTasksResponse struct {
	Tasks      []*Task          `json:"tasks"`
	Pagination types.Pagination `json:"pagination"`
	Total      int64            `json:"total"`
}

// ListTasks 获取任务列表
func (uc *UseCase) ListTasks(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error) {
	// 获取任务列表
	tasks, err := uc.taskRepo.FindByFilters(ctx, req.Filters, req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// 获取总数
	total, err := uc.taskRepo.Count(ctx, req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	return &ListTasksResponse{
		Tasks:      tasks,
		Pagination: req.Pagination,
		Total:      total,
	}, nil
}

// 任务更新相关

// UpdateTask 更新任务
func (uc *UseCase) UpdateTask(ctx context.Context, taskID types.ID, req *UpdateTaskRequest) error {
	if taskID.IsZero() {
		return fmt.Errorf("task ID is required")
	}

	// 验证请求
	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// 在事务中执行
	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 获取任务
		task, err := uc.taskRepo.FindByID(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to find task: %w", err)
		}

		// 检查名称唯一性
		if req.Name != "" && req.Name != task.Name() {
			exists, err := uc.taskRepo.ExistsByName(ctx, req.Name)
			if err != nil {
				return fmt.Errorf("failed to check name existence: %w", err)
			}
			if exists {
				return fmt.Errorf("task with name '%s' already exists", req.Name)
			}
		}

		// 批量更新
		if err := task.UpdateAll(*req); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		// 保存更新
		if err := uc.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to save task: %w", err)
		}

		// 发布领域事件
		events := task.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		// 清除事件
		task.ClearDomainEvents()

		return nil
	})
}

// 任务控制相关

// PauseTask 暂停任务
func (uc *UseCase) PauseTask(ctx context.Context, taskID types.ID) error {
	if taskID.IsZero() {
		return fmt.Errorf("task ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		task, err := uc.taskRepo.FindByID(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to find task: %w", err)
		}

		if err := task.Pause(); err != nil {
			return fmt.Errorf("failed to pause task: %w", err)
		}

		if err := uc.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to save task: %w", err)
		}

		// 发布领域事件
		events := task.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		task.ClearDomainEvents()
		return nil
	})
}

// ResumeTask 恢复任务
func (uc *UseCase) ResumeTask(ctx context.Context, taskID types.ID) error {
	if taskID.IsZero() {
		return fmt.Errorf("task ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		task, err := uc.taskRepo.FindByID(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to find task: %w", err)
		}

		if err := task.Resume(); err != nil {
			return fmt.Errorf("failed to resume task: %w", err)
		}

		if err := uc.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to save task: %w", err)
		}

		// 发布领域事件
		events := task.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		task.ClearDomainEvents()
		return nil
	})
}

// DeleteTask 删除任务
func (uc *UseCase) DeleteTask(ctx context.Context, taskID types.ID) error {
	if taskID.IsZero() {
		return fmt.Errorf("task ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		task, err := uc.taskRepo.FindByID(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to find task: %w", err)
		}

		if err := task.Delete(); err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}

		if err := uc.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to save task: %w", err)
		}

		// 发布领域事件
		events := task.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		task.ClearDomainEvents()
		return nil
	})
}

// 任务手动触发

// TriggerTaskRequest 手动触发任务请求
type TriggerTaskRequest struct {
	TaskID     types.ID      `json:"task_id" validate:"required"`
	Parameters types.JSONMap `json:"parameters,omitempty"`
}

// TriggerTaskResponse 手动触发任务响应
type TriggerTaskResponse struct {
	ExecutionID   types.ID  `json:"execution_id"`
	TaskID        types.ID  `json:"task_id"`
	ScheduledTime time.Time `json:"scheduled_time"`
}

// TriggerTask 手动触发任务执行
func (uc *UseCase) TriggerTask(ctx context.Context, req *TriggerTaskRequest) (*TriggerTaskResponse, error) {
	if req.TaskID.IsZero() {
		return nil, fmt.Errorf("task ID is required")
	}

	var executionID types.ID
	scheduledTime := time.Now()

	err := uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 获取任务
		task, err := uc.taskRepo.FindByID(ctx, req.TaskID)
		if err != nil {
			return fmt.Errorf("failed to find task: %w", err)
		}

		// 检查任务是否可执行
		if !task.CanBeExecuted() {
			return fmt.Errorf("task cannot be executed in current status: %s", task.Status())
		}

		// 生成执行ID（这里简化处理，实际应该通过执行服务创建）
		executionID = types.ID("manual-trigger-" + task.ID().String() + "-" + fmt.Sprintf("%d", scheduledTime.Unix()))

		// 添加调度事件
		task.AddDomainEvent(TaskScheduledEvent{
			TaskID:        task.ID(),
			ExecutionID:   executionID,
			ScheduledTime: scheduledTime,
		})

		// 发布领域事件
		events := task.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		task.ClearDomainEvents()
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &TriggerTaskResponse{
		ExecutionID:   executionID,
		TaskID:        req.TaskID,
		ScheduledTime: scheduledTime,
	}, nil
}

// 统计和分析相关

// GetTaskStatisticsRequest 获取任务统计请求
type GetTaskStatisticsRequest struct {
	TimeRange TimeRange `json:"time_range"`
}

// GetTaskStatistics 获取任务统计
func (uc *UseCase) GetTaskStatistics(ctx context.Context, req *GetTaskStatisticsRequest) (*TaskStatistics, error) {
	return uc.queryService.GetTaskStatistics(ctx, req.TimeRange)
}

// SearchTasksRequest 搜索任务请求
type SearchTasksRequest struct {
	Query      string           `json:"query" validate:"required"`
	Pagination types.Pagination `json:"pagination"`
}

// SearchTasksResponse 搜索任务响应
type SearchTasksResponse struct {
	Tasks      []*Task          `json:"tasks"`
	Pagination types.Pagination `json:"pagination"`
	Total      int64            `json:"total"`
}

// SearchTasks 搜索任务
func (uc *UseCase) SearchTasks(ctx context.Context, req *SearchTasksRequest) (*SearchTasksResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	tasks, err := uc.queryService.SearchTasks(ctx, req.Query, req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks: %w", err)
	}

	// 简化处理，实际应该获取搜索结果总数
	total := int64(len(tasks))

	return &SearchTasksResponse{
		Tasks:      tasks,
		Pagination: req.Pagination,
		Total:      total,
	}, nil
}
