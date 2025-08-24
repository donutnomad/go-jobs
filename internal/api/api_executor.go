package api

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/cast"
	"github.com/yitter/idgenerator-go/idgen"
	"go.uber.org/zap"
)

type IExecutorAPI interface {
	// List 获取执行器列表
	// 获取所有的执行器列表
	// @GET(api/v1/executors)
	List(ctx *gin.Context, req ListExecutorReq) ([]*ExecutorResp, error)

	// Get 获取执行器详情
	// 获取指定id的执行器详情
	// @GET(api/v1/executors/{id})
	Get(ctx *gin.Context, id uint64) (*ExecutorResp, error)

	// Update 更新执行器
	// 更新指定id的执行器
	// @PUT(api/v1/executors/{id})
	Update(ctx *gin.Context, id uint64, req UpdateExecutorReq) (*ExecutorResp, error)

	// UpdateStatus 更新执行器状态
	// 更新指定id的执行器状态
	// @PUT(api/v1/executors/{id}/status)
	UpdateStatus(ctx *gin.Context, id uint64, req UpdateExecutorStatusReq) (string, error)

	// Delete 删除执行器
	// 删除指定id的执行器
	// @DELETE(api/v1/executors/{id})
	Delete(ctx *gin.Context, id uint64) (string, error)

	// Register 注册执行器
	// 注册一个新执行器
	// @POST(api/v1/executors/register)
	Register(ctx *gin.Context, req RegisterExecutorReq) (*ExecutorResp, error)
}

type ExecutorAPI struct {
	logger *zap.Logger
	mu     sync.RWMutex

	usecase      *executor.Usecase
	executorRepo executor.Repo
	taskRepo     task.Repo
}

func NewExecutorAPI(
	logger *zap.Logger,
	executorRepo executor.Repo,
	taskRepo task.Repo,
	usecase *executor.Usecase,
) IExecutorAPI {
	return &ExecutorAPI{
		logger:       logger,
		mu:           sync.RWMutex{},
		executorRepo: executorRepo,
		taskRepo:     taskRepo,
		usecase:      usecase,
	}
}

func (e *ExecutorAPI) Get(ctx *gin.Context, id uint64) (*ExecutorResp, error) {
	exec, err := e.executorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	} else if exec == nil {
		return nil, fmt.Errorf("executor not found")
	}
	ret := new(ExecutorResp).FromDomain(exec)

	assignments, err := e.taskRepo.ListAssignmentsWithExecutor(ctx, exec.Name)
	if err != nil {
		return nil, err
	}
	ret.TaskAssignments = lo.Map(assignments, func(assignment *task.TaskAssignment, _ int) *TaskAssignmentResp2 {
		return new(TaskAssignmentResp2).FromDomain(assignment)
	})

	return ret, nil
}

func (e *ExecutorAPI) List(ctx *gin.Context, req ListExecutorReq) ([]*ExecutorResp, error) {
	executors, err := e.executorRepo.List(ctx, 0, 10_0000)
	if err != nil {
		return nil, err
	}

	var ret []*ExecutorResp
	for _, exe := range executors {
		taskAssignments, err := e.taskRepo.ListAssignmentsWithExecutor(ctx, exe.Name)
		if err != nil {
			return nil, err
		}
		exeRet := new(ExecutorResp).FromDomain(exe)
		for _, assignment := range taskAssignments {
			assignment2 := new(TaskAssignmentResp2).FromDomain(assignment)
			task_, err := e.taskRepo.GetByID(ctx, assignment.TaskID)
			if err != nil {
				return nil, err
			}
			assignment2.Task = new(TaskResp).FromDomain(task_)
			exeRet.TaskAssignments = append(exeRet.TaskAssignments, assignment2)
		}
		ret = append(ret, exeRet)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Status.ToInt() < ret[j].Status.ToInt()
	})
	return ret, nil
}

func (e *ExecutorAPI) Update(ctx *gin.Context, id uint64, req UpdateExecutorReq) (*ExecutorResp, error) {
	exec, err := e.usecase.Update(ctx, id, &executor.ExecutorPatch{
		Name:           mo.Some(req.Name).ToPointer(),
		BaseURL:        mo.Some(req.BaseURL).ToPointer(),
		HealthCheckURL: mo.Some(req.HealthCheckURL).ToPointer(),
	})
	if err != nil {
		return nil, err
	}
	return new(ExecutorResp).FromDomain(exec), nil
}

func (e *ExecutorAPI) UpdateStatus(ctx *gin.Context, id uint64, req UpdateExecutorStatusReq) (string, error) {
	_, err := e.usecase.UpdateStatus(ctx, id, req.Status)
	if err != nil {
		return "", err
	}
	return "status updated", nil
}

func (e *ExecutorAPI) Delete(ctx *gin.Context, id uint64) (string, error) {
	err := e.executorRepo.Execute(ctx, func(ctx context.Context) error {
		exec, err := e.executorRepo.GetByID(ctx, id)
		if err != nil {
			return err
		} else if exec == nil {
			return fmt.Errorf("executor not found")
		}
		if err := e.taskRepo.DeleteAssignmentsByExecutorName(ctx, exec.Name); err != nil {
			return err
		}
		if err := e.executorRepo.Delete(ctx, id); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return "executor deleted", nil
}

func (e *ExecutorAPI) Register(ctx *gin.Context, req RegisterExecutorReq) (*ExecutorResp, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 验证请求数据
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var exec *executor.Executor

	err := e.executorRepo.Execute(ctx, func(ctx context.Context) error {
		var err error

		if req.NameOnly {
			// 仅名称模式：检查是否已存在同名执行器
			existingExec, err := e.executorRepo.GetByName(ctx, req.ExecutorName)
			if err != nil {
				return err
			}

			if existingExec != nil {
				return fmt.Errorf("executor with name %s already exists", req.ExecutorName)
			}

			// 创建仅名称执行器（离线状态）
			exec = &executor.Executor{
				ID:                  uint64(idgen.NextId()),
				Name:                req.ExecutorName,
				InstanceID:          "", // 仅名称模式下为空
				BaseURL:             "", // 仅名称模式下为空
				HealthCheckURL:      "", // 仅名称模式下为空
				CreatedAt:           time.Time{},
				UpdatedAt:           time.Time{},
				Status:              executor.ExecutorStatusOffline, // 默认离线状态
				IsHealthy:           false,
				LastHealthCheck:     nil,
				HealthCheckFailures: 0,
				Metadata:            req.Metadata,
			}

			if err := e.executorRepo.Create(ctx, exec); err != nil {
				return fmt.Errorf("failed to create executor: %w", err)
			}
		} else {
			// 完整模式：原有逻辑
			exec, err = e.executorRepo.GetByID(ctx, cast.ToUint64(req.ExecutorID))
			if err != nil {
				return err
			}
			if exec != nil {
				// 如果执行器已存在且在线，拒绝注册（防止挤掉别人）
				if exec.Status == executor.ExecutorStatusOnline {
					// 检查是否是同一个执行器重新注册（通过 BaseURL 判断）
					if exec.BaseURL != req.ExecutorURL {
						return fmt.Errorf("executor with instance_id %s is already online from different location (current: %s, new: %s)",
							req.ExecutorID, exec.BaseURL, req.ExecutorURL)
					}
				}

				// 更新现有执行器信息
				exec.Name = req.ExecutorName
				exec.InstanceID = req.ExecutorID
				exec.BaseURL = req.ExecutorURL
				exec.HealthCheckURL = req.HealthCheckURL
				exec.Status = executor.ExecutorStatusOnline
				exec.IsHealthy = true
				exec.HealthCheckFailures = 0
				exec.LastHealthCheck = lo.ToPtr(time.Now())

				if err := e.executorRepo.Save(ctx, exec); err != nil {
					return fmt.Errorf("failed to update executor: %w", err)
				}
			} else {
				// 不存在
				exec = &executor.Executor{
					ID:                  uint64(idgen.NextId()),
					Name:                req.ExecutorName,
					InstanceID:          req.ExecutorID,
					BaseURL:             req.ExecutorURL,
					HealthCheckURL:      req.HealthCheckURL,
					CreatedAt:           time.Time{},
					UpdatedAt:           time.Time{},
					Status:              executor.ExecutorStatusOnline,
					IsHealthy:           true,
					LastHealthCheck:     lo.ToPtr(time.Now()),
					HealthCheckFailures: 0,
					Metadata:            req.Metadata,
				}
				if err := e.executorRepo.Create(ctx, exec); err != nil {
					return fmt.Errorf("failed to create executor: %w", err)
				}
			}
		}

		// 注册任务（两种模式都支持）
		for _, def := range req.Tasks {
			if err := e.registerTask(ctx, exec.Name, def); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return new(ExecutorResp).FromDomain(exec), nil
}

func (e *ExecutorAPI) registerTask(ctx context.Context, executorName string, taskDef TaskDefinition) error {
	task_, err := e.taskRepo.GetByName(ctx, taskDef.Name)
	if err != nil {
		return err
	}
	if task_ == nil {
		task_ = &task.Task{
			ID:                  uint64(idgen.NextId()),
			CreatedAt:           time.Time{},
			UpdatedAt:           time.Time{},
			Name:                taskDef.Name,
			CronExpression:      taskDef.CronExpression,
			ExecutionMode:       taskDef.GetExecutionMode(),
			LoadBalanceStrategy: taskDef.GetLoadBalanceStrategy(),
			MaxRetry:            taskDef.GetMaxRetry(),
			TimeoutSeconds:      taskDef.GetTimeoutSeconds(),
			Status:              taskDef.GetStatus(),
			Parameters:          taskDef.GetParameters(),
			Assignments:         nil,
		}
		if err := e.taskRepo.Create(ctx, task_); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
		return nil
	}

	assignment, err := e.taskRepo.GetAssignmentByTaskIDAndExecutorName(ctx, task_.ID, executorName)
	if err != nil {
		return err
	} else if assignment != nil {
		return nil
	}
	return e.taskRepo.CreateAssignment(ctx, &task.TaskAssignment{
		ID:           uint64(idgen.NextId()),
		TaskID:       task_.ID,
		ExecutorName: executorName,
		Priority:     1, // 默认优先级
		Weight:       1, // 默认权重
	})
}
