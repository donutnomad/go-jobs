package executor

import (
	"context"

	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

type Repo interface {
	commonrepo.Transaction
	GetByID(ctx context.Context, id uint64) (*Executor, error)
	GetByName(ctx context.Context, name string) (*Executor, error)
	GetByInstanceID(ctx context.Context, instanceID uint64) (*Executor, error)
	Create(ctx context.Context, executor *Executor) error
	Update(ctx context.Context, id uint64, patch *ExecutorPatch) error
	Save(ctx context.Context, executor *Executor) error
	Delete(ctx context.Context, id uint64) error
	List(ctx context.Context, offset, limit int) ([]*Executor, error)

	FindByStatus(ctx context.Context, status []ExecutorStatus) ([]*Executor, error)

	// GetHealthyExecutorsByTask 获取任务的健康执行器
	GetHealthyExecutorsByTask(ctx context.Context, taskID uint64) ([]*Executor, error)
}
