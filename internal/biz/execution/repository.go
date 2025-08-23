package execution

import (
	"context"

	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"github.com/samber/mo"
)

type Repo interface {
	commonrepo.Transaction
	Create(ctx context.Context, execution *TaskExecution) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*TaskExecution, error)
	Save(ctx context.Context, execution *TaskExecution) error

	Count(ctx context.Context, query CountQuery) (int64, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]*TaskExecution, int64, error)
	
	// CountByTaskAndStatus 按任务ID和状态统计执行记录数量
	CountByTaskAndStatus(ctx context.Context, taskID uint64, statuses []ExecutionStatus) (int64, error)
	
	// CountByExecutorAndStatus 按执行器ID和状态统计执行记录数量
	CountByExecutorAndStatus(ctx context.Context, executorID uint64, statuses []ExecutionStatus) (int64, error)
	
	// CreateSkipped 创建跳过的执行记录
	CreateSkipped(ctx context.Context, taskID uint64, reason string) (*TaskExecution, error)
}

type ListFilter struct {
	StartTime mo.Option[int64]
	EndTime   mo.Option[int64]
	TaskID    mo.Option[uint64]
	Status    mo.Option[ExecutionStatus]
}

type CountQuery struct {
	StartTime mo.Option[int64]
	EndTime   mo.Option[int64]
	TaskID    mo.Option[uint64]
	Status    mo.Option[ExecutionStatus]
}
