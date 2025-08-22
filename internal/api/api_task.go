package api

import (
	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/executor"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/internal/scheduler"
	"go.uber.org/zap"
)

type ITaskAPI interface {
	// List 获取任务列表
	// 获取所有的任务列表
	// @GET(api/v1/tasks)
	List(ctx *gin.Context, req GetTasksReq) ([]models.Task, error)

	// Get 获取任务详情
	// 获取指定id的任务详情
	// @GET(api/v1/tasks/{id})
	Get(ctx *gin.Context, id string) (models.Task, error)

	// Create 创建任务
	// 创建一个新任务
	// @POST(api/v1/tasks)
	Create(ctx *gin.Context, req CreateTaskRequest) (models.Task, error)

	// Delete 删除任务
	// 删除指定id的任务
	// @DELETE(api/v1/tasks/{id})
	Delete(ctx *gin.Context, id string) (string, error)

	// UpdateTask 更新任务
	// 更新指定id的任务
	// @PUT(api/v1/tasks/{id})
	UpdateTask(ctx *gin.Context, id string, req UpdateTaskRequest) (models.Task, error)

	// TriggerTask 手动触发任务
	// 手动触发指定id的任务
	// @POST(api/v1/tasks/{id}/trigger)
	TriggerTask(ctx *gin.Context, id string, req executor.TriggerTaskRequest) (models.TaskExecution, error)

	// Pause 暂停任务
	// 暂停指定id的任务
	// @POST(api/v1/tasks/{id}/pause)
	Pause(ctx *gin.Context, id string) (string, error)

	// Resume 恢复任务
	// 恢复指定id的任务
	// @POST(api/v1/tasks/{id}/resume)
	Resume(ctx *gin.Context, id string) (string, error)

	// GetTaskExecutors 获取任务的执行器列表
	// 获取指定id的任务的执行器列表
	// @GET(api/v1/tasks/{id}/executors)
	GetTaskExecutors(ctx *gin.Context, id string) ([]models.TaskExecutor, error)

	// AssignExecutor 为任务分配执行器
	// 为指定id的任务分配执行器
	// @POST(api/v1/tasks/{id}/executors)
	AssignExecutor(ctx *gin.Context, id string, req AssignExecutorRequest) (models.TaskExecutor, error)

	// UpdateExecutorAssignment 更新任务执行器分配
	// 更新指定id的任务执行器分配
	// @PUT(api/v1/tasks/{id}/executors/{executor_id})
	UpdateExecutorAssignment(ctx *gin.Context, id string, executorID string, req UpdateExecutorAssignmentRequest) (models.TaskExecutor, error)

	// UnassignExecutor 取消任务执行器分配
	// 取消指定id的任务执行器分配
	// @DELETE(api/v1/tasks/{id}/executors/{executor_id})
	UnassignExecutor(ctx *gin.Context, id string, executorID string) (string, error)

	// GetTaskStats 获取任务统计
	// 获取指定id的任务统计
	// @GET(api/v1/tasks/{id}/stats)
	GetTaskStats(ctx *gin.Context, id string) (TaskStats, error)
}

type TaskStats struct {
	SuccessRate24h   float64            `json:"success_rate_24h"`
	Total24h         int64              `json:"total_24h"`
	Success24h       int64              `json:"success_24h"`
	Health90d        HealthStatus       `json:"health_90d"`
	RecentExecutions []RecentExecutions `json:"recent_executions"`
	DailyStats90d    []map[string]any   `json:"daily_stats_90d"`
}

type RecentExecutions struct {
	Date        string  `json:"date"`
	Total       int     `json:"total"`
	Success     int     `json:"success"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

var _ ITaskAPI = (*TaskAPI)(nil)

type TaskAPI struct {
	storage   *orm.Storage
	scheduler *scheduler.Scheduler
}

func NewTaskAPI(storage *orm.Storage, scheduler *scheduler.Scheduler) ITaskAPI {
	return &TaskAPI{
		storage:   storage,
		scheduler: scheduler,
	}
}

type GetTasksReq struct {
	Status models.TaskStatus `form:"status" binding:"omitempty"`
}

func (t *TaskAPI) List(ctx *gin.Context, req GetTasksReq) ([]models.Task, error) {
	var tasks []models.Task
	query := t.storage.DB().Preload("TaskExecutors").Preload("TaskExecutors.Executor")

	// 支持状态过滤
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

func (t *TaskAPI) Get(ctx *gin.Context, id string) (models.Task, error) {
	var task models.Task

	query := t.storage.DB().Preload("TaskExecutors").Preload("TaskExecutors.Executor")

	if err := query.
		Where("id = ?", id).
		First(&task).
		Error; err != nil {
		return models.Task{}, errors.Join(err, errors.New("task not found"))
	}
	return task, nil
}

func (t *TaskAPI) Create(ctx *gin.Context, req CreateTaskRequest) (models.Task, error) {
	task := models.Task{
		ID:                  generateID(),
		Name:                req.Name,
		CronExpression:      req.CronExpression,
		Parameters:          req.Parameters,
		ExecutionMode:       req.ExecutionMode,
		LoadBalanceStrategy: req.LoadBalanceStrategy,
		MaxRetry:            req.MaxRetry,
		TimeoutSeconds:      req.TimeoutSeconds,
		Status:              models.TaskStatusActive,
	}

	// 设置默认值
	if task.ExecutionMode == "" {
		task.ExecutionMode = models.ExecutionModeParallel
	}
	if task.LoadBalanceStrategy == "" {
		task.LoadBalanceStrategy = models.LoadBalanceRoundRobin
	}
	if task.MaxRetry == 0 {
		task.MaxRetry = 3
	}
	if task.TimeoutSeconds == 0 {
		task.TimeoutSeconds = 300
	}

	if err := t.storage.DB().Create(&task).Error; err != nil {
		return models.Task{}, errors.Join(err, errors.New("create task failed"))
	}

	return task, nil
}

func (t *TaskAPI) UpdateTask(ctx *gin.Context, taskID string, req UpdateTaskRequest) (models.Task, error) {
	var task models.Task
	if err := t.storage.DB().Where("id = ?", taskID).First(&task).Error; err != nil {
		return models.Task{}, errors.Join(err, errors.New("task not found"))
	}

	// 更新字段
	if req.Name != "" {
		task.Name = req.Name
	}
	if req.CronExpression != "" {
		task.CronExpression = req.CronExpression
	}
	if req.Parameters != nil {
		task.Parameters = req.Parameters
	}
	if req.ExecutionMode != "" {
		task.ExecutionMode = req.ExecutionMode
	}
	if req.LoadBalanceStrategy != "" {
		task.LoadBalanceStrategy = req.LoadBalanceStrategy
	}
	if req.MaxRetry > 0 {
		task.MaxRetry = req.MaxRetry
	}
	if req.TimeoutSeconds > 0 {
		task.TimeoutSeconds = req.TimeoutSeconds
	}
	if req.Status != "" {
		task.Status = req.Status
	}

	if err := t.storage.DB().Save(&task).Error; err != nil {
		return models.Task{}, errors.Join(err, errors.New("update task failed"))
	}

	return task, nil
}

func (t *TaskAPI) Delete(ctx *gin.Context, id string) (string, error) {
	// 软删除，将状态设置为deleted
	result := t.storage.DB().
		Model(&models.Task{}).
		Where("id = ?", id).
		Update("status", models.TaskStatusDeleted)

	if result.Error != nil {
		return "", errors.Join(result.Error, errors.New("delete task failed"))
	}

	if result.RowsAffected == 0 {
		return "", errors.New("task not found")
	}

	return "task deleted successfully", nil
}

func (t *TaskAPI) TriggerTask(ctx *gin.Context, id string, req executor.TriggerTaskRequest) (models.TaskExecution, error) {
	execution, err := t.scheduler.TriggerTask(ctx.Request.Context(), id, req.Parameters)
	if err != nil {
		return models.TaskExecution{}, errors.Join(err, errors.New("trigger task failed"))
	}

	return *execution, nil
}

func (t *TaskAPI) Pause(ctx *gin.Context, id string) (string, error) {
	// 查找任务
	var task models.Task
	if err := t.storage.DB().Where("id = ?", id).First(&task).Error; err != nil {
		return "", errors.Join(err, errors.New("task not found"))
	}

	// 检查任务状态
	if task.Status == models.TaskStatusPaused {
		return "", errors.New("task is already paused")
	}

	if task.Status == models.TaskStatusDeleted {
		return "", errors.New("cannot pause deleted task")
	}

	// 更新任务状态为暂停
	if err := t.storage.DB().
		Model(&task).
		Where("id = ?", id).
		Update("status", models.TaskStatusPaused).Error; err != nil {
		return "", errors.Join(err, errors.New("update task status failed"))
	}

	// 重新加载调度器任务
	if err := t.scheduler.ReloadTasks(); err != nil {
		log.Println("failed to reload tasks after pause", zap.Error(err))
	}

	return "task paused successfully", nil
}

func (t *TaskAPI) Resume(ctx *gin.Context, id string) (string, error) {
	// 查找任务
	var task models.Task
	if err := t.storage.DB().Where("id = ?", id).First(&task).Error; err != nil {
		return "", errors.Join(err, errors.New("task not found"))
	}

	// 检查任务状态
	if task.Status == models.TaskStatusActive {
		return "", errors.New("task is already active")
	}

	if task.Status == models.TaskStatusDeleted {
		return "", errors.New("cannot resume deleted task")
	}

	// 更新任务状态为活跃
	if err := t.storage.DB().
		Model(&task).
		Where("id = ?", id).
		Update("status", models.TaskStatusActive).Error; err != nil {
		return "", errors.Join(err, errors.New("update task status failed"))
	}

	// 重新加载调度器任务
	if err := t.scheduler.ReloadTasks(); err != nil {
		log.Println("failed to reload tasks after resume", zap.Error(err))
	}

	return "task resumed successfully", nil
}

func (t *TaskAPI) GetTaskExecutors(ctx *gin.Context, id string) ([]models.TaskExecutor, error) {
	var taskExecutors []models.TaskExecutor
	if err := t.storage.DB().
		Preload("Executor").
		Where("task_id = ?", id).
		Find(&taskExecutors).
		Error; err != nil {
		return nil, errors.Join(err, errors.New("failed to get executors"))
	}

	return taskExecutors, nil
}

func (t *TaskAPI) AssignExecutor(ctx *gin.Context, id string, req AssignExecutorRequest) (models.TaskExecutor, error) {
	// 验证任务是否存在
	var task models.Task
	if err := t.storage.DB().Where("id = ?", id).First(&task).Error; err != nil {
		return models.TaskExecutor{}, errors.Join(err, errors.New("task not found"))
	}

	// 验证执行器是否存在
	var executor models.Executor
	if err := t.storage.DB().Where("id = ?", req.ExecutorID).First(&executor).Error; err != nil {
		return models.TaskExecutor{}, errors.Join(err, errors.New("executor not found"))
	}

	// 创建任务执行器关联
	taskExecutor := models.TaskExecutor{
		ID:         generateID(),
		TaskID:     id,
		ExecutorID: req.ExecutorID,
		Priority:   req.Priority,
		Weight:     req.Weight,
	}

	// 设置默认值
	if taskExecutor.Priority == 0 {
		taskExecutor.Priority = 1
	}
	if taskExecutor.Weight == 0 {
		taskExecutor.Weight = 1
	}

	if err := t.storage.DB().Create(&taskExecutor).Error; err != nil {
		return models.TaskExecutor{}, errors.Join(err, errors.New("create task executor failed"))
	}

	return taskExecutor, nil
}

func (t *TaskAPI) UpdateExecutorAssignment(ctx *gin.Context, id string, executorID string, req UpdateExecutorAssignmentRequest) (models.TaskExecutor, error) {
	// 查找现有分配
	var taskExecutor models.TaskExecutor
	if err := t.storage.DB().
		Where("task_id = ? AND executor_id = ?", id, executorID).
		First(&taskExecutor).Error; err != nil {
		return models.TaskExecutor{}, errors.Join(err, errors.New("assignment not found"))
	}

	// 更新分配
	taskExecutor.Priority = req.Priority
	taskExecutor.Weight = req.Weight

	if err := t.storage.DB().Save(&taskExecutor).Error; err != nil {
		return models.TaskExecutor{}, errors.Join(err, errors.New("update task executor failed"))
	}

	return taskExecutor, nil
}

func (t *TaskAPI) UnassignExecutor(ctx *gin.Context, id string, executorID string) (string, error) {
	result := t.storage.DB().
		Where("task_id = ? AND executor_id = ?", id, executorID).
		Delete(&models.TaskExecutor{})

	if result.Error != nil {
		return "", errors.Join(result.Error, errors.New("delete task executor failed"))
	}

	if result.RowsAffected == 0 {
		return "", errors.New("assignment not found")
	}

	return "executor unassigned", nil
}

func (t *TaskAPI) GetTaskStats(ctx *gin.Context, taskID string) (TaskStats, error) {
	// 获取24小时成功率
	var successRate24h float64
	var totalCount24h int64
	var successCount24h int64

	// 计算24小时前的时间
	since24h := time.Now().Add(-24 * time.Hour)

	// 获取24小时内的总执行次数
	if err := t.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND created_at >= ?", taskID, since24h).
		Count(&totalCount24h).Error; err != nil {
		return TaskStats{}, errors.Join(err, errors.New("failed to get 24h total count"))
	}

	// 获取24小时内的成功执行次数
	if err := t.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusSuccess, since24h).
		Count(&successCount24h).Error; err != nil {
		return TaskStats{}, errors.Join(err, errors.New("failed to get 24h success count"))
	}

	if totalCount24h > 0 {
		successRate24h = float64(successCount24h) / float64(totalCount24h) * 100
	}

	// 获取90天健康度统计
	healthStats90d := t.calculateHealthStats(taskID, 90)

	// 获取90天每日统计（用于状态图）
	var dailyStats []map[string]any
	for i := 89; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayTotal, daySuccess int64

		// 总数
		t.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND created_at >= ? AND created_at < ?",
				taskID, startOfDay, endOfDay).
			Count(&dayTotal)

		// 成功数
		t.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
				taskID, models.ExecutionStatusSuccess, startOfDay, endOfDay).
			Count(&daySuccess)

		successRate := float64(100) // 默认100%（无执行时）
		if dayTotal > 0 {
			successRate = float64(daySuccess) / float64(dayTotal) * 100
		}

		dailyStats = append(dailyStats, map[string]any{
			"date":        startOfDay.Format("2006-01-02"),
			"successRate": successRate,
			"total":       dayTotal,
		})
	}

	// 获取最近执行统计
	var recentExecutions []RecentExecutions

	// 按天统计最近7天的执行情况
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayTotal, daySuccess, dayFailed int64

		// 总数
		t.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND created_at >= ? AND created_at < ?",
				taskID, startOfDay, endOfDay).
			Count(&dayTotal)

		// 成功数
		t.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
				taskID, models.ExecutionStatusSuccess, startOfDay, endOfDay).
			Count(&daySuccess)

		// 失败数
		t.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
				taskID, []string{string(models.ExecutionStatusFailed), string(models.ExecutionStatusTimeout)},
				startOfDay, endOfDay).
			Count(&dayFailed)

		successRate := float64(0)
		if dayTotal > 0 {
			successRate = float64(daySuccess) / float64(dayTotal) * 100
		}

		recentExecutions = append(recentExecutions, RecentExecutions{
			Date:        startOfDay.Format("2006-01-02"),
			Total:       int(dayTotal),
			Success:     int(daySuccess),
			Failed:      int(dayFailed),
			SuccessRate: successRate,
		})
	}

	return TaskStats{
		SuccessRate24h:   successRate24h,
		Total24h:         totalCount24h,
		Success24h:       successCount24h,
		Health90d:        healthStats90d,
		RecentExecutions: recentExecutions,
		DailyStats90d:    dailyStats,
	}, nil
}

// calculateHealthStats 计算健康度统计
func (t *TaskAPI) calculateHealthStats(taskID string, days int) HealthStatus {
	since := time.Now().AddDate(0, 0, -days)

	var totalCount, successCount, failedCount, timeoutCount int64

	// 总执行次数
	t.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND created_at >= ?", taskID, since).
		Count(&totalCount)

	// 成功次数
	t.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusSuccess, since).
		Count(&successCount)

	// 失败次数
	t.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusFailed, since).
		Count(&failedCount)

	// 超时次数
	t.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusTimeout, since).
		Count(&timeoutCount)

	// 计算健康度分数 (0-100)
	healthScore := float64(100)
	if totalCount > 0 {
		// 成功率占70%权重
		successRate := float64(successCount) / float64(totalCount)
		healthScore = successRate * 70

		// 超时率占30%权重（超时越少分数越高）
		timeoutRate := float64(timeoutCount) / float64(totalCount)
		healthScore += (1 - timeoutRate) * 30
	}

	// 计算平均执行时间
	var avgDuration float64
	t.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND created_at >= ? AND start_time IS NOT NULL AND end_time IS NOT NULL",
			taskID, since).
		Select("AVG(TIMESTAMPDIFF(SECOND, start_time, end_time))").
		Scan(&avgDuration)

	return HealthStatus{
		HealthScore:        healthScore,
		TotalCount:         totalCount,
		SuccessCount:       successCount,
		FailedCount:        failedCount,
		TimeoutCount:       timeoutCount,
		AvgDurationSeconds: avgDuration,
		PeriodDays:         days,
	}
}

type HealthStatus struct {
	HealthScore        float64 `json:"health_score"`
	TotalCount         int64   `json:"total_count"`
	SuccessCount       int64   `json:"success_count"`
	FailedCount        int64   `json:"failed_count"`
	TimeoutCount       int64   `json:"timeout_count"`
	AvgDurationSeconds float64 `json:"avg_duration_seconds"`
	PeriodDays         int     `json:"period_days"`
}
