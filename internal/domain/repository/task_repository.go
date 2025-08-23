package repository

import (
	"context"

	"github.com/jobs/scheduler/internal/domain/entity"
)

// TaskRepository 任务仓储接口
type TaskRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, task *entity.Task) error
	GetByID(ctx context.Context, id string) (*entity.Task, error)
	GetByName(ctx context.Context, name string) (*entity.Task, error)
	Update(ctx context.Context, task *entity.Task) error
	Delete(ctx context.Context, id string) error

	// 查询操作
	List(ctx context.Context, filter TaskFilter) ([]*entity.Task, error)
	Count(ctx context.Context, filter TaskFilter) (int64, error)

	// 业务查询
	ListActive(ctx context.Context) ([]*entity.Task, error)
	ListByStatus(ctx context.Context, status entity.TaskStatus) ([]*entity.Task, error)

	// 执行器关联操作
	AssignExecutor(ctx context.Context, taskID, executorName string, priority, weight int) (*entity.TaskExecutor, error)
	UnassignExecutor(ctx context.Context, taskID, executorName string) error
	UpdateAssignment(ctx context.Context, taskID, executorName string, priority, weight int) (*entity.TaskExecutor, error)
	GetTaskExecutors(ctx context.Context, taskID string) ([]*entity.TaskExecutor, error)
}

// TaskFilter 任务查询过滤器
type TaskFilter struct {
	Status entity.TaskStatus
	Name   string
	Limit  int
	Offset int
}
