package api

import (
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/executor"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IExecutorAPI interface {
	// List 获取执行器列表
	// 获取所有的执行器列表
	// @GET(api/v1/executors)
	List(ctx *gin.Context, req ListExecutorRequest) ([]*models.Executor, error)

	// Get 获取执行器详情
	// 获取指定id的执行器详情
	// @GET(api/v1/executors/{id})
	Get(ctx *gin.Context, id string) (*models.Executor, error)

	// Register 注册执行器
	// 注册一个新执行器
	// @POST(api/v1/executors/register)
	Register(ctx *gin.Context, req executor.RegisterRequest) (*models.Executor, error)

	// Update 更新执行器
	// 更新指定id的执行器
	// @PUT(api/v1/executors/{id})
	Update(ctx *gin.Context, id string, req UpdateExecutorRequest) (models.Executor, error)

	// UpdateStatus 更新执行器状态
	// 更新指定id的执行器状态
	// @PUT(api/v1/executors/{id}/status)
	UpdateStatus(ctx *gin.Context, id string, req executor.UpdateStatusRequest) (string, error)

	// Delete 删除执行器
	// 删除指定id的执行器
	// @DELETE(api/v1/executors/{id})
	Delete(ctx *gin.Context, id string) (string, error)
}

var _ IExecutorAPI = (*ExecutorAPI)(nil)

type ExecutorAPI struct {
	storage         *orm.Storage
	executorManager *executor.Manager
	logger          *zap.Logger
}

func NewExecutorAPI(storage *orm.Storage, executorManager *executor.Manager, logger *zap.Logger) *ExecutorAPI {
	return &ExecutorAPI{storage: storage, executorManager: executorManager, logger: logger}
}

func (e *ExecutorAPI) List(ctx *gin.Context, req ListExecutorRequest) ([]*models.Executor, error) {
	executors, err := e.executorManager.ListExecutors(ctx.Request.Context())
	if err != nil {
		return nil, err
	}

	// 如果需要包含任务信息，为每个执行器加载关联的任务
	if req.IncludeTasks {
		for _, exe := range executors {
			var taskExecutors []models.TaskExecutor
			err := e.storage.DB().
				Preload("Task").
				Where("executor_id = ?", exe.ID).
				Find(&taskExecutors).Error
			if err != nil {
				e.logger.Error("failed to load task executors",
					zap.String("executor_id", exe.ID),
					zap.Error(err))
				continue
			}
			exe.TaskExecutors = taskExecutors
		}
	}
	sort.Slice(executors, func(i, j int) bool {
		return executors[i].Status.ToInt() < executors[j].Status.ToInt()
	})

	return executors, nil
}

func (e *ExecutorAPI) Get(ctx *gin.Context, id string) (*models.Executor, error) {
	return e.executorManager.GetExecutorByID(ctx.Request.Context(), id)
}

func (e *ExecutorAPI) Register(ctx *gin.Context, req executor.RegisterRequest) (*models.Executor, error) {
	return e.executorManager.RegisterExecutor(ctx.Request.Context(), req)
}

func (e *ExecutorAPI) Update(ctx *gin.Context, id string, req UpdateExecutorRequest) (models.Executor, error) {
	// 查找执行器
	var ret models.Executor
	if err := e.storage.DB().Where("id = ?", id).First(&ret).Error; err != nil {
		return models.Executor{}, err
	}

	// 更新字段
	if req.Name != "" {
		ret.Name = req.Name
	}
	if req.BaseURL != "" {
		ret.BaseURL = req.BaseURL
	}
	if req.HealthCheckURL != "" {
		ret.HealthCheckURL = req.HealthCheckURL
	}

	// 保存更新
	if err := e.storage.DB().Save(&ret).Error; err != nil {
		return models.Executor{}, err
	}

	return ret, nil
}

func (e *ExecutorAPI) UpdateStatus(ctx *gin.Context, id string, req executor.UpdateStatusRequest) (string, error) {
	err := e.executorManager.UpdateExecutorStatus(ctx.Request.Context(), id, req.Status, req.Reason)
	if err != nil {
		return "", err
	}
	return "status updated", nil
}

func (e *ExecutorAPI) Delete(ctx *gin.Context, id string) (string, error) {
	if err := e.storage.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("executor_id = ?", id).Delete(&models.TaskExecutor{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", id).Delete(&models.Executor{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}
	return "executor deleted", nil
}
