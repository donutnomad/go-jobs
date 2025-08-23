package repository

import (
	"context"

	"github.com/jobs/scheduler/internal/domain/entity"
)

// ExecutorRepository 执行器仓储接口
type ExecutorRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, executor *entity.Executor) error
	GetByID(ctx context.Context, id string) (*entity.Executor, error)
	GetByInstanceID(ctx context.Context, instanceID string) (*entity.Executor, error)
	GetByName(ctx context.Context, name string) (*entity.Executor, error)
	Update(ctx context.Context, executor *entity.Executor) error
	Delete(ctx context.Context, id string) error

	// 查询操作
	List(ctx context.Context, filter ExecutorFilter) ([]*entity.Executor, error)
	Count(ctx context.Context, filter ExecutorFilter) (int64, error)

	// 业务查询
	ListOnline(ctx context.Context) ([]*entity.Executor, error)
	ListHealthy(ctx context.Context) ([]*entity.Executor, error)
	GetExecutorsForTask(ctx context.Context, taskID string) ([]*entity.Executor, error)
}

// ExecutorFilter 执行器查询过滤器
type ExecutorFilter struct {
	Name         string
	Status       entity.ExecutorStatus
	IsHealthy    *bool
	IncludeTasks bool
	Limit        int
	Offset       int
}
