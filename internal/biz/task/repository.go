package task

import (
	"context"

	"github.com/samber/mo"
)

type Repo interface {
	Create(ctx context.Context, task *Task) error
	GetByName(ctx context.Context, name string) (*Task, error)
	GetByID(ctx context.Context, id uint64) (*Task, error)
	Delete(ctx context.Context, id uint64) error
	Update(ctx context.Context, id uint64, patch *TaskPatch) error
	List(ctx context.Context, filter *TaskFilter) ([]*Task, error)
	
	// FindActiveTasks 查找所有活跃任务
	FindActiveTasks(ctx context.Context) ([]*Task, error)

	// FindByIDWithAssignments 获取任务及其关联的执行器
	FindByIDWithAssignments(ctx context.Context, id uint64) (*Task, error)
	ListWithAssignments(ctx context.Context, filter *TaskFilter) ([]*Task, error)

	CreateAssignment(ctx context.Context, assignment *TaskAssignment) error
	DeleteAssignment(ctx context.Context, id uint64) error
	DeleteAssignmentsByExecutorName(ctx context.Context, executorName string) error
	UpdateAssignment(ctx context.Context, id uint64, patch *TaskAssignmentPatch) error
	ListAssignmentsWithExecutor(ctx context.Context, executorName string) ([]*TaskAssignment, error)
	GetAssignmentByTaskIDAndExecutorName(ctx context.Context, taskID uint64, executorName string) (*TaskAssignment, error)
}

type TaskFilter struct {
	Status mo.Option[TaskStatus]
}
