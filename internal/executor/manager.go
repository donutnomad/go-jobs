package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/storage"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Manager struct {
	storage *storage.Storage
	logger  *zap.Logger
	mu      sync.RWMutex
}

func NewManager(storage *storage.Storage, logger *zap.Logger) *Manager {
	return &Manager{
		storage: storage,
		logger:  logger,
	}
}

// RegisterExecutor 注册执行器和相关任务
func (m *Manager) RegisterExecutor(ctx context.Context, req RegisterRequest) (*models.Executor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否存在使用相同 instance_id 的执行器
	var executor models.Executor
	err := m.storage.DB().Where("instance_id = ?", req.ExecutorID).First(&executor).Error

	if err == nil {
		// 如果执行器已存在且在线，拒绝注册（防止挤掉别人）
		if executor.Status == models.ExecutorStatusOnline {
			// 检查是否是同一个执行器重新注册（通过 BaseURL 判断）
			if executor.BaseURL != req.ExecutorURL {
				return nil, fmt.Errorf("executor with instance_id %s is already online from different location (current: %s, new: %s)",
					req.ExecutorID, executor.BaseURL, req.ExecutorURL)
			}
			// 如果是同一个执行器（相同的 BaseURL），允许更新信息
		}

		// 更新现有执行器信息
		executor.Name = req.ExecutorName
		executor.BaseURL = req.ExecutorURL
		executor.HealthCheckURL = req.HealthCheckURL
		if executor.HealthCheckURL == "" {
			executor.HealthCheckURL = req.ExecutorURL + "/health"
		}
		executor.Status = models.ExecutorStatusOnline
		executor.IsHealthy = true
		executor.HealthCheckFailures = 0
		var now = time.Now()
		executor.LastHealthCheck = &now

		if err := m.storage.DB().Save(&executor).Error; err != nil {
			return nil, fmt.Errorf("failed to update executor: %w", err)
		}
	} else if err == gorm.ErrRecordNotFound {
		// ExecutorID不存在，创建新执行器
		var now = time.Now()
		executor = models.Executor{
			ID:                  uuid.New().String(),
			Name:                req.ExecutorName,
			InstanceID:          req.ExecutorID,
			BaseURL:             req.ExecutorURL,
			HealthCheckURL:      req.HealthCheckURL,
			Status:              models.ExecutorStatusOnline,
			IsHealthy:           true,
			HealthCheckFailures: 0,
			LastHealthCheck:     &now,
			Metadata:            req.Metadata,
		}

		if executor.HealthCheckURL == "" {
			executor.HealthCheckURL = req.ExecutorURL + "/health"
		}

		if err := m.storage.DB().Create(&executor).Error; err != nil {
			return nil, fmt.Errorf("failed to create executor: %w", err)
		}
	} else {
		return nil, fmt.Errorf("failed to query executor: %w", err)
	}

	// 注册任务
	if len(req.Tasks) > 0 {
		for _, taskDef := range req.Tasks {
			if err := m.registerTask(ctx, executor.ID, taskDef); err != nil {
				m.logger.Error("failed to register task",
					zap.String("executor_id", executor.ID),
					zap.String("task_name", taskDef.Name),
					zap.Error(err))
				// 继续处理其他任务，不因为单个任务失败而终止
			}
		}
	}

	m.logger.Info("executor registered",
		zap.String("executor_id", executor.ID),
		zap.String("instance_id", executor.InstanceID),
		zap.String("name", executor.Name),
		zap.Int("tasks_count", len(req.Tasks)))

	return &executor, nil
}

// registerTask 注册单个任务
func (m *Manager) registerTask(ctx context.Context, executorID string, taskDef TaskDefinition) error {
	// 查找任务（按名称）
	var task models.Task
	err := m.storage.DB().Where("name = ?", taskDef.Name).First(&task).Error

	if err == gorm.ErrRecordNotFound {
		// 任务不存在，创建新任务
		task = models.Task{
			ID:                  uuid.New().String(),
			Name:                taskDef.Name,
			CronExpression:      taskDef.CronExpression,
			Parameters:          taskDef.Parameters,
			ExecutionMode:       taskDef.ExecutionMode,
			LoadBalanceStrategy: taskDef.LoadBalanceStrategy,
			MaxRetry:            taskDef.MaxRetry,
			TimeoutSeconds:      taskDef.TimeoutSeconds,
			Status:              taskDef.Status,
		}

		// 设置默认值
		if task.MaxRetry == 0 {
			task.MaxRetry = 3
		}
		if task.TimeoutSeconds == 0 {
			task.TimeoutSeconds = 300
		}
		if task.Status == "" {
			task.Status = models.TaskStatusPaused // 默认为暂停状态
		}
		if task.Parameters == nil {
			task.Parameters = make(map[string]interface{})
		}

		if err := m.storage.DB().Create(&task).Error; err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		m.logger.Info("task created",
			zap.String("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.String("status", string(task.Status)))
	} else if err != nil {
		return fmt.Errorf("failed to query task: %w", err)
	}
	// 如果任务存在，不修改任务信息（按需求）

	// 检查任务执行器关联是否已存在
	var taskExecutor models.TaskExecutor
	err = m.storage.DB().Where("task_id = ? AND executor_id = ?", task.ID, executorID).First(&taskExecutor).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新的任务执行器关联
		taskExecutor = models.TaskExecutor{
			ID:         uuid.New().String(),
			TaskID:     task.ID,
			ExecutorID: executorID,
			Priority:   1, // 默认优先级
			Weight:     1, // 默认权重
		}

		if err := m.storage.DB().Create(&taskExecutor).Error; err != nil {
			return fmt.Errorf("failed to create task executor association: %w", err)
		}

		m.logger.Info("task executor association created",
			zap.String("task_id", task.ID),
			zap.String("executor_id", executorID))
	} else if err != nil {
		return fmt.Errorf("failed to query task executor: %w", err)
	}
	// 如果关联已存在，不做修改

	return nil
}

/* TODO: 重新实现任务执行器注册
// registerTaskExecutor 注册任务与执行器的关联
func (m *Manager) registerTaskExecutor(ctx context.Context, executorID string, taskReg TaskRegistration) error {
	// 查找或创建任务
	var task models.Task
	err := m.storage.DB().Where("name = ?", taskReg.TaskName).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新任务
		task = models.Task{
			ID:                  uuid.New().String(),
			Name:                taskReg.TaskName,
			CronExpression:      taskReg.CronExpression,
			Parameters:          taskReg.Parameters,
			ExecutionMode:       taskReg.ExecutionMode,
			LoadBalanceStrategy: taskReg.LoadBalanceStrategy,
			MaxRetry:            taskReg.MaxRetry,
			TimeoutSeconds:      taskReg.TimeoutSeconds,
			Status:              models.TaskStatusActive,
		}
		if err := m.storage.DB().Create(&task).Error; err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query task: %w", err)
	}

	// 检查关联是否已存在
	var taskExecutor models.TaskExecutor
	err = m.storage.DB().Where("task_id = ? AND executor_id = ?", task.ID, executorID).First(&taskExecutor).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新关联
		taskExecutor = models.TaskExecutor{
			ID:         uuid.New().String(),
			TaskID:     task.ID,
			ExecutorID: executorID,
			Priority:   taskReg.Priority,
			Weight:     taskReg.Weight,
		}
		if err := m.storage.DB().Create(&taskExecutor).Error; err != nil {
			return fmt.Errorf("failed to create task executor: %w", err)
		}
	} else if err == nil {
		// 更新现有关联
		taskExecutor.Priority = taskReg.Priority
		taskExecutor.Weight = taskReg.Weight
		if err := m.storage.DB().Save(&taskExecutor).Error; err != nil {
			return fmt.Errorf("failed to update task executor: %w", err)
		}
	} else {
		return fmt.Errorf("failed to query task executor: %w", err)
	}

	return nil
}
*/

// UpdateExecutorStatus 更新执行器状态
func (m *Manager) UpdateExecutorStatus(ctx context.Context, executorID string, status models.ExecutorStatus, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var executor models.Executor
	if err := m.storage.DB().Where("id = ?", executorID).First(&executor).Error; err != nil {
		return fmt.Errorf("executor not found: %w", err)
	}

	executor.Status = status
	if status == models.ExecutorStatusOnline {
		executor.IsHealthy = true
		executor.HealthCheckFailures = 0
	}

	if err := m.storage.DB().Save(&executor).Error; err != nil {
		return fmt.Errorf("failed to update executor status: %w", err)
	}

	m.logger.Info("executor status updated",
		zap.String("executor_id", executorID),
		zap.String("status", string(status)),
		zap.String("reason", reason))

	return nil
}

// GetHealthyExecutors 获取健康的执行器列表
func (m *Manager) GetHealthyExecutors(ctx context.Context, taskID string) ([]*models.Executor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var executors []*models.Executor
	query := m.storage.DB().
		Joins("JOIN task_executors ON task_executors.executor_id = executors.id").
		Where("task_executors.task_id = ?", taskID).
		Where("executors.status = ?", models.ExecutorStatusOnline).
		Where("executors.is_healthy = ?", true).
		Order("task_executors.priority DESC")

	if err := query.Find(&executors).Error; err != nil {
		return nil, fmt.Errorf("failed to get healthy executors: %w", err)
	}

	return executors, nil
}

// GetExecutorByID 根据ID获取执行器
func (m *Manager) GetExecutorByID(ctx context.Context, executorID string) (*models.Executor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var executor models.Executor
	if err := m.storage.DB().Where("id = ?", executorID).First(&executor).Error; err != nil {
		return nil, fmt.Errorf("executor not found: %w", err)
	}

	return &executor, nil
}

// ListExecutors 列出所有执行器
func (m *Manager) ListExecutors(ctx context.Context) ([]*models.Executor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var executors []*models.Executor
	if err := m.storage.DB().Find(&executors).Error; err != nil {
		return nil, fmt.Errorf("failed to list executors: %w", err)
	}

	// 统计每个执行器的运行任务数
	for _, executor := range executors {
		var count int64
		m.storage.DB().Model(&models.TaskExecution{}).
			Where("executor_id = ? AND status = ?", executor.ID, models.ExecutionStatusRunning).
			Count(&count)
	}

	return executors, nil
}

// GetTaskExecutors 获取任务的所有执行器
func (m *Manager) GetTaskExecutors(ctx context.Context, taskID string) ([]*models.TaskExecutor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var taskExecutors []*models.TaskExecutor
	if err := m.storage.DB().
		Preload("Executor").
		Where("task_id = ?", taskID).
		Find(&taskExecutors).Error; err != nil {
		return nil, fmt.Errorf("failed to get task executors: %w", err)
	}

	return taskExecutors, nil
}
