package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/domain/entity"
	domainError "github.com/jobs/scheduler/internal/domain/error"
	"github.com/jobs/scheduler/internal/domain/repository"
	"github.com/jobs/scheduler/internal/models"
	"gorm.io/gorm"
)

type executionRepositoryImpl struct {
	db *gorm.DB
}

// NewExecutionRepository 创建执行记录仓储实现
func NewExecutionRepository(db *gorm.DB) repository.ExecutionRepository {
	return &executionRepositoryImpl{db: db}
}

func (r *executionRepositoryImpl) Create(ctx context.Context, execution *entity.TaskExecution) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}

	model := r.toModel(execution)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return domainError.NewBusinessError("EXECUTION_CREATE_FAILED", "创建执行记录失败", err)
	}

	execution.ID = model.ID
	return nil
}

func (r *executionRepositoryImpl) GetByID(ctx context.Context, id string) (*entity.TaskExecution, error) {
	var model models.TaskExecution
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainError.ErrExecutionNotFound
	}
	if err != nil {
		return nil, domainError.NewBusinessError("EXECUTION_QUERY_FAILED", "查询执行记录失败", err)
	}

	return r.toEntity(&model), nil
}

func (r *executionRepositoryImpl) Update(ctx context.Context, execution *entity.TaskExecution) error {
	model := r.toModel(execution)
	err := r.db.WithContext(ctx).Save(model).Error
	if err != nil {
		return domainError.NewBusinessError("EXECUTION_UPDATE_FAILED", "更新执行记录失败", err)
	}
	return nil
}

func (r *executionRepositoryImpl) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.TaskExecution{})
	if result.Error != nil {
		return domainError.NewBusinessError("EXECUTION_DELETE_FAILED", "删除执行记录失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainError.ErrExecutionNotFound
	}
	return nil
}

func (r *executionRepositoryImpl) List(ctx context.Context, filter repository.ExecutionFilter) ([]*entity.TaskExecution, int64, error) {
	var execModels []models.TaskExecution
	query := r.db.WithContext(ctx)

	// 应用过滤条件
	if filter.TaskID != "" {
		query = query.Where("task_id = ?", filter.TaskID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.StartTime != nil {
		query = query.Where("scheduled_time >= ?", filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("scheduled_time <= ?", filter.EndTime)
	}

	var total int64
	countQuery := r.db.WithContext(ctx).Model(&models.TaskExecution{})
	if filter.TaskID != "" {
		countQuery = countQuery.Where("task_id = ?", filter.TaskID)
	}
	if filter.Status != "" {
		countQuery = countQuery.Where("status = ?", filter.Status)
	}
	if filter.StartTime != nil {
		countQuery = countQuery.Where("scheduled_time >= ?", filter.StartTime)
	}
	if filter.EndTime != nil {
		countQuery = countQuery.Where("scheduled_time <= ?", filter.EndTime)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, domainError.NewBusinessError("EXECUTION_COUNT_FAILED", "统计执行记录失败", err)
	}

	// 分页查询
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize

	err := query.Order("scheduled_time DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&execModels).Error

	if err != nil {
		return nil, 0, domainError.NewBusinessError("EXECUTION_LIST_FAILED", "查询执行记录列表失败", err)
	}

	entities := make([]*entity.TaskExecution, len(execModels))
	for i, model := range execModels {
		entities[i] = r.toEntity(&model)
	}

	return entities, total, nil
}

func (r *executionRepositoryImpl) Count(ctx context.Context, filter repository.ExecutionFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.TaskExecution{})

	if filter.TaskID != "" {
		query = query.Where("task_id = ?", filter.TaskID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.StartTime != nil {
		query = query.Where("scheduled_time >= ?", filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("scheduled_time <= ?", filter.EndTime)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, domainError.NewBusinessError("EXECUTION_COUNT_FAILED", "统计执行记录失败", err)
	}

	return count, nil
}

func (r *executionRepositoryImpl) GetStats(ctx context.Context, filter repository.ExecutionStatsFilter) (*repository.ExecutionStats, error) {
	var stats repository.ExecutionStats

	query := r.db.WithContext(ctx).Model(&models.TaskExecution{})
	if filter.TaskID != "" {
		query = query.Where("task_id = ?", filter.TaskID)
	}
	if filter.StartTime != nil {
		query = query.Where("scheduled_time >= ?", filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("scheduled_time <= ?", filter.EndTime)
	}

	// 统计总数
	if err := query.Count(&stats.Total).Error; err != nil {
		return nil, domainError.NewBusinessError("EXECUTION_STATS_FAILED", "获取执行统计失败", err)
	}

	// 统计各状态数量
	query.Where("status = ?", models.ExecutionStatusSuccess).Count(&stats.Success)
	query.Where("status = ?", models.ExecutionStatusFailed).Count(&stats.Failed)
	query.Where("status = ?", models.ExecutionStatusRunning).Count(&stats.Running)
	query.Where("status = ?", models.ExecutionStatusPending).Count(&stats.Pending)

	return &stats, nil
}

func (r *executionRepositoryImpl) GetTaskStats(ctx context.Context, taskID string, days int) (*repository.TaskHealthStats, error) {
	since := time.Now().AddDate(0, 0, -days)

	var totalCount, successCount, failedCount, timeoutCount int64

	// 总执行次数
	r.db.WithContext(ctx).Model(&models.TaskExecution{}).
		Where("task_id = ? AND created_at >= ?", taskID, since).
		Count(&totalCount)

	// 成功次数
	r.db.WithContext(ctx).Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusSuccess, since).
		Count(&successCount)

	// 失败次数
	r.db.WithContext(ctx).Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusFailed, since).
		Count(&failedCount)

	// 超时次数
	r.db.WithContext(ctx).Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusTimeout, since).
		Count(&timeoutCount)

	// 计算健康度分数 (0-100)
	healthScore := float64(100)
	if totalCount > 0 {
		successRate := float64(successCount) / float64(totalCount)
		healthScore = successRate * 70

		timeoutRate := float64(timeoutCount) / float64(totalCount)
		healthScore += (1 - timeoutRate) * 30
	}

	// 计算平均执行时间
	var avgDuration float64
	r.db.WithContext(ctx).Model(&models.TaskExecution{}).
		Where("task_id = ? AND created_at >= ? AND start_time IS NOT NULL AND end_time IS NOT NULL",
			taskID, since).
		Select("COALESCE(AVG(TIMESTAMPDIFF(SECOND, start_time, end_time)), 0)").
		Scan(&avgDuration)

	return &repository.TaskHealthStats{
		HealthScore:        healthScore,
		TotalCount:         totalCount,
		SuccessCount:       successCount,
		FailedCount:        failedCount,
		TimeoutCount:       timeoutCount,
		AvgDurationSeconds: avgDuration,
		PeriodDays:         days,
	}, nil
}

func (r *executionRepositoryImpl) GetDailyStats(ctx context.Context, taskID string, days int) ([]*repository.DailyStats, error) {
	var dailyStats []*repository.DailyStats

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayTotal, daySuccess int64

		// 总数
		r.db.WithContext(ctx).Model(&models.TaskExecution{}).
			Where("task_id = ? AND created_at >= ? AND created_at < ?",
				taskID, startOfDay, endOfDay).
			Count(&dayTotal)

		// 成功数
		r.db.WithContext(ctx).Model(&models.TaskExecution{}).
			Where("task_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
				taskID, models.ExecutionStatusSuccess, startOfDay, endOfDay).
			Count(&daySuccess)

		successRate := float64(100) // 默认100%（无执行时）
		if dayTotal > 0 {
			successRate = float64(daySuccess) / float64(dayTotal) * 100
		}

		dailyStats = append(dailyStats, &repository.DailyStats{
			Date:        startOfDay.Format("2006-01-02"),
			SuccessRate: successRate,
			Total:       dayTotal,
		})
	}

	return dailyStats, nil
}

func (r *executionRepositoryImpl) GetRecentExecutions(ctx context.Context, taskID string, days int) ([]*repository.RecentExecutionStats, error) {
	var recentStats []*repository.RecentExecutionStats

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayTotal, daySuccess, dayFailed int64

		// 总数
		r.db.WithContext(ctx).Model(&models.TaskExecution{}).
			Where("task_id = ? AND created_at >= ? AND created_at < ?",
				taskID, startOfDay, endOfDay).
			Count(&dayTotal)

		// 成功数
		r.db.WithContext(ctx).Model(&models.TaskExecution{}).
			Where("task_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
				taskID, models.ExecutionStatusSuccess, startOfDay, endOfDay).
			Count(&daySuccess)

		// 失败数
		r.db.WithContext(ctx).Model(&models.TaskExecution{}).
			Where("task_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
				taskID, []string{string(models.ExecutionStatusFailed), string(models.ExecutionStatusTimeout)},
				startOfDay, endOfDay).
			Count(&dayFailed)

		successRate := float64(0)
		if dayTotal > 0 {
			successRate = float64(daySuccess) / float64(dayTotal) * 100
		}

		recentStats = append(recentStats, &repository.RecentExecutionStats{
			Date:        startOfDay.Format("2006-01-02"),
			Total:       int(dayTotal),
			Success:     int(daySuccess),
			Failed:      int(dayFailed),
			SuccessRate: successRate,
		})
	}

	return recentStats, nil
}

func (r *executionRepositoryImpl) ListRunning(ctx context.Context) ([]*entity.TaskExecution, error) {
	var execModels []models.TaskExecution
	err := r.db.WithContext(ctx).
		Where("status = ?", "running").
		Find(&execModels).Error

	if err != nil {
		return nil, domainError.NewBusinessError("LIST_RUNNING_FAILED", "查询运行中的执行记录失败", err)
	}

	entities := make([]*entity.TaskExecution, len(execModels))
	for i, model := range execModels {
		entities[i] = r.toEntity(&model)
	}

	return entities, nil
}

func (r *executionRepositoryImpl) ListPending(ctx context.Context) ([]*entity.TaskExecution, error) {
	var execModels []models.TaskExecution
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Find(&execModels).Error

	if err != nil {
		return nil, domainError.NewBusinessError("LIST_PENDING_FAILED", "查询待执行记录失败", err)
	}

	entities := make([]*entity.TaskExecution, len(execModels))
	for i, model := range execModels {
		entities[i] = r.toEntity(&model)
	}

	return entities, nil
}

func (r *executionRepositoryImpl) ListByTaskID(ctx context.Context, taskID string, limit int) ([]*entity.TaskExecution, error) {
	var execModels []models.TaskExecution
	query := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("scheduled_time DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&execModels).Error
	if err != nil {
		return nil, domainError.NewBusinessError("LIST_BY_TASK_FAILED", "查询任务执行记录失败", err)
	}

	entities := make([]*entity.TaskExecution, len(execModels))
	for i, model := range execModels {
		entities[i] = r.toEntity(&model)
	}

	return entities, nil
}

func (r *executionRepositoryImpl) CountRunningByExecutor(ctx context.Context, executorID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.TaskExecution{}).
		Where("executor_id = ? AND status = ?", executorID, models.ExecutionStatusRunning).
		Count(&count).Error

	if err != nil {
		return 0, domainError.NewBusinessError("COUNT_RUNNING_FAILED", "统计执行器运行任务失败", err)
	}

	return count, nil
}

// 模型转换方法
func (r *executionRepositoryImpl) toModel(execution *entity.TaskExecution) *models.TaskExecution {
	return &models.TaskExecution{
		ID:            execution.ID,
		TaskID:        execution.TaskID,
		ExecutorID:    execution.ExecutorID,
		ScheduledTime: execution.ScheduledTime,
		StartTime:     execution.StartTime,
		EndTime:       execution.EndTime,
		Status:        models.ExecutionStatus(execution.Status),
		Result:        models.JSONMap(execution.Result),
		Logs:          execution.Logs,
		CreatedAt:     execution.CreatedAt,
	}
}

func (r *executionRepositoryImpl) toEntity(model *models.TaskExecution) *entity.TaskExecution {
	execution := &entity.TaskExecution{
		ID:            model.ID,
		TaskID:        model.TaskID,
		ExecutorID:    model.ExecutorID,
		ScheduledTime: model.ScheduledTime,
		StartTime:     model.StartTime,
		EndTime:       model.EndTime,
		Status:        entity.ExecutionStatus(model.Status),
		Result:        map[string]any(model.Result),
		Logs:          model.Logs,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.CreatedAt, // 使用CreatedAt作为UpdatedAt
	}

	// 手动查询关联的Task（如果需要）
	// 在应用层实现关联，不依赖GORM Preload
	if model.TaskID != "" {
		var task models.Task
		if err := r.db.Where("id = ?", model.TaskID).First(&task).Error; err == nil {
			execution.Task = &entity.Task{
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

	// 手动查询关联的Executor（如果需要）
	if model.ExecutorID != nil && *model.ExecutorID != "" {
		var executor models.Executor
		if err := r.db.Where("id = ?", *model.ExecutorID).First(&executor).Error; err == nil {
			execution.Executor = &entity.Executor{
				ID:                  executor.ID,
				Name:                executor.Name,
				InstanceID:          executor.InstanceID,
				BaseURL:             executor.BaseURL,
				HealthCheckURL:      executor.HealthCheckURL,
				Status:              entity.ExecutorStatus(executor.Status),
				IsHealthy:           executor.IsHealthy,
				HealthCheckFailures: executor.HealthCheckFailures,
				LastHealthCheck:     executor.LastHealthCheck,
				Metadata:            executor.Metadata,
				CreatedAt:           executor.CreatedAt,
				UpdatedAt:           executor.UpdatedAt,
			}
		}
	}

	return execution
}
