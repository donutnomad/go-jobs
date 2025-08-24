package api

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/scheduler"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/yitter/idgenerator-go/idgen"
	"go.uber.org/zap"
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
	TriggerTask(ctx *gin.Context, id uint64, req TriggerTaskRequest) (string, error)

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
	emitter       scheduler.IEmitter
	usecase       *task.Usecase
	repo          task.Repo
	executionRepo execution.Repo
	executorRepo  executor.Repo
}

func NewTaskAPI(emitter scheduler.IEmitter, usecase *task.Usecase, repo task.Repo, executionRepo execution.Repo, executorRepo executor.Repo) ITaskAPI {
	return &TaskAPI{
		emitter:       emitter,
		usecase:       usecase,
		repo:          repo,
		executionRepo: executionRepo,
		executorRepo:  executorRepo,
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
	ret := new(TaskWithAssignmentsResp).FromDomain(task_)
	for idx, item := range ret.Assignments {
		name, err := t.executorRepo.GetByName(ctx, item.ExecutorName)
		if err != nil {
			return nil, err
		}
		item.Executor = new(ExecutorResp).FromDomain(name)
		ret.Assignments[idx] = item
	}
	return ret, nil
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

func (t *TaskAPI) TriggerTask(ctx *gin.Context, id uint64, req TriggerTaskRequest) (string, error) {
	task_, err := t.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	} else if task_ == nil {
		return "", errors.New("task not found")
	}
    err = t.emitter.SubmitNewTask(id, req.Parameters)
    if err != nil {
        if errors.Is(err, scheduler.ErrNotLeader) {
            // Not leader: accept request but inform client this instance won't process it
            return "not leader: request accepted; will be handled by leader", nil
        }
        return "", errors.New("failed to submit task to emitter: " + err.Error())
    }
    return "ok", nil
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
	executor_, err := t.executorRepo.GetByID(ctx, req.GetExecutorID())
	if err != nil {
		return nil, err
	} else if executor_ == nil {
		return nil, errors.New("executor not found")
	}

	// executor.is_healthy
	newAssignment, err := t.usecase.AssignExecutor(ctx, id, executor_.Name, req.Priority, req.Weight)
	if err != nil {
		return nil, err
	}
	return new(TaskAssignmentResp).FromDomain(newAssignment), nil
}

func (t *TaskAPI) UpdateExecutorAssignment(ctx *gin.Context, id uint64, executorID uint64, req UpdateExecutorAssignmentReq) (*TaskAssignmentResp, error) {
	executor_, err := t.executorRepo.GetByID(ctx, executorID)
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
	executor_, err := t.executorRepo.GetByID(ctx, executorID)
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
	now := time.Now()

	// 获取24小时内的总执行次数
	var err error
	totalCount24h, err = t.executionRepo.CountByTaskAndTimeRange(ctx, taskID, since24h, now)
	if err != nil {
		return TaskStatsResp{}, errors.Join(err, errors.New("failed to get 24h total count"))
	}

	// 获取24小时内的成功执行次数
	successCount24h, err = t.executionRepo.CountByTaskStatusAndTimeRange(ctx, taskID, execution.ExecutionStatusSuccess, since24h, now)
	if err != nil {
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

		// 总数
		dayTotal, err := t.executionRepo.CountByTaskAndTimeRange(ctx, taskID, startOfDay, endOfDay)
		if err != nil {
			dayTotal = 0
		}

		// 成功数
		daySuccess, err := t.executionRepo.CountByTaskStatusAndTimeRange(ctx, taskID, execution.ExecutionStatusSuccess, startOfDay, endOfDay)
		if err != nil {
			daySuccess = 0
		}

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

		// 总数
		dayTotal, err := t.executionRepo.CountByTaskAndTimeRange(ctx, taskID, startOfDay, endOfDay)
		if err != nil {
			dayTotal = 0
		}

		// 成功数
		daySuccess, err := t.executionRepo.CountByTaskStatusAndTimeRange(ctx, taskID, execution.ExecutionStatusSuccess, startOfDay, endOfDay)
		if err != nil {
			daySuccess = 0
		}

		// 失败数
		dayFailed, err := t.executionRepo.CountByTaskStatusesAndTimeRange(ctx, taskID,
			[]execution.ExecutionStatus{execution.ExecutionStatusFailed, execution.ExecutionStatusTimeout},
			startOfDay, endOfDay)
		if err != nil {
			dayFailed = 0
		}

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
	now := time.Now()
	ctx := context.Background()

	// 总执行次数
	totalCount, err := t.executionRepo.CountByTaskAndTimeRange(ctx, taskID, since, now)
	if err != nil {
		totalCount = 0
	}

	// 定义要统计的状态及其对应的计数变量
	statusCounts := map[execution.ExecutionStatus]*int64{
		execution.ExecutionStatusSuccess: new(int64),
		execution.ExecutionStatusFailed:  new(int64),
		execution.ExecutionStatusTimeout: new(int64),
	}

	// 批量统计各状态的执行次数
	for status, countPtr := range statusCounts {
		count, err := t.executionRepo.CountByTaskStatusAndTimeRange(ctx, taskID, status, since, now)
		if err != nil {
			count = 0
		}
		*countPtr = count
	}

	successCount := *statusCounts[execution.ExecutionStatusSuccess]
	failedCount := *statusCounts[execution.ExecutionStatusFailed]
	timeoutCount := *statusCounts[execution.ExecutionStatusTimeout]

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
	avgDuration, err := t.executionRepo.GetAvgDuration(ctx, taskID, since)
	if err != nil {
		avgDuration = 0
	}

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
