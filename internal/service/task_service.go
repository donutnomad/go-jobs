package service

import (
	"context"
	"time"

	"github.com/jobs/scheduler/internal/domain/entity"
	domainError "github.com/jobs/scheduler/internal/domain/error"
	"github.com/jobs/scheduler/internal/domain/repository"
)

// ITaskService 任务服务接口
type ITaskService interface {
	// 任务管理
	CreateTask(ctx context.Context, name, cronExpression string, parameters map[string]any,
		executionMode entity.ExecutionMode, loadBalanceStrategy entity.LoadBalanceStrategy,
		maxRetry, timeoutSeconds int) (*entity.Task, error)
	GetTask(ctx context.Context, id string) (*entity.Task, error)
	GetTaskByName(ctx context.Context, name string) (*entity.Task, error)
	UpdateTask(ctx context.Context, id string, name, cronExpression string, parameters map[string]any,
		executionMode entity.ExecutionMode, loadBalanceStrategy entity.LoadBalanceStrategy,
		maxRetry, timeoutSeconds int, status entity.TaskStatus) (*entity.Task, error)
	DeleteTask(ctx context.Context, id string) error
	ListTasks(ctx context.Context, filter repository.TaskFilter) ([]*entity.Task, error)

	// 任务状态管理
	PauseTask(ctx context.Context, id string) error
	ResumeTask(ctx context.Context, id string) error

	// 执行器分配管理
	AssignExecutor(ctx context.Context, taskID, executorID string, priority, weight int) (*entity.TaskExecutor, error)
	UnassignExecutor(ctx context.Context, taskID, executorID string) error
	UpdateExecutorAssignment(ctx context.Context, taskID, executorID string, priority, weight int) (*entity.TaskExecutor, error)
	GetTaskExecutors(ctx context.Context, taskID string) ([]*entity.TaskExecutor, error)

	// 任务触发
	TriggerTask(ctx context.Context, taskID string, parameters map[string]any) (*entity.TaskExecution, error)

	// 任务统计
	GetTaskStats(ctx context.Context, taskID string) (*TaskStatsResult, error)
}

type TaskStatsResult struct {
	SuccessRate24h   float64
	Total24h         int64
	Success24h       int64
	Health90d        *repository.TaskHealthStats
	RecentExecutions []*repository.RecentExecutionStats
	DailyStats90d    []*repository.DailyStats
}

// IEmitter 事件发射器接口（用于与调度器通信）
type IEmitter interface {
	SubmitNewTask(taskID string, executionID string) error
	ReloadTasks() error
	CancelExecutionTimer(executionID string) error
}

type TaskService struct {
	taskRepo      repository.TaskRepository
	executorRepo  repository.ExecutorRepository
	executionRepo repository.ExecutionRepository
	emitter       IEmitter
}

// NewTaskService 创建任务服务
func NewTaskService(
	taskRepo repository.TaskRepository,
	executorRepo repository.ExecutorRepository,
	executionRepo repository.ExecutionRepository,
	emitter IEmitter,
) ITaskService {
	return &TaskService{
		taskRepo:      taskRepo,
		executorRepo:  executorRepo,
		executionRepo: executionRepo,
		emitter:       emitter,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, name, cronExpression string, parameters map[string]any,
	executionMode entity.ExecutionMode, loadBalanceStrategy entity.LoadBalanceStrategy,
	maxRetry, timeoutSeconds int) (*entity.Task, error) {

	// 检查任务名是否已存在
	if _, err := s.taskRepo.GetByName(ctx, name); err == nil {
		return nil, domainError.ErrTaskAlreadyExists
	}

	task, err := entity.NewTask(name, cronExpression)
	if err != nil {
		return nil, err
	}

	// 设置可选参数
	if parameters != nil {
		task.Parameters = parameters
	}
	if executionMode != "" {
		task.ExecutionMode = executionMode
	}
	if loadBalanceStrategy != "" {
		task.LoadBalanceStrategy = loadBalanceStrategy
	}
	if maxRetry > 0 {
		task.MaxRetry = maxRetry
	}
	if timeoutSeconds > 0 {
		task.TimeoutSeconds = timeoutSeconds
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*entity.Task, error) {
	return s.taskRepo.GetByID(ctx, id)
}

func (s *TaskService) GetTaskByName(ctx context.Context, name string) (*entity.Task, error) {
	return s.taskRepo.GetByName(ctx, name)
}

func (s *TaskService) UpdateTask(ctx context.Context, id string, name, cronExpression string, parameters map[string]any,
	executionMode entity.ExecutionMode, loadBalanceStrategy entity.LoadBalanceStrategy,
	maxRetry, timeoutSeconds int, status entity.TaskStatus) (*entity.Task, error) {

	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新任务信息
	task.Update(name, cronExpression, parameters, executionMode, loadBalanceStrategy, maxRetry, timeoutSeconds)

	// 更新状态
	if status != "" && status != task.Status {
		switch status {
		case entity.TaskStatusPaused:
			if err := task.Pause(); err != nil {
				return nil, err
			}
		case entity.TaskStatusActive:
			if err := task.Resume(); err != nil {
				return nil, err
			}
		case entity.TaskStatusDeleted:
			task.Delete()
		}
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	return s.taskRepo.Delete(ctx, id)
}

func (s *TaskService) ListTasks(ctx context.Context, filter repository.TaskFilter) ([]*entity.Task, error) {
	return s.taskRepo.List(ctx, filter)
}

func (s *TaskService) PauseTask(ctx context.Context, id string) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := task.Pause(); err != nil {
		return err
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return err
	}

	// 通知调度器重新加载任务
	if err := s.emitter.ReloadTasks(); err != nil {
		// 记录日志但不影响操作结果
	}

	return nil
}

func (s *TaskService) ResumeTask(ctx context.Context, id string) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := task.Resume(); err != nil {
		return err
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return err
	}

	// 通知调度器重新加载任务
	if err := s.emitter.ReloadTasks(); err != nil {
		// 记录日志但不影响操作结果
	}

	return nil
}

func (s *TaskService) AssignExecutor(ctx context.Context, taskID, executorID string, priority, weight int) (*entity.TaskExecutor, error) {
	// 验证任务是否存在
	if _, err := s.taskRepo.GetByID(ctx, taskID); err != nil {
		return nil, err
	}

	// 验证执行器是否存在，并获取执行器名称
	executor, err := s.executorRepo.GetByID(ctx, executorID)
	if err != nil {
		return nil, err
	}

	// 设置默认值
	if priority <= 0 {
		priority = 1
	}
	if weight <= 0 {
		weight = 1
	}

	return s.taskRepo.AssignExecutor(ctx, taskID, executor.Name, priority, weight)
}

func (s *TaskService) UnassignExecutor(ctx context.Context, taskID, executorID string) error {
	// 验证执行器是否存在，并获取执行器名称
	executor, err := s.executorRepo.GetByID(ctx, executorID)
	if err != nil {
		return err
	}

	return s.taskRepo.UnassignExecutor(ctx, taskID, executor.Name)
}

func (s *TaskService) UpdateExecutorAssignment(ctx context.Context, taskID, executorID string, priority, weight int) (*entity.TaskExecutor, error) {
	// 验证执行器是否存在，并获取执行器名称
	executor, err := s.executorRepo.GetByID(ctx, executorID)
	if err != nil {
		return nil, err
	}

	return s.taskRepo.UpdateAssignment(ctx, taskID, executor.Name, priority, weight)
}

func (s *TaskService) GetTaskExecutors(ctx context.Context, taskID string) ([]*entity.TaskExecutor, error) {
	return s.taskRepo.GetTaskExecutors(ctx, taskID)
}

func (s *TaskService) TriggerTask(ctx context.Context, taskID string, parameters map[string]any) (*entity.TaskExecution, error) {
	// 获取任务
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 检查任务是否可以执行
	if !task.CanBeExecuted() {
		return nil, domainError.NewBusinessError("TASK_NOT_EXECUTABLE", "任务不能执行", nil)
	}

	// 合并参数
	if parameters != nil {
		if task.Parameters == nil {
			task.Parameters = make(map[string]any)
		}
		for k, v := range parameters {
			task.Parameters[k] = v
		}
	}

	// 创建执行记录
	execution := entity.NewTaskExecution(task.ID)

	if err := s.executionRepo.Create(ctx, execution); err != nil {
		return nil, err
	}

	// 提交任务到调度器
	if err := s.emitter.SubmitNewTask(taskID, execution.ID); err != nil {
		// 如果提交失败，删除已创建的执行记录
		s.executionRepo.Delete(ctx, execution.ID)
		return nil, domainError.NewBusinessError("SUBMIT_TASK_FAILED", "提交任务失败", err)
	}

	return execution, nil
}

func (s *TaskService) GetTaskStats(ctx context.Context, taskID string) (*TaskStatsResult, error) {
	// 获取24小时数据
	since24h := time.Now().Add(-24 * time.Hour)
	stats24h, err := s.executionRepo.GetStats(ctx, repository.ExecutionStatsFilter{
		TaskID:    taskID,
		StartTime: &since24h,
	})
	if err != nil {
		return nil, err
	}

	successRate24h := float64(0)
	if stats24h.Total > 0 {
		successRate24h = float64(stats24h.Success) / float64(stats24h.Total) * 100
	}

	// 获取90天健康度统计
	health90d, err := s.executionRepo.GetTaskStats(ctx, taskID, 90)
	if err != nil {
		return nil, err
	}

	// 获取最近执行统计（7天）
	recentExecutions, err := s.executionRepo.GetRecentExecutions(ctx, taskID, 7)
	if err != nil {
		return nil, err
	}

	// 获取90天每日统计
	dailyStats90d, err := s.executionRepo.GetDailyStats(ctx, taskID, 90)
	if err != nil {
		return nil, err
	}

	return &TaskStatsResult{
		SuccessRate24h:   successRate24h,
		Total24h:         stats24h.Total,
		Success24h:       stats24h.Success,
		Health90d:        health90d,
		RecentExecutions: recentExecutions,
		DailyStats90d:    dailyStats90d,
	}, nil
}
