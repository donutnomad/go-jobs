package repositories

import (
	"context"
	"fmt"
	"time"

	taskbiz "github.com/jobs/scheduler/internal/app/biz/task"
	"github.com/jobs/scheduler/internal/app/infra/models"
	"github.com/jobs/scheduler/internal/app/infra/persistence"
	"github.com/jobs/scheduler/internal/app/types"
	"gorm.io/gorm"
)

// TaskRepository 任务仓储实现
type TaskRepository struct {
	*persistence.DefaultRepo
}

// NewTaskRepository 创建任务仓储
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{
		DefaultRepo: persistence.NewDefaultRepo(db),
	}
}

// Save 保存任务
func (r *TaskRepository) Save(ctx context.Context, task *taskbiz.Task) error {
	db := r.Db(ctx)

	var model models.TaskModel
	model.FromEntity(task)

	if err := db.Create(&model).Error(); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	return nil
}

// FindByID 根据ID查找任务
func (r *TaskRepository) FindByID(ctx context.Context, id types.ID) (*taskbiz.Task, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	db := r.Db(ctx)

	var model models.TaskModel
	if err := db.Where("id = ?", string(id)).First(&model).Error(); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	return model.ToEntity()
}

// Update 更新任务
func (r *TaskRepository) Update(ctx context.Context, task *taskbiz.Task) error {
	db := r.Db(ctx)

	var model models.TaskModel
	model.FromEntity(task)

	if err := db.Where("id = ?", model.ID).Updates(&model).Error(); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// Delete 删除任务
func (r *TaskRepository) Delete(ctx context.Context, id types.ID) error {
	db := r.Db(ctx)

	if err := db.Where("id = ?", string(id)).Delete(&models.TaskModel{}).Error(); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// FindByName 根据名称查找任务
func (r *TaskRepository) FindByName(ctx context.Context, name string) (*taskbiz.Task, error) {
	db := r.Db(ctx)

	var model models.TaskModel
	if err := db.Where("name = ?", name).First(&model).Error(); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found: %s", name)
		}
		return nil, fmt.Errorf("failed to find task by name: %w", err)
	}

	return model.ToEntity()
}

// FindByStatus 根据状态查找任务
func (r *TaskRepository) FindByStatus(ctx context.Context, status taskbiz.TaskStatus, pagination types.Pagination) ([]*taskbiz.Task, error) {
	db := r.Db(ctx)

	var models []models.TaskModel
	query := db.Where("status = ?", int(status))

	// 应用分页
	if pagination.PageSize > 0 {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
	}

	if err := query.Find(&models).Error(); err != nil {
		return nil, fmt.Errorf("failed to find tasks by status: %w", err)
	}

	tasks := make([]*taskbiz.Task, len(models))
	for i, model := range models {
		task, err := model.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to entity: %w", err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// FindAll 查找所有任务
func (r *TaskRepository) FindAll(ctx context.Context, pagination types.Pagination) ([]*taskbiz.Task, error) {
	db := r.Db(ctx)

	var models []models.TaskModel
	query := db

	// 应用分页
	if pagination.PageSize > 0 {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
	}

	if err := query.Find(&models).Error(); err != nil {
		return nil, fmt.Errorf("failed to find all tasks: %w", err)
	}

	tasks := make([]*taskbiz.Task, len(models))
	for i, model := range models {
		task, err := model.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to entity: %w", err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// FindActiveSchedulableTasks 查找活跃的可调度任务
func (r *TaskRepository) FindActiveSchedulableTasks(ctx context.Context) ([]*taskbiz.Task, error) {
	return r.FindByStatus(ctx, taskbiz.TaskStatusActive, types.Pagination{})
}

// FindByFilters 根据过滤条件查找任务
func (r *TaskRepository) FindByFilters(ctx context.Context, filters taskbiz.TaskFilters, pagination types.Pagination) ([]*taskbiz.Task, error) {
	db := r.Db(ctx)

	query := db.Model(&models.TaskModel{})

	// 应用过滤条件
	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Status != nil {
		query = query.Where("status = ?", int(*filters.Status))
	}
	if filters.ExecutionMode != nil {
		query = query.Where("execution_mode = ?", string(*filters.ExecutionMode))
	}
	if filters.LoadBalanceStrategy != nil {
		query = query.Where("load_balance_strategy = ?", string(*filters.LoadBalanceStrategy))
	}
	if filters.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filters.CreatedBefore)
	}
	if filters.UpdatedAfter != nil {
		query = query.Where("updated_at >= ?", *filters.UpdatedAfter)
	}
	if filters.UpdatedBefore != nil {
		query = query.Where("updated_at <= ?", *filters.UpdatedBefore)
	}

	// 应用分页
	if pagination.PageSize > 0 {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
	}

	var models []models.TaskModel
	if err := query.Find(&models).Error(); err != nil {
		return nil, fmt.Errorf("failed to find tasks by filters: %w", err)
	}

	tasks := make([]*taskbiz.Task, len(models))
	for i, model := range models {
		task, err := model.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to entity: %w", err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// Count 统计任务数量
func (r *TaskRepository) Count(ctx context.Context, filters taskbiz.TaskFilters) (int64, error) {
	db := r.Db(ctx)

	query := db.Model(&models.TaskModel{})

	// 应用过滤条件
	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Status != nil {
		query = query.Where("status = ?", int(*filters.Status))
	}
	if filters.ExecutionMode != nil {
		query = query.Where("execution_mode = ?", string(*filters.ExecutionMode))
	}
	if filters.LoadBalanceStrategy != nil {
		query = query.Where("load_balance_strategy = ?", string(*filters.LoadBalanceStrategy))
	}
	if filters.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filters.CreatedBefore)
	}
	if filters.UpdatedAfter != nil {
		query = query.Where("updated_at >= ?", *filters.UpdatedAfter)
	}
	if filters.UpdatedBefore != nil {
		query = query.Where("updated_at <= ?", *filters.UpdatedBefore)
	}

	var count int64
	if err := query.Count(&count).Error(); err != nil {
		return 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return count, nil
}

// FindTasksReadyForScheduling 查找准备调度的任务
func (r *TaskRepository) FindTasksReadyForScheduling(ctx context.Context, currentTime time.Time, limit int) ([]*taskbiz.Task, error) {
	// 这里应该根据cron表达式和上次执行时间判断哪些任务需要执行
	// 简化实现，返回所有活跃任务
	return r.FindByStatus(ctx, taskbiz.TaskStatusActive, types.Pagination{
		Page:     1,
		PageSize: limit,
	})
}

// FindTasksByExecutionMode 根据执行模式查找任务
func (r *TaskRepository) FindTasksByExecutionMode(ctx context.Context, mode taskbiz.ExecutionMode) ([]*taskbiz.Task, error) {
	db := r.Db(ctx)

	var models []models.TaskModel
	if err := db.Where("execution_mode = ?", string(mode)).Find(&models).Error(); err != nil {
		return nil, fmt.Errorf("failed to find tasks by execution mode: %w", err)
	}

	tasks := make([]*taskbiz.Task, len(models))
	for i, model := range models {
		task, err := model.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to entity: %w", err)
		}
		tasks[i] = task
	}

	return tasks, nil
}

// BatchUpdateStatus 批量更新状态
func (r *TaskRepository) BatchUpdateStatus(ctx context.Context, taskIDs []types.ID, status taskbiz.TaskStatus) error {
	if len(taskIDs) == 0 {
		return nil
	}

	db := r.Db(ctx)

	ids := make([]string, len(taskIDs))
	for i, id := range taskIDs {
		ids[i] = string(id)
	}

	if err := db.Model(&models.TaskModel{}).Where("id IN ?", ids).Update("status", int(status)).Error(); err != nil {
		return fmt.Errorf("failed to batch update status: %w", err)
	}

	return nil
}

// BatchDelete 批量删除
func (r *TaskRepository) BatchDelete(ctx context.Context, taskIDs []types.ID) error {
	if len(taskIDs) == 0 {
		return nil
	}

	db := r.Db(ctx)

	ids := make([]string, len(taskIDs))
	for i, id := range taskIDs {
		ids[i] = string(id)
	}

	if err := db.Where("id IN ?", ids).Delete(&models.TaskModel{}).Error(); err != nil {
		return fmt.Errorf("failed to batch delete tasks: %w", err)
	}

	return nil
}

// GetStatusCounts 获取状态统计
func (r *TaskRepository) GetStatusCounts(ctx context.Context) (map[taskbiz.TaskStatus]int64, error) {
	db := r.Db(ctx)

	type StatusCount struct {
		Status int   `json:"status"`
		Count  int64 `json:"count"`
	}

	var results []StatusCount
	if err := db.Model(&models.TaskModel{}).Select("status, count(*) as count").Group("status").Find(&results).Error(); err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	counts := make(map[taskbiz.TaskStatus]int64)
	for _, result := range results {
		counts[taskbiz.TaskStatus(result.Status)] = result.Count
	}

	return counts, nil
}

// GetExecutionModeStats 获取执行模式统计
func (r *TaskRepository) GetExecutionModeStats(ctx context.Context) (map[taskbiz.ExecutionMode]int64, error) {
	db := r.Db(ctx)

	type ModeCount struct {
		ExecutionMode string `json:"execution_mode"`
		Count         int64  `json:"count"`
	}

	var results []ModeCount
	if err := db.Model(&models.TaskModel{}).Select("execution_mode, count(*) as count").Group("execution_mode").Find(&results).Error(); err != nil {
		return nil, fmt.Errorf("failed to get execution mode stats: %w", err)
	}

	stats := make(map[taskbiz.ExecutionMode]int64)
	for _, result := range results {
		stats[taskbiz.ExecutionMode(result.ExecutionMode)] = result.Count
	}

	return stats, nil
}

// ExistsByName 检查名称是否存在
func (r *TaskRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	db := r.Db(ctx)

	var count int64
	if err := db.Model(&models.TaskModel{}).Where("name = ?", name).Count(&count).Error(); err != nil {
		return false, fmt.Errorf("failed to check name existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByID 检查ID是否存在
func (r *TaskRepository) ExistsByID(ctx context.Context, id types.ID) (bool, error) {
	db := r.Db(ctx)

	var count int64
	if err := db.Model(&models.TaskModel{}).Where("id = ?", string(id)).Count(&count).Error(); err != nil {
		return false, fmt.Errorf("failed to check ID existence: %w", err)
	}

	return count > 0, nil
}
