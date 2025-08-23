package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IExecutionAPI interface {
	// List 获取执行历史列表
	// 获取所有的执行历史列表
	// @GET(api/v1/executions)
	List(ctx *gin.Context, req ListExecutionReq) (ListExecutionResp, error)

	// Get 获取执行历史详情
	// 获取指定id的执行历史详情
	// @GET(api/v1/executions/{id})
	Get(ctx *gin.Context, id uint64) (*TaskExecutionResp, error)

	// Stats 获取执行统计
	// 获取指定任务的执行统计
	// @GET(api/v1/executions/stats)
	Stats(ctx *gin.Context, req ExecutionStatsReq) (*ExecutionStatsResp, error)

	// Callback 执行回调
	// 执行指定id的执行回调
	// @POST(api/v1/executions/{id}/callback)
	Callback(ctx *gin.Context, id uint64, req ExecutionCallbackRequest) (string, error)

	// Stop 停止执行
	// 停止指定id的执行
	// @POST(api/v1/executions/{id}/stop)
	Stop(ctx *gin.Context, id uint64) (string, error)
}

func ExecutionCallbackURL(listenAddr string, isTLS bool) func(uint64) string {
	return func(id uint64) string {
		var prefix = "http"
		if isTLS {
			prefix = "https"
		}
		return fmt.Sprintf("%s://%s/api/v1/executions/%d/callback", prefix, listenAddr, id)
	}
}

type ExecutionAPI struct {
	db      *gorm.DB
	logger  *zap.Logger
	emitter IEmitter

	executionRepo execution.Repo
	taskRepo      task.Repo
	executorRepo  executor.Repo
}

func NewExecutionAPI(db *gorm.DB, logger *zap.Logger, emitter IEmitter, executionRepo execution.Repo, taskRepo task.Repo, executorRepo executor.Repo) *ExecutionAPI {
	return &ExecutionAPI{db: db, logger: logger, emitter: emitter, executionRepo: executionRepo, taskRepo: taskRepo, executorRepo: executorRepo}
}

func (e *ExecutionAPI) Get(ctx *gin.Context, id uint64) (*TaskExecutionResp, error) {
	execution_, err := e.executionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	} else if execution_ == nil {
		return nil, errors.New("execution not found")
	}

	out := new(TaskExecutionResp).FromDomain(execution_)

	task_, err := e.taskRepo.GetByID(ctx, execution_.TaskID)
	if err != nil {
		return nil, err
	}
	if (task_) != nil {
		out.Task = new(TaskResp).FromDomain(task_)
	}

	if execution_.ExecutorID > 0 {
		executor_, err := e.executorRepo.GetByID(ctx, execution_.ExecutorID)
		if err != nil {
			return nil, err
		}
		out.Executor = new(ExecutorResp).FromDomain(executor_)
	}

	return out, nil
}

func (e *ExecutionAPI) Callback(ctx *gin.Context, id uint64, req ExecutionCallbackRequest) (string, error) {
	execution_, err := e.executionRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	} else if execution_ == nil {
		return "", errors.New("execution not found")
	}

	// 更新执行状态
	execution_.Status = req.Status
	execution_.EndTime = lo.ToPtr(time.Now())
	execution_.Result = req.Result
	execution_.Logs = req.Logs

	if err := e.executionRepo.Save(ctx, execution_); err != nil {
		return "", fmt.Errorf("failed to update execution: %w", err)
	}

	_ = e.emitter.CancelExecutionTimer(id)

	return "callback processed", nil
}

func (e *ExecutionAPI) Stop(ctx *gin.Context, id uint64) (string, error) {
	record, err := e.executionRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	} else if record == nil {
		return "", errors.New("execution not found")
	}
	if record.Status != execution.ExecutionStatusRunning {
		return "", errors.New("execution is not running")
	}
	record.Status = execution.ExecutionStatusCancelled

	if record.ExecutorID > 0 {
		executor_, err := e.executorRepo.GetByID(ctx, record.ExecutorID)
		if err != nil {
			return "", err
		}
		stopReq := map[string]any{
			"execution_id": id,
		}
		resp, err := http.Post(executor_.GetStopURL(), "application/json", bytes.NewBuffer(mustMarshal(stopReq)))
		if err != nil {
			e.logger.Error("failed to call executor stop endpoint",
				zap.Uint64("execution_id", id),
				zap.Uint64("executor_id", record.ExecutorID),
				zap.Error(err))
			return "", errors.Join(err, errors.New("failed to stop execution on executor"))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errorResp map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&errorResp)
			e.logger.Error("executor stop endpoint returned error",
				zap.Uint64("execution_id", id),
				zap.Int("status_code", resp.StatusCode),
				zap.Any("error", errorResp))
		}
	}

	if err := e.executionRepo.Save(ctx, record); err != nil {
		return "", errors.Join(err, errors.New("failed to update execution status"))
	}
	return "stop request sent to executor", nil
}

func (e *ExecutionAPI) Stats(ctx *gin.Context, req ExecutionStatsReq) (*ExecutionStatsResp, error) {
	var stats ExecutionStatsResp
	var err error

	countQuery := execution.CountQuery{}
	countQuery.StartTime = mo.EmptyableToOption(req.StartTime)
	countQuery.EndTime = mo.EmptyableToOption(req.EndTime)
	countQuery.TaskID = mo.EmptyableToOption(req.TaskID)

	countQuery.Status = mo.Some(execution.ExecutionStatusSuccess)
	stats.Success, err = e.executionRepo.Count(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	countQuery.Status = mo.Some(execution.ExecutionStatusFailed)
	stats.Failed, err = e.executionRepo.Count(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	countQuery.Status = mo.Some(execution.ExecutionStatusRunning)
	stats.Running, err = e.executionRepo.Count(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	countQuery.Status = mo.Some(execution.ExecutionStatusPending)
	stats.Pending, err = e.executionRepo.Count(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	return &stats, nil
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

	executions, total, err := e.executionRepo.List(ctx, execution.ListFilter{
		StartTime: mo.Some(req.StartTime),
		EndTime:   mo.Some(req.EndTime),
		TaskID:    mo.Some(req.TaskID),
		Status:    mo.Some(req.Status),
	}, offset, pageSize)
	if err != nil {
		return ListExecutionResp{}, err
	}

	var outList []*TaskExecutionResp
	for _, execution_ := range executions {
		out := new(TaskExecutionResp).FromDomain(execution_)
		task_, _ := e.taskRepo.GetByID(ctx, execution_.TaskID)
		if task_ != nil {
			out.Task = new(TaskResp).FromDomain(task_)
		}
		if execution_.ExecutorID > 0 {
			executor_, _ := e.executorRepo.GetByID(ctx, execution_.ExecutorID)
			if executor_ != nil {
				out.Executor = new(ExecutorResp).FromDomain(executor_)
			}
		}
		outList = append(outList, out)
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return ListExecutionResp{
		Data:       outList,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func mustMarshal(t any) []byte {
	jsonData, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return jsonData
}
