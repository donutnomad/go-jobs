package service

import (
	"context"
	"fmt"

	"github.com/jobs/scheduler/internal/domain/entity"
	domainError "github.com/jobs/scheduler/internal/domain/error"
	"github.com/jobs/scheduler/internal/domain/repository"
	"go.uber.org/zap"
)

// IExecutorService 执行器服务接口
type IExecutorService interface {
	// 执行器管理
	RegisterExecutor(ctx context.Context, req *RegisterExecutorRequest) (*entity.Executor, error)
	GetExecutor(ctx context.Context, id string) (*entity.Executor, error)
	GetExecutorByName(ctx context.Context, name string) (*entity.Executor, error)
	UpdateExecutor(ctx context.Context, id, name, baseURL, healthCheckURL string) (*entity.Executor, error)
	UpdateExecutorStatus(ctx context.Context, id string, status entity.ExecutorStatus) error
	DeleteExecutor(ctx context.Context, id string) error
	ListExecutors(ctx context.Context, filter repository.ExecutorFilter) ([]*entity.Executor, error)

	// 健康检查
	UpdateExecutorHealth(ctx context.Context, id string, isHealthy bool) error
}

// RegisterExecutorRequest 注册执行器请求
type RegisterExecutorRequest struct {
	ExecutorID     string
	ExecutorName   string
	ExecutorURL    string
	HealthCheckURL string
	Tasks          []TaskDefinitionRequest
	Metadata       map[string]any
}

// TaskDefinitionRequest 任务定义请求
type TaskDefinitionRequest struct {
	Name                string
	ExecutionMode       entity.ExecutionMode
	CronExpression      string
	LoadBalanceStrategy entity.LoadBalanceStrategy
	MaxRetry            int
	TimeoutSeconds      int
	Parameters          map[string]any
	Status              entity.TaskStatus
}

type ExecutorService struct {
	executorRepo repository.ExecutorRepository
	taskRepo     repository.TaskRepository
	logger       *zap.Logger
}

// NewExecutorService 创建执行器服务
func NewExecutorService(
	executorRepo repository.ExecutorRepository,
	taskRepo repository.TaskRepository,
	logger *zap.Logger,
) IExecutorService {
	return &ExecutorService{
		executorRepo: executorRepo,
		taskRepo:     taskRepo,
		logger:       logger,
	}
}

func (s *ExecutorService) RegisterExecutor(ctx context.Context, req *RegisterExecutorRequest) (*entity.Executor, error) {
	// 检查是否存在使用相同 instance_id 的执行器
	existingExecutor, err := s.executorRepo.GetByInstanceID(ctx, req.ExecutorID)

	if err == nil {
		// 执行器已存在
		if existingExecutor.Status == entity.ExecutorStatusOnline {
			// 检查是否是同一个执行器重新注册（通过 BaseURL 判断）
			if existingExecutor.BaseURL != req.ExecutorURL {
				return nil, fmt.Errorf("执行器实例ID %s 已在其他位置在线 (当前: %s, 新: %s)",
					req.ExecutorID, existingExecutor.BaseURL, req.ExecutorURL)
			}
		}

		// 更新现有执行器信息
		existingExecutor.Update(req.ExecutorName, req.ExecutorURL, req.HealthCheckURL)
		existingExecutor.UpdateStatus(entity.ExecutorStatusOnline)
		existingExecutor.SetHealthy(true)

		if err := s.executorRepo.Update(ctx, existingExecutor); err != nil {
			return nil, err
		}

		executor := existingExecutor

		// 注册任务
		if len(req.Tasks) > 0 {
			if err := s.registerTasks(ctx, executor.Name, req.Tasks); err != nil {
				s.logger.Error("注册任务失败",
					zap.String("executor_name", executor.Name),
					zap.Error(err))
			}
		}

		s.logger.Info("执行器更新注册成功",
			zap.String("executor_id", executor.ID),
			zap.String("instance_id", executor.InstanceID),
			zap.String("name", executor.Name),
			zap.Int("tasks_count", len(req.Tasks)))

		return executor, nil
	} else if err.Error() != domainError.ErrExecutorNotFound.Error() {
		return nil, err
	}

	// 创建新执行器
	executor, err := entity.NewExecutor(req.ExecutorName, req.ExecutorID, req.ExecutorURL, req.HealthCheckURL, req.Metadata)
	if err != nil {
		return nil, err
	}

	if err := s.executorRepo.Create(ctx, executor); err != nil {
		return nil, err
	}

	// 注册任务
	if len(req.Tasks) > 0 {
		if err := s.registerTasks(ctx, executor.Name, req.Tasks); err != nil {
			s.logger.Error("注册任务失败",
				zap.String("executor_name", executor.Name),
				zap.Error(err))
		}
	}

	s.logger.Info("执行器注册成功",
		zap.String("executor_id", executor.ID),
		zap.String("instance_id", executor.InstanceID),
		zap.String("name", executor.Name),
		zap.Int("tasks_count", len(req.Tasks)))

	return executor, nil
}

func (s *ExecutorService) registerTasks(ctx context.Context, executorName string, taskDefs []TaskDefinitionRequest) error {
	for _, taskDef := range taskDefs {
		if err := s.registerSingleTask(ctx, executorName, taskDef); err != nil {
			s.logger.Error("注册单个任务失败",
				zap.String("executor_name", executorName),
				zap.String("task_name", taskDef.Name),
				zap.Error(err))
			// 继续处理其他任务，不因为单个任务失败而终止
		}
	}
	return nil
}

func (s *ExecutorService) registerSingleTask(ctx context.Context, executorName string, taskDef TaskDefinitionRequest) error {
	// 查找任务（按名称）
	task, err := s.taskRepo.GetByName(ctx, taskDef.Name)

	if err != nil && err != domainError.ErrTaskNotFound {
		return err
	}

	if err == domainError.ErrTaskNotFound {
		// 任务不存在，创建新任务
		task, err = entity.NewTask(taskDef.Name, taskDef.CronExpression)
		if err != nil {
			return err
		}

		// 设置任务属性
		if taskDef.Parameters != nil {
			task.Parameters = taskDef.Parameters
		}
		if taskDef.ExecutionMode != "" {
			task.ExecutionMode = taskDef.ExecutionMode
		}
		if taskDef.LoadBalanceStrategy != "" {
			task.LoadBalanceStrategy = taskDef.LoadBalanceStrategy
		}
		if taskDef.MaxRetry > 0 {
			task.MaxRetry = taskDef.MaxRetry
		}
		if taskDef.TimeoutSeconds > 0 {
			task.TimeoutSeconds = taskDef.TimeoutSeconds
		}
		if taskDef.Status != "" {
			task.Status = taskDef.Status
		} else {
			task.Status = entity.TaskStatusPaused // 默认为暂停状态
		}

		if err := s.taskRepo.Create(ctx, task); err != nil {
			return err
		}

		s.logger.Info("任务创建成功",
			zap.String("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.String("status", string(task.Status)))
	}

	// 检查任务执行器关联是否已存在
	existingAssignments, err := s.taskRepo.GetTaskExecutors(ctx, task.ID)
	if err != nil {
		return err
	}

	// 检查是否已有相同执行器的分配
	for _, assignment := range existingAssignments {
		if assignment.ExecutorName == executorName {
			// 关联已存在，不做修改
			return nil
		}
	}

	// 创建新的任务执行器关联
	_, err = s.taskRepo.AssignExecutor(ctx, task.ID, executorName, 1, 1)
	if err != nil {
		return err
	}

	s.logger.Info("任务执行器关联创建成功",
		zap.String("task_id", task.ID),
		zap.String("executor_name", executorName))

	return nil
}

func (s *ExecutorService) GetExecutor(ctx context.Context, id string) (*entity.Executor, error) {
	return s.executorRepo.GetByID(ctx, id)
}

func (s *ExecutorService) GetExecutorByName(ctx context.Context, name string) (*entity.Executor, error) {
	return s.executorRepo.GetByName(ctx, name)
}

func (s *ExecutorService) UpdateExecutor(ctx context.Context, id, name, baseURL, healthCheckURL string) (*entity.Executor, error) {
	executor, err := s.executorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	executor.Update(name, baseURL, healthCheckURL)

	if err := s.executorRepo.Update(ctx, executor); err != nil {
		return nil, err
	}

	return executor, nil
}

func (s *ExecutorService) UpdateExecutorStatus(ctx context.Context, id string, status entity.ExecutorStatus) error {
	executor, err := s.executorRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	executor.UpdateStatus(status)

	return s.executorRepo.Update(ctx, executor)
}

func (s *ExecutorService) DeleteExecutor(ctx context.Context, id string) error {
	return s.executorRepo.Delete(ctx, id)
}

func (s *ExecutorService) ListExecutors(ctx context.Context, filter repository.ExecutorFilter) ([]*entity.Executor, error) {
	return s.executorRepo.List(ctx, filter)
}

func (s *ExecutorService) UpdateExecutorHealth(ctx context.Context, id string, isHealthy bool) error {
	executor, err := s.executorRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	executor.SetHealthy(isHealthy)

	return s.executorRepo.Update(ctx, executor)
}
