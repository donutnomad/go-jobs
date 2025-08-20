package service

import (
	"context"

	"github.com/jobs/scheduler/internal/app/biz/execution"
	"github.com/jobs/scheduler/internal/app/biz/executionview"
	"github.com/jobs/scheduler/internal/app/biz/executor"
	"github.com/jobs/scheduler/internal/app/biz/scheduler"
	"github.com/jobs/scheduler/internal/app/biz/shared"
	"github.com/jobs/scheduler/internal/app/biz/task"
	"github.com/jobs/scheduler/internal/app/biz/taskview"
)

// TaskService 任务服务
type TaskService struct {
	taskUseCase *task.UseCase
	taskQuery   taskview.IQueryService
}

// NewTaskService 创建任务服务
func NewTaskService(taskUseCase *task.UseCase, taskQuery taskview.IQueryService) *TaskService {
	return &TaskService{
		taskUseCase: taskUseCase,
		taskQuery:   taskQuery,
	}
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(ctx context.Context, req task.CreateTaskRequest) (*task.Task, error) {
	return s.taskUseCase.CreateTask(ctx, req)
}

// GetTask 获取任务
func (s *TaskService) GetTask(ctx context.Context, id shared.ID) (*task.Task, error) {
	return s.taskUseCase.GetTask(ctx, id)
}

// ListTasks 列出任务
func (s *TaskService) ListTasks(ctx context.Context, filter taskview.TaskFilter, page, pageSize int) (*taskview.TaskListView, error) {
	return s.taskQuery.ListTasks(ctx, filter, page, pageSize)
}

// GetTaskDetail 获取任务详情
func (s *TaskService) GetTaskDetail(ctx context.Context, id shared.ID) (*taskview.TaskDetailView, error) {
	return s.taskQuery.GetTaskDetail(ctx, id)
}

// UpdateTask 更新任务
func (s *TaskService) UpdateTask(ctx context.Context, id shared.ID, req task.UpdateTaskRequest) (*task.Task, error) {
	return s.taskUseCase.UpdateTask(ctx, id, req)
}

// PauseTask 暂停任务
func (s *TaskService) PauseTask(ctx context.Context, id shared.ID) error {
	return s.taskUseCase.PauseTask(ctx, id)
}

// ResumeTask 恢复任务
func (s *TaskService) ResumeTask(ctx context.Context, id shared.ID) error {
	return s.taskUseCase.ResumeTask(ctx, id)
}

// DeleteTask 删除任务
func (s *TaskService) DeleteTask(ctx context.Context, id shared.ID) error {
	return s.taskUseCase.DeleteTask(ctx, id)
}

// ExecutorService 执行器服务
type ExecutorService struct {
	executorUseCase *executor.UseCase
}

// NewExecutorService 创建执行器服务
func NewExecutorService(executorUseCase *executor.UseCase) *ExecutorService {
	return &ExecutorService{
		executorUseCase: executorUseCase,
	}
}

// RegisterExecutor 注册执行器
func (s *ExecutorService) RegisterExecutor(ctx context.Context, req executor.RegisterExecutorRequest) (*executor.Executor, error) {
	return s.executorUseCase.RegisterExecutor(ctx, req)
}

// ExecutionService 执行服务
type ExecutionService struct {
	executionUseCase *execution.UseCase
	ExecutionQuery   executionview.IQueryService
}

// NewExecutionService 创建执行服务
func NewExecutionService(executionUseCase *execution.UseCase, executionQuery executionview.IQueryService) *ExecutionService {
	return &ExecutionService{
		executionUseCase: executionUseCase,
		ExecutionQuery:   executionQuery,
	}
}

// TriggerExecution 触发执行
func (s *ExecutionService) TriggerExecution(ctx context.Context, req execution.TriggerExecutionRequest) (*execution.TaskExecution, error) {
	return s.executionUseCase.TriggerExecution(ctx, req)
}

// SchedulerService 调度器服务
type SchedulerService struct {
	schedulerUseCase *scheduler.UseCase
}

// NewSchedulerService 创建调度器服务
func NewSchedulerService(schedulerUseCase *scheduler.UseCase) *SchedulerService {
	return &SchedulerService{
		schedulerUseCase: schedulerUseCase,
	}
}

// StartScheduler 启动调度器
func (s *SchedulerService) StartScheduler(ctx context.Context, instanceID, host string, port int, config *scheduler.Config) (*scheduler.Instance, error) {
	return s.schedulerUseCase.StartScheduler(ctx, instanceID, host, port, config)
}
