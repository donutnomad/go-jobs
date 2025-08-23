package api

import (
	"bytes"
	"context"
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
	return e.loadFrom(ctx, execution_), nil
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

	return "ok", nil
}

func (e *ExecutionAPI) Stop(ctx *gin.Context, id uint64) (string, error) {
	var msg = "ok"

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
			msg = fmt.Sprintf("executor stop endpoint returned status %d: %v", resp.StatusCode, errorResp)
			e.logger.Error("executor stop endpoint returned error",
				zap.Uint64("execution_id", id),
				zap.Int("status_code", resp.StatusCode),
				zap.Any("error", errorResp))
		}
	}

	if err := e.executionRepo.Save(ctx, record); err != nil {
		return "", errors.Join(err, errors.New("failed to update execution status"))
	}
	return msg, nil
}

func (e *ExecutionAPI) Stats(ctx *gin.Context, req ExecutionStatsReq) (*ExecutionStatsResp, error) {
	countQuery := execution.CountQuery{
		StartTime: mo.EmptyableToOption(req.StartTime),
		EndTime:   mo.EmptyableToOption(req.EndTime),
		TaskID:    mo.EmptyableToOption(req.TaskID),
	}

	var stats ExecutionStatsResp
	for _, item := range []struct {
		Status execution.ExecutionStatus
		Count  *int64
	}{
		{
			execution.ExecutionStatusSuccess,
			&stats.Success,
		},
		{
			execution.ExecutionStatusFailed,
			&stats.Failed,
		},
		{
			execution.ExecutionStatusPending,
			&stats.Pending,
		},
		{
			execution.ExecutionStatusRunning,
			&stats.Running,
		},
	} {
		countQuery.Status = mo.Some(item.Status)
		count, err := e.executionRepo.Count(ctx, countQuery)
		if err != nil {
			return nil, err
		}
		*item.Count = count
	}

	return &stats, nil
}

func (e *ExecutionAPI) List(ctx *gin.Context, req ListExecutionReq) (ListExecutionResp, error) {
	executions, total, err := e.executionRepo.List(ctx, execution.ListFilter{
		StartTime: mo.Some(req.StartTime),
		EndTime:   mo.Some(req.EndTime),
		TaskID:    mo.Some(req.TaskID),
		Status:    mo.Some(req.Status),
	}, req.GetOffset(), req.GetLimit())
	if err != nil {
		return ListExecutionResp{}, err
	}

	outList := lo.Map(executions, func(item *execution.TaskExecution, index int) *TaskExecutionResp {
		return e.loadFrom(ctx, item)
	})

	return ListExecutionResp{
		Data:       outList,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.GetLimit(),
		TotalPages: req.GetTotalPages(total),
	}, nil
}

func (e *ExecutionAPI) loadFrom(ctx context.Context, input *execution.TaskExecution) *TaskExecutionResp {
	out := new(TaskExecutionResp).FromDomain(input)
	task_, _ := e.taskRepo.GetByID(ctx, input.TaskID)
	if task_ != nil {
		out.Task = new(TaskResp).FromDomain(task_)
	}
	if input.ExecutorID > 0 {
		executor_, _ := e.executorRepo.GetByID(ctx, input.ExecutorID)
		if executor_ != nil {
			out.Executor = new(ExecutorResp).FromDomain(executor_)
		}
	}
	return out
}

func mustMarshal(t any) []byte {
	jsonData, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return jsonData
}
