package api

import (
	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/infra/persistence/executionrepo"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/yitter/idgenerator-go/idgen"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ITaskAPI interface {
	// List 获取任务列表
	// 获取所有的任务列表
	// @GET(api/v1/tasks)
	List(ctx *gin.Context, req GetTasksReq) ([]*TaskWithAssignmentsResp, error)

	// Get 获取任务详情
	// 获取指定id的任务详情
	// @GET(api/v1/tasks/{id})
	Get(ctx *gin.Context, id uint64) (*TaskWithAssignmentsResp, error)

	// Create 创建任务
	// 创建一个新任务
	// @POST(api/v1/tasks)
	Create(ctx *gin.Context, req CreateTaskReq) (*TaskResp, error)

	// Delete 删除任务
	// 删除指定id的任务
	// @DELETE(api/v1/tasks/{id})
	Delete(ctx *gin.Context, id uint64) (string, error)

	// UpdateTask 更新任务
	// 更新指定id的任务
	// @PUT(api/v1/tasks/{id})
	UpdateTask(ctx *gin.Context, id uint64, req UpdateTaskReq) (*TaskResp, error)

	// TriggerTask 手动触发任务
	// 手动触发指定id的任务
	// @POST(api/v1/tasks/{id}/trigger)
	TriggerTask(ctx *gin.Context, id uint64, req TriggerTaskRequest) (*TaskExecutionResp, error)

	// Pause 暂停任务
	// 暂停指定id的任务
	// @POST(api/v1/tasks/{id}/pause)
	Pause(ctx *gin.Context, id uint64) (string, error)

	// Resume 恢复任务
	// 恢复指定id的任务
	// @POST(api/v1/tasks/{id}/resume)
	Resume(ctx *gin.Context, id uint64) (string, error)

	// GetTaskExecutors 获取任务的执行器列表
	// 获取指定id的任务的执行器列表
	// @GET(api/v1/tasks/{id}/executors)
	GetTaskExecutors(ctx *gin.Context, id uint64) ([]*TaskAssignmentResp, error)

	// AssignExecutor 为任务分配执行器
	// 为指定id的任务分配执行器
	// @POST(api/v1/tasks/{id}/executors)
	AssignExecutor(ctx *gin.Context, id uint64, req AssignExecutorReq) (*TaskAssignmentResp, error)

	// UpdateExecutorAssignment 更新任务执行器分配
	// 更新指定id的任务执行器分配
	// @PUT(api/v1/tasks/{id}/executors/{executor_id})
	UpdateExecutorAssignment(ctx *gin.Context, id uint64, executorID uint64, req UpdateExecutorAssignmentReq) (*TaskAssignmentResp, error)

	// UnassignExecutor 取消任务执行器分配
	// 取消指定id的任务执行器分配
	// @DELETE(api/v1/tasks/{id}/executors/{executor_id})
	UnassignExecutor(ctx *gin.Context, id uint64, executorID uint64) (string, error)

	// GetTaskStats 获取任务统计
	// 获取指定id的任务统计
	// @GET(api/v1/tasks/{id}/stats)
	GetTaskStats(ctx *gin.Context, id uint64) (TaskStatsResp, error)
}

type TaskAPI struct {
	db            *gorm.DB
	emitter       IEmitter
	usecase       *task.Usecase
	repo          task.Repo
	executionRepo execution.Repo
}

func NewTaskAPI(db *gorm.DB, emitter IEmitter, usecase *task.Usecase, repo task.Repo) ITaskAPI {
	return &TaskAPI{
		db:      db,
		emitter: emitter,
		usecase: usecase,
		repo:    repo,
	}
}

func (t *TaskAPI) List(ctx *gin.Context, req GetTasksReq) ([]*TaskWithAssignmentsResp, error) {
	ret, err := t.repo.ListWithAssignments(ctx, &task.TaskFilter{
		Status: mo.EmptyableToOption(req.Status),
	})
	if err != nil {
		return nil, err
	}
	return lo.Map(ret, func(task *task.Task, _ int) *TaskWithAssignmentsResp {
		return new(TaskWithAssignmentsResp).FromDomain(task)
	}), nil
}

func (t *TaskAPI) Get(ctx *gin.Context, id uint64) (*TaskWithAssignmentsResp, error) {
	task_, err := t.repo.FindByIDWithAssignments(ctx, id)
	if err != nil {
		return nil, err
	} else if task_ == nil {
		return nil, errors.New("task not found")
	}
	return new(TaskWithAssignmentsResp).FromDomain(task_), nil
}

func (t *TaskAPI) Create(ctx *gin.Context, req CreateTaskReq) (*TaskResp, error) {
	ret := task.Task{
		ID:                  uint64(idgen.NextId()),
		Name:                req.Name,
		CronExpression:      req.CronExpression,
		Parameters:          req.Parameters,
		ExecutionMode:       req.GetExecutionMode(),
		LoadBalanceStrategy: req.GetLoadBalanceStrategy(),
		Status:              task.TaskStatusActive,
		MaxRetry:            req.GetMaxRetry(),
		TimeoutSeconds:      req.GetTimeoutSeconds(),
	}
	err := t.usecase.Create(ctx, &ret)
	if err != nil {
		return nil, err
	}
	return new(TaskResp).FromDomain(&ret), nil
}

func (t *TaskAPI) UpdateTask(ctx *gin.Context, taskID uint64, req UpdateTaskReq) (*TaskResp, error) {
	err := t.usecase.Update(ctx, taskID, &task.UpdateRequest{
		Name:                mo.EmptyableToOption(req.Name),
		CronExpression:      mo.EmptyableToOption(req.CronExpression),
		Parameters:          mo.EmptyableToOption(req.Parameters),
		ExecutionMode:       mo.EmptyableToOption(req.ExecutionMode),
		LoadBalanceStrategy: mo.EmptyableToOption(req.LoadBalanceStrategy),
		MaxRetry:            mo.EmptyableToOption(req.MaxRetry),
		TimeoutSeconds:      mo.EmptyableToOption(req.TimeoutSeconds),
		Status:              mo.EmptyableToOption(req.Status),
	})
	if err != nil {
		return nil, err
	}
	task_, err := t.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	} else if task_ == nil {
		return nil, errors.New("task not found")
	}
	return new(TaskResp).FromDomain(task_), nil
}

func (t *TaskAPI) Delete(ctx *gin.Context, id uint64) (string, error) {
	err := t.usecase.Delete(ctx, id)
	if err != nil {
		return "", err
	}
	return "task deleted successfully", nil
}

func (t *TaskAPI) TriggerTask(ctx *gin.Context, id uint64, req TriggerTaskRequest) (*TaskExecutionResp, error) {
	task_, err := t.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	} else if task_ == nil {
		return nil, errors.New("task not found")
	}
	new_ := execution.TaskExecution{
		ID:            uint64(idgen.NextId()),
		TaskID:        task_.ID,
		ScheduledTime: time.Now(),
		Status:        execution.ExecutionStatusPending,
	}
	err = t.executionRepo.Create(ctx, &new_)
	if err != nil {
		return nil, err
	}

	err = t.emitter.SubmitNewTask(id, req.Parameters, new_.ID)
	if err != nil {
		_ = t.executionRepo.Delete(ctx, new_.ID)
		return nil, errors.New("failed to submit task to emitter: " + err.Error())
	}

	out := new(TaskExecutionResp).FromDomain(&new_)
	out.Task = new(TaskResp).FromDomain(task_)
	return out, nil
}

func (t *TaskAPI) Pause(ctx *gin.Context, id uint64) (string, error) {
	err := t.usecase.Pause(ctx, id)
	if err != nil {
		return "", err
	}
	// 重新加载调度器任务
	if err := t.emitter.ReloadTasks(); err != nil {
		log.Println("failed to reload tasks after pause", zap.Error(err))
	}
	return "task paused successfully", nil
}

func (t *TaskAPI) Resume(ctx *gin.Context, id uint64) (string, error) {
	err := t.usecase.Resume(ctx, id)
	if err != nil {
		return "", err
	}
	// 重新加载调度器任务
	if err := t.emitter.ReloadTasks(); err != nil {
		log.Println("failed to reload tasks after resume", zap.Error(err))
	}
	return "task resumed successfully", nil
}

func (t *TaskAPI) GetTaskExecutors(ctx *gin.Context, id uint64) ([]*TaskAssignmentResp, error) {
	ret, err := t.repo.FindByIDWithAssignments(ctx, id)
	if err != nil {
		return nil, err
	} else if ret == nil {
		return nil, errors.New("task not found")
	}

	taskExecutors := lo.Map(ret.Assignments, func(assignment *task.TaskAssignment, _ int) *TaskAssignmentResp {
		return new(TaskAssignmentResp).FromDomain(assignment)
	})
	return taskExecutors, nil
}

func (t *TaskAPI) AssignExecutor(ctx *gin.Context, id uint64, req AssignExecutorReq) (*TaskAssignmentResp, error) {
	executor_, err := t.repo.GetByID(ctx, req.ExecutorID)
	if err != nil {
		return nil, err
	} else if executor_ == nil {
		return nil, errors.New("executor not found")
	}

	newAssignment, err := t.usecase.AssignExecutor(ctx, id, executor_.Name, req.Priority, req.Weight)
	if err != nil {
		return nil, err
	}
	return new(TaskAssignmentResp).FromDomain(newAssignment), nil
}

func (t *TaskAPI) UpdateExecutorAssignment(ctx *gin.Context, id uint64, executorID uint64, req UpdateExecutorAssignmentReq) (*TaskAssignmentResp, error) {
	executor_, err := t.repo.GetByID(ctx, executorID)
	if err != nil {
		return nil, err
	} else if executor_ == nil {
		return nil, errors.New("executor not found")
	}

	assignment, err := t.usecase.UpdateAssignment(ctx, id, executor_.Name, mo.EmptyableToOption(req.Priority), mo.EmptyableToOption(req.Weight))
	if err != nil {
		return nil, err
	}
	return new(TaskAssignmentResp).FromDomain(assignment), nil
}

func (t *TaskAPI) UnassignExecutor(ctx *gin.Context, id uint64, executorID uint64) (string, error) {
	executor_, err := t.repo.GetByID(ctx, executorID)
	if err != nil {
		return "", err
	} else if executor_ == nil {
		return "", errors.New("executor not found")
	}

	err = t.usecase.UnassignExecutor(ctx, id, executor_.Name)
	if err != nil {
		return "", err
	}

	return "executor unassigned", nil
}

func (t *TaskAPI) GetTaskStats(ctx *gin.Context, taskID uint64) (TaskStatsResp, error) {
	// 获取24小时成功率
	var successRate24h float64
	var totalCount24h int64
	var successCount24h int64

	// 计算24小时前的时间
	since24h := time.Now().Add(-24 * time.Hour)

	// 获取24小时内的总执行次数
	if err := t.db.Model(&executionrepo.TaskExecution{}).
		Where("task_id = ? AND created_at >= ?", taskID, since24h).
		Count(&totalCount24h).Error; err != nil {
		return TaskStatsResp{}, errors.Join(err, errors.New("failed to get 24h total count"))
	}

	// 获取24小时内的成功执行次数
	if err := t.db.Model(&executionrepo.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, execution.ExecutionStatusSuccess, since24h).
		Count(&successCount24h).Error; err != nil {
		return TaskStatsResp{}, errors.Join(err, errors.New("failed to get 24h success count"))
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
		t.db.Model(&executionrepo.TaskExecution{}).
			Where("task_id = ? AND created_at >= ? AND created_at < ?",
				taskID, startOfDay, endOfDay).
			Count(&dayTotal)

		// 成功数
		t.db.Model(&executionrepo.TaskExecution{}).
			Where("task_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
				taskID, execution.ExecutionStatusSuccess, startOfDay, endOfDay).
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
		t.db.Model(&executionrepo.TaskExecution{}).
			Where("task_id = ? AND created_at >= ? AND created_at < ?",
				taskID, startOfDay, endOfDay).
			Count(&dayTotal)

		// 成功数
		t.db.Model(&executionrepo.TaskExecution{}).
			Where("task_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
				taskID, execution.ExecutionStatusSuccess, startOfDay, endOfDay).
			Count(&daySuccess)

		// 失败数
		t.db.Model(&executionrepo.TaskExecution{}).
			Where("task_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
				taskID, []string{string(execution.ExecutionStatusFailed), string(execution.ExecutionStatusTimeout)},
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

	return TaskStatsResp{
		SuccessRate24h:   successRate24h,
		Total24h:         totalCount24h,
		Success24h:       successCount24h,
		Health90d:        healthStats90d,
		RecentExecutions: recentExecutions,
		DailyStats90d:    dailyStats,
	}, nil
}

// calculateHealthStats 计算健康度统计
func (t *TaskAPI) calculateHealthStats(taskID uint64, days int) HealthStatus {
	since := time.Now().AddDate(0, 0, -days)

	var totalCount, successCount, failedCount, timeoutCount int64

	// 总执行次数
	t.db.Model(&executionrepo.TaskExecution{}).
		Where("task_id = ? AND created_at >= ?", taskID, since).
		Count(&totalCount)

	// 成功次数
	t.db.Model(&executionrepo.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, execution.ExecutionStatusSuccess, since).
		Count(&successCount)

	// 失败次数
	t.db.Model(&executionrepo.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, execution.ExecutionStatusFailed, since).
		Count(&failedCount)

	// 超时次数
	t.db.Model(&executionrepo.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, execution.ExecutionStatusTimeout, since).
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
	t.db.Model(&executionrepo.TaskExecution{}).
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
