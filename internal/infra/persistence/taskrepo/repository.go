package taskrepo

import (
	"context"
	"errors"

	"github.com/google/wire"
	domain "github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

var Provider = wire.NewSet(NewMysqlRepositoryImpl)

type MysqlRepositoryImpl struct {
	commonrepo.DefaultRepo
}

func NewMysqlRepositoryImpl(db commonrepo.DB) domain.Repo {
	return &MysqlRepositoryImpl{DefaultRepo: commonrepo.NewDefaultRepo(db)}
}

func (r *MysqlRepositoryImpl) Create(ctx context.Context, task *domain.Task) error {
	po := new(TaskPo).FromDomain(task)
	err := r.Db(ctx).Create(po).Error
	if err != nil {
		return err
	}
	task.ID = po.ID
	task.CreatedAt = po.CreatedAt
	task.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *MysqlRepositoryImpl) GetByName(ctx context.Context, name string) (*domain.Task, error) {
	var po TaskPo
	if err := r.Db(ctx).Where("name = ?", name).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *MysqlRepositoryImpl) GetByID(ctx context.Context, id uint64) (*domain.Task, error) {
	var po TaskPo
	if err := r.Db(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *MysqlRepositoryImpl) Delete(ctx context.Context, id uint64) error {
	return r.Db(ctx).Model(&TaskPo{}).Where("id = ?", id).Update("status", domain.TaskStatusDeleted).Error
}

func (r *MysqlRepositoryImpl) Update(ctx context.Context, id uint64, patch *domain.TaskPatch) error {
	values := patchToMap(patch)
	if len(values) == 0 {
		return nil
	}
	return r.Db(ctx).Model(&TaskPo{}).Where("id = ?", id).Updates(values).Error
}

func (r *MysqlRepositoryImpl) List(ctx context.Context, filter *domain.TaskFilter) ([]*domain.Task, error) {
	var pos []TaskPo
	query := r.Db(ctx).Model(&TaskPo{})
	if filter.Status.IsPresent() {
		query = query.Where("status = ?", filter.Status.MustGet())
	}
	if err := query.Find(&pos).Error; err != nil {
		return nil, err
	}
	return lo.Map(pos, func(po TaskPo, _ int) *domain.Task {
		return po.ToDomain()
	}), nil
}

func (r *MysqlRepositoryImpl) FindActiveTasks(ctx context.Context) ([]*domain.Task, error) {
	var pos []TaskPo
	if err := r.Db(ctx).Where("status = ?", domain.TaskStatusActive).Find(&pos).Error; err != nil {
		return nil, err
	}
	return lo.Map(pos, func(po TaskPo, _ int) *domain.Task {
		return po.ToDomain()
	}), nil
}

func (r *MysqlRepositoryImpl) FindByIDWithAssignments(ctx context.Context, id uint64) (*domain.Task, error) {
	var po TaskPo
	if err := r.Db(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	assignments, err := r.listWithAssignment(ctx, id)
	if err != nil {
		return nil, err
	}
	task := po.ToDomain()
	task.Assignments = assignments
	return task, nil
}

func (r *MysqlRepositoryImpl) ListWithAssignments(ctx context.Context, filter *domain.TaskFilter) ([]*domain.Task, error) {
	pos, err := r.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	var ret []*domain.Task
	for _, task := range pos {
		assignments, err := r.listWithAssignment(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		task.Assignments = assignments
		ret = append(ret, task)
	}
	return ret, nil
}

func (r *MysqlRepositoryImpl) listWithAssignment(ctx context.Context, id uint64) ([]*domain.TaskAssignment, error) {
	var assignments []TaskAssignmentPo
	if err := r.Db(ctx).Model(&TaskAssignmentPo{}).Where("task_id = ?", id).Limit(100).Find(&assignments).Error; err != nil {
		return nil, err
	}
	return lo.Map(assignments, func(po TaskAssignmentPo, _ int) *domain.TaskAssignment {
		return po.ToDomain()
	}), nil
}

func (r *MysqlRepositoryImpl) CreateAssignment(ctx context.Context, assignment *domain.TaskAssignment) error {
	po := new(TaskAssignmentPo).FromDomain(assignment)
	err := r.Db(ctx).Create(po).Error
	if err != nil {
		return err
	}
	assignment.ID = po.ID
	return nil
}

func (r *MysqlRepositoryImpl) DeleteAssignment(ctx context.Context, id uint64) error {
	return r.Db(ctx).Delete(&TaskAssignmentPo{}, id).Error
}

func (r *MysqlRepositoryImpl) DeleteAssignmentsByExecutorName(ctx context.Context, executorName string) error {
	return r.Db(ctx).Delete(&TaskAssignmentPo{}, "executor_name = ?", executorName).Error
}

func (r *MysqlRepositoryImpl) UpdateAssignment(ctx context.Context, id uint64, patch *domain.TaskAssignmentPatch) error {
	values := assignmentPatchToMap(patch)
	if len(values) == 0 {
		return nil
	}
	return r.Db(ctx).Model(&TaskAssignmentPo{}).Where("id = ?", id).Updates(values).Error
}

func (r *MysqlRepositoryImpl) GetAssignmentByTaskIDAndExecutorName(ctx context.Context, taskID uint64, executorName string) (*domain.TaskAssignment, error) {
	var po TaskAssignmentPo
	if err := r.Db(ctx).Where("task_id = ? AND executor_name = ?", taskID, executorName).First(&po).Error; err != nil {
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *MysqlRepositoryImpl) ListAssignmentsWithExecutor(ctx context.Context, executorName string) ([]*domain.TaskAssignment, error) {
	var assignments []TaskAssignmentPo
	if err := r.Db(ctx).Model(&TaskAssignmentPo{}).Where("executor_name = ?", executorName).Find(&assignments).Error; err != nil {
		return nil, err
	}
	return lo.Map(assignments, func(po TaskAssignmentPo, _ int) *domain.TaskAssignment {
		return po.ToDomain()
	}), nil
}
