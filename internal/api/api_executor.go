package api

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IExecutorAPI interface {
	// List 获取执行器列表
	// 获取所有的执行器列表
	// @GET(api/v1/executors)
	List(ctx *gin.Context, req ListExecutorReq) ([]*models.Executor, error)

	// Get 获取执行器详情
	// 获取指定id的执行器详情
	// @GET(api/v1/executors/{id})
	Get(ctx *gin.Context, id string) (*models.Executor, error)

	// Register 注册执行器
	// 注册一个新执行器
	// @POST(api/v1/executors/register)
	Register(ctx *gin.Context, req RegisterExecutorReq) (*models.Executor, error)

	// Update 更新执行器
	// 更新指定id的执行器
	// @PUT(api/v1/executors/{id})
	Update(ctx *gin.Context, id string, req UpdateExecutorReq) (models.Executor, error)

	// UpdateStatus 更新执行器状态
	// 更新指定id的执行器状态
	// @PUT(api/v1/executors/{id}/status)
	UpdateStatus(ctx *gin.Context, id string, req UpdateExecutorStatusReq) (string, error)

	// Delete 删除执行器
	// 删除指定id的执行器
	// @DELETE(api/v1/executors/{id})
	Delete(ctx *gin.Context, id string) (string, error)
}

type ExecutorAPI struct {
	db     *gorm.DB
	logger *zap.Logger
	mu     sync.RWMutex
}

func NewExecutorAPI(db *gorm.DB, logger *zap.Logger) IExecutorAPI {
	return &ExecutorAPI{db: db, logger: logger, mu: sync.RWMutex{}}
}

func (e *ExecutorAPI) List(ctx *gin.Context, req ListExecutorReq) ([]*models.Executor, error) {
	var executors []*models.Executor
	if err := e.db.WithContext(ctx).Find(&executors).Error; err != nil {
		return nil, fmt.Errorf("failed to list executors: %w", err)
	}

	// 统计每个执行器的运行任务数
	for _, exec := range executors {
		var count int64
		e.db.Model(&models.TaskExecution{}).
			Where("executor_id = ? AND status = ?", exec.ID, models.ExecutionStatusRunning).
			Count(&count)
	}

	// 为每个执行器加载关联的任务（直接加载，不再依赖include_tasks参数）
	for _, exe := range executors {
		var taskExecutors []models.TaskExecutor
		err := e.db.
			Where("executor_name = ?", exe.Name).
			Find(&taskExecutors).Error
		if err != nil {
			e.logger.Error("failed to load task executors",
				zap.String("executor_name", exe.Name),
				zap.Error(err))
			continue
		}
		
		// 为每个TaskExecutor加载关联的Task信息
		for i := range taskExecutors {
			var task models.Task
			if err := e.db.Where("id = ?", taskExecutors[i].TaskID).First(&task).Error; err == nil {
				taskExecutors[i].Task = &task
			}
		}
		
		// 在应用层手动填充关联数据
		exe.TaskExecutors = taskExecutors
	}
	sort.Slice(executors, func(i, j int) bool {
		return executors[i].Status.ToInt() < executors[j].Status.ToInt()
	})
	return executors, nil
}

func (e *ExecutorAPI) Get(ctx *gin.Context, id string) (*models.Executor, error) {
	var exec models.Executor
	if err := e.db.WithContext(ctx).Where("id = ?", id).First(&exec).Error; err != nil {
		return nil, fmt.Errorf("executor not found: %w", err)
	}
	
	// 手动加载关联的TaskExecutors
	var taskExecutors []models.TaskExecutor
	if err := e.db.Where("executor_name = ?", exec.Name).Find(&taskExecutors).Error; err == nil {
		// 为每个TaskExecutor加载关联的Task信息
		for i := range taskExecutors {
			var task models.Task
			if err := e.db.Where("id = ?", taskExecutors[i].TaskID).First(&task).Error; err == nil {
				taskExecutors[i].Task = &task
			}
		}
		exec.TaskExecutors = taskExecutors
	}
	
	return &exec, nil
}

func (e *ExecutorAPI) Update(ctx *gin.Context, id string, req UpdateExecutorReq) (models.Executor, error) {
	// 查找执行器
	var ret models.Executor
	if err := e.db.Where("id = ?", id).First(&ret).Error; err != nil {
		return models.Executor{}, err
	}

	// 更新字段
	if req.Name != "" {
		ret.Name = req.Name
	}
	if req.BaseURL != "" {
		ret.BaseURL = req.BaseURL
	}
	if req.HealthCheckURL != "" {
		ret.HealthCheckURL = req.HealthCheckURL
	}

	// 保存更新
	if err := e.db.Save(&ret).Error; err != nil {
		return models.Executor{}, err
	}

	return ret, nil
}

func (e *ExecutorAPI) UpdateStatus(ctx *gin.Context, id string, req UpdateExecutorStatusReq) (string, error) {
	var exec models.Executor
	if err := e.db.WithContext(ctx).Where("id = ?", id).First(&exec).Error; err != nil {
		return "", fmt.Errorf("executor not found: %w", err)
	}
	exec.SetStatus(req.Status)
	if err := e.db.Save(&exec).Error; err != nil {
		return "", fmt.Errorf("failed to update executor status: %w", err)
	}
	return "status updated", nil
}

func (e *ExecutorAPI) Delete(ctx *gin.Context, id string) (string, error) {
	if err := e.db.Transaction(func(tx *gorm.DB) error {
		// 先查找执行器获取其名称
		var exec models.Executor
		if err := tx.Where("id = ?", id).First(&exec).Error; err != nil {
			return err
		}

		// 删除基于executor_name的TaskExecutor关联
		if err := tx.Where("executor_name = ?", exec.Name).Delete(&models.TaskExecutor{}).Error; err != nil {
			return err
		}

		// 删除执行器本身
		if err := tx.Where("id = ?", id).Delete(&models.Executor{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}
	return "executor deleted", nil
}

func (e *ExecutorAPI) Register(ctx *gin.Context, req RegisterExecutorReq) (*models.Executor, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查是否存在使用相同 instance_id 的执行器
	var executor models.Executor
	err := e.db.Where("instance_id = ?", req.ExecutorID).First(&executor).Error

	if err == nil {
		// 如果执行器已存在且在线，拒绝注册（防止挤掉别人）
		if executor.Status == models.ExecutorStatusOnline {
			// 检查是否是同一个执行器重新注册（通过 BaseURL 判断）
			if executor.BaseURL != req.ExecutorURL {
				return nil, fmt.Errorf("executor with instance_id %s is already online from different location (current: %s, new: %s)",
					req.ExecutorID, executor.BaseURL, req.ExecutorURL)
			}
		}

		// 更新现有执行器信息
		executor.Name = req.ExecutorName
		executor.BaseURL = req.ExecutorURL
		executor.HealthCheckURL = req.HealthCheckURL
		executor.Status = models.ExecutorStatusOnline
		executor.IsHealthy = true
		executor.HealthCheckFailures = 0
		var now = time.Now()
		executor.LastHealthCheck = &now

		if err := e.db.Save(&executor).Error; err != nil {
			return nil, fmt.Errorf("failed to update executor: %w", err)
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
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
		if err := e.db.Create(&executor).Error; err != nil {
			return nil, fmt.Errorf("failed to create executor: %w", err)
		}
	} else {
		return nil, fmt.Errorf("failed to query executor: %w", err)
	}

	// 注册任务
	if len(req.Tasks) > 0 {
		for _, taskDef := range req.Tasks {
			if err := e.registerTask(executor.Name, taskDef); err != nil {
				e.logger.Error("failed to register task",
					zap.String("executor_name", executor.Name),
					zap.String("task_name", taskDef.Name),
					zap.Error(err))
				// 继续处理其他任务，不因为单个任务失败而终止
			}
		}
	}

	e.logger.Info("executor registered",
		zap.String("executor_id", executor.ID),
		zap.String("instance_id", executor.InstanceID),
		zap.String("name", executor.Name),
		zap.Int("tasks_count", len(req.Tasks)))

	return &executor, nil
}

func (e *ExecutorAPI) registerTask(executorName string, taskDef TaskDefinition) error {
	// 查找任务（按名称）
	var task models.Task
	err := e.db.Where("name = ?", taskDef.Name).First(&task).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
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
			task.Parameters = make(map[string]any)
		}

		if err := e.db.Create(&task).Error; err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		e.logger.Info("task created",
			zap.String("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.String("status", string(task.Status)))
	} else if err != nil {
		return fmt.Errorf("failed to query task: %w", err)
	}
	// 如果任务存在，不修改任务信息（按需求）

	// 检查任务执行器关联是否已存在
	var taskExecutor models.TaskExecutor
	err = e.db.Where("task_id = ? AND executor_name = ?", task.ID, executorName).First(&taskExecutor).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建新的任务执行器关联
		taskExecutor = models.TaskExecutor{
			ID:           uuid.New().String(),
			TaskID:       task.ID,
			ExecutorName: executorName,
			Priority:     1, // 默认优先级
			Weight:       1, // 默认权重
		}

		if err := e.db.Create(&taskExecutor).Error; err != nil {
			return fmt.Errorf("failed to create task executor association: %w", err)
		}

		e.logger.Info("task executor association created",
			zap.String("task_id", task.ID),
			zap.String("executor_name", executorName))
	} else if err != nil {
		return fmt.Errorf("failed to query task executor: %w", err)
	}
	// 如果关联已存在，不做修改

	return nil
}
