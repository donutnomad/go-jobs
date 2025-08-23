package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/domain/entity"
	domainError "github.com/jobs/scheduler/internal/domain/error"
	"github.com/jobs/scheduler/internal/domain/repository"
	"github.com/jobs/scheduler/internal/models"
	"gorm.io/gorm"
)

type executorRepositoryImpl struct {
	db *gorm.DB
}

// NewExecutorRepository 创建执行器仓储实现
func NewExecutorRepository(db *gorm.DB) repository.ExecutorRepository {
	return &executorRepositoryImpl{db: db}
}

func (r *executorRepositoryImpl) Create(ctx context.Context, executor *entity.Executor) error {
	if executor.ID == "" {
		executor.ID = uuid.New().String()
	}

	model := r.toModel(executor)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return domainError.NewBusinessError("EXECUTOR_CREATE_FAILED", "创建执行器失败", err)
	}

	executor.ID = model.ID
	return nil
}

func (r *executorRepositoryImpl) GetByID(ctx context.Context, id string) (*entity.Executor, error) {
	var model models.Executor
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainError.ErrExecutorNotFound
	}
	if err != nil {
		return nil, domainError.NewBusinessError("EXECUTOR_QUERY_FAILED", "查询执行器失败", err)
	}

	executor := r.toEntity(&model)

	// 手动查询并关联任务信息
	var taskExecutors []models.TaskExecutor
	if err := r.db.Where("executor_name = ?", model.Name).Find(&taskExecutors).Error; err == nil {
		executor.TaskExecutors = make([]entity.TaskExecutor, len(taskExecutors))
		for j, te := range taskExecutors {
			executor.TaskExecutors[j] = entity.TaskExecutor{
				ID:           te.ID,
				TaskID:       te.TaskID,
				ExecutorName: te.ExecutorName,
				Priority:     te.Priority,
				Weight:       te.Weight,
			}

			// 手动查询关联的Task
			var task models.Task
			if err := r.db.Where("id = ?", te.TaskID).First(&task).Error; err == nil {
				executor.TaskExecutors[j].Task = &entity.Task{
					ID:                  task.ID,
					Name:                task.Name,
					CronExpression:      task.CronExpression,
					Parameters:          map[string]any(task.Parameters),
					ExecutionMode:       entity.ExecutionMode(task.ExecutionMode),
					LoadBalanceStrategy: entity.LoadBalanceStrategy(task.LoadBalanceStrategy),
					MaxRetry:            task.MaxRetry,
					TimeoutSeconds:      task.TimeoutSeconds,
					Status:              entity.TaskStatus(task.Status),
					CreatedAt:           task.CreatedAt,
					UpdatedAt:           task.UpdatedAt,
				}
			}
		}
	}

	return executor, nil
}

func (r *executorRepositoryImpl) GetByInstanceID(ctx context.Context, instanceID string) (*entity.Executor, error) {
	var model models.Executor
	err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainError.ErrExecutorNotFound
	}
	if err != nil {
		return nil, domainError.NewBusinessError("EXECUTOR_QUERY_FAILED", "查询执行器失败", err)
	}

	return r.toEntity(&model), nil
}

func (r *executorRepositoryImpl) GetByName(ctx context.Context, name string) (*entity.Executor, error) {
	var model models.Executor
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainError.ErrExecutorNotFound
	}
	if err != nil {
		return nil, domainError.NewBusinessError("EXECUTOR_QUERY_FAILED", "查询执行器失败", err)
	}

	return r.toEntity(&model), nil
}

func (r *executorRepositoryImpl) Update(ctx context.Context, executor *entity.Executor) error {
	model := r.toModel(executor)
	err := r.db.WithContext(ctx).Save(model).Error
	if err != nil {
		return domainError.NewBusinessError("EXECUTOR_UPDATE_FAILED", "更新执行器失败", err)
	}
	return nil
}

func (r *executorRepositoryImpl) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
	})

	if err != nil {
		return domainError.NewBusinessError("EXECUTOR_DELETE_FAILED", "删除执行器失败", err)
	}
	return nil
}

func (r *executorRepositoryImpl) List(ctx context.Context, filter repository.ExecutorFilter) ([]*entity.Executor, error) {
	var execModels []models.Executor
	query := r.db.WithContext(ctx)

	if filter.Name != "" {
		query = query.Where("name = ?", filter.Name)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.IsHealthy != nil {
		query = query.Where("is_healthy = ?", *filter.IsHealthy)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	err := query.Find(&execModels).Error
	if err != nil {
		return nil, domainError.NewBusinessError("EXECUTOR_LIST_FAILED", "查询执行器列表失败", err)
	}

	entities := make([]*entity.Executor, len(execModels))
	for i, model := range execModels {
		entities[i] = r.toEntity(&model)

		// 直接查询并关联任务信息（移除IncludeTasks条件检查）
		var taskExecutors []models.TaskExecutor
		if err := r.db.Where("executor_name = ?", model.Name).Find(&taskExecutors).Error; err == nil {
			entities[i].TaskExecutors = make([]entity.TaskExecutor, len(taskExecutors))
			for j, te := range taskExecutors {
				entities[i].TaskExecutors[j] = entity.TaskExecutor{
					ID:           te.ID,
					TaskID:       te.TaskID,
					ExecutorName: te.ExecutorName,
					Priority:     te.Priority,
					Weight:       te.Weight,
				}

				// 手动查询关联的Task
				var task models.Task
				if err := r.db.Where("id = ?", te.TaskID).First(&task).Error; err == nil {
					entities[i].TaskExecutors[j].Task = &entity.Task{
						ID:                  task.ID,
						Name:                task.Name,
						CronExpression:      task.CronExpression,
						Parameters:          map[string]any(task.Parameters),
						ExecutionMode:       entity.ExecutionMode(task.ExecutionMode),
						LoadBalanceStrategy: entity.LoadBalanceStrategy(task.LoadBalanceStrategy),
						MaxRetry:            task.MaxRetry,
						TimeoutSeconds:      task.TimeoutSeconds,
						Status:              entity.TaskStatus(task.Status),
						CreatedAt:           task.CreatedAt,
						UpdatedAt:           task.UpdatedAt,
					}
				}
			}
		}
	}

	return entities, nil
}

func (r *executorRepositoryImpl) Count(ctx context.Context, filter repository.ExecutorFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Executor{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.IsHealthy != nil {
		query = query.Where("is_healthy = ?", *filter.IsHealthy)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, domainError.NewBusinessError("EXECUTOR_COUNT_FAILED", "统计执行器数量失败", err)
	}

	return count, nil
}

func (r *executorRepositoryImpl) ListOnline(ctx context.Context) ([]*entity.Executor, error) {
	return r.List(ctx, repository.ExecutorFilter{Status: entity.ExecutorStatusOnline})
}

func (r *executorRepositoryImpl) ListHealthy(ctx context.Context) ([]*entity.Executor, error) {
	healthy := true
	return r.List(ctx, repository.ExecutorFilter{IsHealthy: &healthy})
}

func (r *executorRepositoryImpl) GetExecutorsForTask(ctx context.Context, taskID string) ([]*entity.Executor, error) {
	var taskExecutors []models.TaskExecutor
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Find(&taskExecutors).Error

	if err != nil {
		return nil, domainError.NewBusinessError("GET_TASK_EXECUTORS_FAILED", "获取任务执行器失败", err)
	}

	executors := make([]*entity.Executor, 0)

	// 为每个TaskExecutor获取所有同名的执行器实例
	for _, te := range taskExecutors {
		var execModels []models.Executor
		err = r.db.WithContext(ctx).
			Where("name = ?", te.ExecutorName).
			Find(&execModels).Error
		if err != nil {
			continue // 跳过出错的执行器
		}

		// 转换所有同名的执行器实例
		for _, execModel := range execModels {
			executors = append(executors, r.toEntity(&execModel))
		}
	}

	return executors, nil
}

// 模型转换方法
func (r *executorRepositoryImpl) toModel(executor *entity.Executor) *models.Executor {
	return &models.Executor{
		ID:                  executor.ID,
		Name:                executor.Name,
		InstanceID:          executor.InstanceID,
		BaseURL:             executor.BaseURL,
		HealthCheckURL:      executor.HealthCheckURL,
		Status:              models.ExecutorStatus(executor.Status),
		IsHealthy:           executor.IsHealthy,
		HealthCheckFailures: executor.HealthCheckFailures,
		LastHealthCheck:     executor.LastHealthCheck,
		Metadata:            executor.Metadata,
		CreatedAt:           executor.CreatedAt,
		UpdatedAt:           executor.UpdatedAt,
	}
}

func (r *executorRepositoryImpl) toEntity(model *models.Executor) *entity.Executor {
	return &entity.Executor{
		ID:                  model.ID,
		Name:                model.Name,
		InstanceID:          model.InstanceID,
		BaseURL:             model.BaseURL,
		HealthCheckURL:      model.HealthCheckURL,
		Status:              entity.ExecutorStatus(model.Status),
		IsHealthy:           model.IsHealthy,
		HealthCheckFailures: model.HealthCheckFailures,
		LastHealthCheck:     model.LastHealthCheck,
		Metadata:            model.Metadata,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
	}
}
