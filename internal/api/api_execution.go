package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/internal/scheduler"
	"go.uber.org/zap"
)

type IExecutionAPI interface {
	// List 获取执行历史列表
	// 获取所有的执行历史列表
	// @GET(api/v1/executions)
	List(ctx *gin.Context, req ListExecutionReq) (ListExecutionResp, error)

	// Get 获取执行历史详情
	// 获取指定id的执行历史详情
	// @GET(api/v1/executions/{id})
	Get(ctx *gin.Context, id string) (*models.TaskExecution, error)

	// Stats 获取执行统计
	// 获取指定任务的执行统计
	// @GET(api/v1/executions/stats)
	Stats(ctx *gin.Context, req StatsRequest) (ExecutionStatsResp, error)

	// Callback 执行回调
	// 执行指定id的执行回调
	// @POST(api/v1/executions/{id}/callback)
	Callback(ctx *gin.Context, id string, req scheduler.ExecutionCallbackRequest) (string, error)

	// Stop 停止执行
	// 停止指定id的执行
	// @POST(api/v1/executions/{id}/stop)
	Stop(ctx *gin.Context, id string) (string, error)
}

type ExecutionAPI struct {
	storage    *orm.Storage
	taskRunner *scheduler.TaskRunner
	logger     *zap.Logger
}

func NewExecutionAPI(storage *orm.Storage, taskRunner *scheduler.TaskRunner, logger *zap.Logger) IExecutionAPI {
	return &ExecutionAPI{
		storage:    storage,
		taskRunner: taskRunner,
		logger:     logger,
	}
}

type StatsRequest struct {
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	TaskID    string `form:"task_id"`
}

func (e *ExecutionAPI) Stats(ctx *gin.Context, req StatsRequest) (ExecutionStatsResp, error) {
	var stats ExecutionStatsResp

	query := e.storage.DB().Model(&models.TaskExecution{})
	if start := req.StartTime; start != "" {
		query = query.Where("scheduled_time >= ?", start)
	}
	if end := req.EndTime; end != "" {
		query = query.Where("scheduled_time <= ?", end)
	}
	if taskID := req.TaskID; taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}

	// 统计总数
	if err := query.Count(&stats.Total).Error; err != nil {
		return ExecutionStatsResp{}, err
	}

	// 统计各状态数量
	query.Where("status = ?", models.ExecutionStatusSuccess).Count(&stats.Success)
	query.Where("status = ?", models.ExecutionStatusFailed).Count(&stats.Failed)
	query.Where("status = ?", models.ExecutionStatusRunning).Count(&stats.Running)
	query.Where("status = ?", models.ExecutionStatusPending).Count(&stats.Pending)

	return stats, nil
}

func (e *ExecutionAPI) List(ctx *gin.Context, req ListExecutionReq) (ListExecutionResp, error) {
	// 分页参数
	page := max(1, req.Page)
	pageSize := 20 // 默认每页20条
	if req.PageSize != 0 {
		pageSize = req.PageSize
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	var executions []models.TaskExecution
	query := e.storage.DB().Model(&models.TaskExecution{})

	// 支持任务ID过滤
	if taskID := req.TaskID; taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}
	// 支持状态过滤
	if status := req.Status; status != "" {
		query = query.Where("status = ?", status)
	}
	// 支持时间范围过滤
	if start := req.StartTime; start != "" {
		query = query.Where("scheduled_time >= ?", start)
	}
	if end := req.EndTime; end != "" {
		query = query.Where("scheduled_time <= ?", end)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return ListExecutionResp{}, err
	}

	// 查询数据
	query = query.Preload("Task").Preload("Executor")
	if err := query.Order("scheduled_time DESC").Limit(pageSize).Offset(offset).Find(&executions).Error; err != nil {
		return ListExecutionResp{}, err
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return ListExecutionResp{
		Data:       executions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (e *ExecutionAPI) Get(ctx *gin.Context, id string) (*models.TaskExecution, error) {
	var execution models.TaskExecution
	if err := e.storage.DB().
		Preload("Task").
		Preload("Executor").
		Where("id = ?", id).
		First(&execution).Error; err != nil {
		return nil, err
	}

	return &execution, nil
}

func (e *ExecutionAPI) Callback(ctx *gin.Context, id string, req scheduler.ExecutionCallbackRequest) (string, error) {
	err := e.taskRunner.HandleCallback(ctx.Request.Context(), id, req)
	if err != nil {
		return "", err
	}
	return "callback processed", nil
}

func (e *ExecutionAPI) Stop(ctx *gin.Context, id string) (string, error) {
	var record models.TaskExecution
	if err := e.storage.DB().
		Preload("Executor").
		Where("id = ?", id).
		First(&record).Error; err != nil {
		return "", err
	}
	if record.Status != models.ExecutionStatusRunning {
		return "", errors.New("execution is not running")
	}

	// 调用执行器的停止接口
	if record.Executor != nil {
		stopReq := map[string]string{
			"execution_id": id,
		}
		resp, err := http.Post(record.Executor.GetStopURL(), "application/json", bytes.NewBuffer(mustMarshal(stopReq)))
		if err != nil {
			e.logger.Error("failed to call executor stop endpoint",
				zap.String("execution_id", id),
				zap.String("executor_id", *record.ExecutorID),
				zap.Error(err))
			return "", errors.Join(err, errors.New("failed to stop execution on executor"))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errorResp map[string]any
			json.NewDecoder(resp.Body).Decode(&errorResp)
			e.logger.Error("executor stop endpoint returned error",
				zap.String("execution_id", id),
				zap.Int("status_code", resp.StatusCode),
				zap.Any("error", errorResp))
		}
	}

	// 更新执行状态为取消中
	record.Status = models.ExecutionStatusCancelled
	if err := e.storage.DB().Save(&record).Error; err != nil {
		return "", errors.Join(err, errors.New("failed to update execution status"))
	}

	return "stop request sent to executor", nil
}

func mustMarshal(t any) []byte {
	jsonData, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return jsonData
}

func mustMarshalString(t any) string {
	jsonData, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return string(jsonData)
}
