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

type taskRepositoryImpl struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓储实现
func NewTaskRepository(db *gorm.DB) repository.TaskRepository {
	return &taskRepositoryImpl{db: db}
}

func (r *taskRepositoryImpl) Create(ctx context.Context, task *entity.Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	model := r.toModel(task)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return domainError.NewBusinessError("TASK_CREATE_FAILED", "创建任务失败", err)
	}

	// 更新实体ID
	task.ID = model.ID
	return nil
}

func (r *taskRepositoryImpl) GetByID(ctx context.Context, id string) (*entity.Task, error) {
	var model models.Task
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainError.ErrTaskNotFound
	}
	if err != nil {
		return nil, domainError.NewBusinessError("TASK_QUERY_FAILED", "查询任务失败", err)
	}

	return r.toEntity(&model), nil
}

func (r *taskRepositoryImpl) GetByName(ctx context.Context, name string) (*entity.Task, error) {
	var model models.Task
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainError.ErrTaskNotFound
	}
	if err != nil {
		return nil, domainError.NewBusinessError("TASK_QUERY_FAILED", "查询任务失败", err)
	}

	return r.toEntity(&model), nil
}

func (r *taskRepositoryImpl) Update(ctx context.Context, task *entity.Task) error {
	model := r.toModel(task)
	err := r.db.WithContext(ctx).Save(model).Error
	if err != nil {
		return domainError.NewBusinessError("TASK_UPDATE_FAILED", "更新任务失败", err)
	}
	return nil
}

func (r *taskRepositoryImpl) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Task{}).
		Where("id = ?", id).
		Update("status", models.TaskStatusDeleted)

	if result.Error != nil {
		return domainError.NewBusinessError("TASK_DELETE_FAILED", "删除任务失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainError.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepositoryImpl) List(ctx context.Context, filter repository.TaskFilter) ([]*entity.Task, error) {
	var models []models.Task
	query := r.db.WithContext(ctx)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	err := query.Find(&models).Error
	if err != nil {
		return nil, domainError.NewBusinessError("TASK_LIST_FAILED", "查询任务列表失败", err)
	}

	entities := make([]*entity.Task, len(models))
	for i, model := range models {
		entities[i] = r.toEntity(&model)
	}

	return entities, nil
}

func (r *taskRepositoryImpl) Count(ctx context.Context, filter repository.TaskFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Task{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, domainError.NewBusinessError("TASK_COUNT_FAILED", "统计任务数量失败", err)
	}

	return count, nil
}

func (r *taskRepositoryImpl) ListActive(ctx context.Context) ([]*entity.Task, error) {
	return r.List(ctx, repository.TaskFilter{Status: entity.TaskStatusActive})
}

func (r *taskRepositoryImpl) ListByStatus(ctx context.Context, status entity.TaskStatus) ([]*entity.Task, error) {
	return r.List(ctx, repository.TaskFilter{Status: status})
}

func (r *taskRepositoryImpl) AssignExecutor(ctx context.Context, taskID, executorName string, priority, weight int) (*entity.TaskExecutor, error) {
	taskExecutor := models.TaskExecutor{
		ID:           uuid.New().String(),
		TaskID:       taskID,
		ExecutorName: executorName,
		Priority:     priority,
		Weight:       weight,
	}

	err := r.db.WithContext(ctx).Create(&taskExecutor).Error
	if err != nil {
		return nil, domainError.NewBusinessError("ASSIGN_EXECUTOR_FAILED", "分配执行器失败", err)
	}

	return &entity.TaskExecutor{
		ID:           taskExecutor.ID,
		TaskID:       taskExecutor.TaskID,
		ExecutorName: taskExecutor.ExecutorName,
		Priority:     taskExecutor.Priority,
		Weight:       taskExecutor.Weight,
	}, nil
}

func (r *taskRepositoryImpl) UnassignExecutor(ctx context.Context, taskID, executorName string) error {
	result := r.db.WithContext(ctx).
		Where("task_id = ? AND executor_name = ?", taskID, executorName).
		Delete(&models.TaskExecutor{})

	if result.Error != nil {
		return domainError.NewBusinessError("UNASSIGN_EXECUTOR_FAILED", "取消分配执行器失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainError.ErrAssignmentNotFound
	}
	return nil
}

func (r *taskRepositoryImpl) UpdateAssignment(ctx context.Context, taskID, executorName string, priority, weight int) (*entity.TaskExecutor, error) {
	var taskExecutor models.TaskExecutor
	err := r.db.WithContext(ctx).
		Where("task_id = ? AND executor_name = ?", taskID, executorName).
		First(&taskExecutor).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainError.ErrAssignmentNotFound
	}
	if err != nil {
		return nil, domainError.NewBusinessError("UPDATE_ASSIGNMENT_FAILED", "更新分配失败", err)
	}

	taskExecutor.Priority = priority
	taskExecutor.Weight = weight

	err = r.db.WithContext(ctx).Save(&taskExecutor).Error
	if err != nil {
		return nil, domainError.NewBusinessError("UPDATE_ASSIGNMENT_FAILED", "保存分配失败", err)
	}

	return &entity.TaskExecutor{
		ID:           taskExecutor.ID,
		TaskID:       taskExecutor.TaskID,
		ExecutorName: taskExecutor.ExecutorName,
		Priority:     taskExecutor.Priority,
		Weight:       taskExecutor.Weight,
	}, nil
}

func (r *taskRepositoryImpl) GetTaskExecutors(ctx context.Context, taskID string) ([]*entity.TaskExecutor, error) {
	var taskExecutors []models.TaskExecutor
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Find(&taskExecutors).Error

	if err != nil {
		return nil, domainError.NewBusinessError("GET_TASK_EXECUTORS_FAILED", "获取任务执行器失败", err)
	}

	entities := make([]*entity.TaskExecutor, len(taskExecutors))
	for i, model := range taskExecutors {
		entities[i] = &entity.TaskExecutor{
			ID:           model.ID,
			TaskID:       model.TaskID,
			ExecutorName: model.ExecutorName,
			Priority:     model.Priority,
			Weight:       model.Weight,
		}
	}

	return entities, nil
}

// 模型转换方法
func (r *taskRepositoryImpl) toModel(task *entity.Task) *models.Task {
	return &models.Task{
		ID:                  task.ID,
		Name:                task.Name,
		CronExpression:      task.CronExpression,
		Parameters:          models.JSONMap(task.Parameters),
		ExecutionMode:       models.ExecutionMode(task.ExecutionMode),
		LoadBalanceStrategy: models.LoadBalanceStrategy(task.LoadBalanceStrategy),
		MaxRetry:            task.MaxRetry,
		TimeoutSeconds:      task.TimeoutSeconds,
		Status:              models.TaskStatus(task.Status),
		CreatedAt:           task.CreatedAt,
		UpdatedAt:           task.UpdatedAt,
	}
}

func (r *taskRepositoryImpl) toEntity(model *models.Task) *entity.Task {
	task := &entity.Task{
		ID:                  model.ID,
		Name:                model.Name,
		CronExpression:      model.CronExpression,
		Parameters:          map[string]any(model.Parameters),
		ExecutionMode:       entity.ExecutionMode(model.ExecutionMode),
		LoadBalanceStrategy: entity.LoadBalanceStrategy(model.LoadBalanceStrategy),
		MaxRetry:            model.MaxRetry,
		TimeoutSeconds:      model.TimeoutSeconds,
		Status:              entity.TaskStatus(model.Status),
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
	}

	// 手动查询关联的TaskExecutors（在应用层实现关联）
	var taskExecutors []models.TaskExecutor
	if err := r.db.Where("task_id = ?", model.ID).Find(&taskExecutors).Error; err == nil {
		if len(taskExecutors) > 0 {
			task.TaskExecutors = make([]entity.TaskExecutor, len(taskExecutors))
			for i, te := range taskExecutors {
				task.TaskExecutors[i] = entity.TaskExecutor{
					ID:           te.ID,
					TaskID:       te.TaskID,
					ExecutorName: te.ExecutorName,
					Priority:     te.Priority,
					Weight:       te.Weight,
				}
			}
		}
	}

	return task
}
