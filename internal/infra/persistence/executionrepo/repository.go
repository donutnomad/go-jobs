package executionrepo

import (
	"context"

	"github.com/jobs/scheduler/internal/biz/execution"
	domain "github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

type MysqlRepositoryImpl struct {
	commonrepo.DefaultRepo
}

func NewMysqlRepositoryImpl(db commonrepo.DB) domain.Repo {
	return &MysqlRepositoryImpl{
		DefaultRepo: commonrepo.NewDefaultRepo(db),
	}
}

// Count implements execution.Repo.
func (r *MysqlRepositoryImpl) Count(ctx context.Context, query domain.CountQuery) (int64, error) {
	var db = r.Db(ctx).Model(&TaskExecution{})

	if query.StartTime.IsPresent() {
		db = db.Where("scheduled_time >= ?", query.StartTime.MustGet())
	}
	if query.EndTime.IsPresent() {
		db = db.Where("scheduled_time <= ?", query.EndTime.MustGet())
	}
	if query.TaskID.IsPresent() {
		db = db.Where("task_id = ?", query.TaskID.MustGet())
	}
	if query.Status.IsPresent() {
		db = db.Where("status = ?", query.Status.MustGet())
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *MysqlRepositoryImpl) GetByID(ctx context.Context, id uint64) (*domain.TaskExecution, error) {
	var po = new(TaskExecution)
	if err := r.Db(ctx).First(po, id).Error; err != nil {
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *MysqlRepositoryImpl) Save(ctx context.Context, execution *domain.TaskExecution) error {
	po := new(TaskExecution).FromDomain(execution)
	err := r.Db(ctx).Save(po).Error
	if err != nil {
		return err
	}
	execution.ID = po.ID
	execution.CreatedAt = po.CreatedAt
	execution.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *MysqlRepositoryImpl) Create(ctx context.Context, execution *execution.TaskExecution) error {
	po := new(TaskExecution).FromDomain(execution)
	err := r.Db(ctx).Create(po).Error
	if err != nil {
		return err
	}
	execution.ID = po.ID
	execution.CreatedAt = po.CreatedAt
	execution.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *MysqlRepositoryImpl) Delete(ctx context.Context, id uint64) error {
	return r.Db(ctx).Delete(&TaskExecution{}, id).Error
}

func (r *MysqlRepositoryImpl) List(ctx context.Context, filter domain.ListFilter, offset, limit int) ([]*domain.TaskExecution, int64, error) {
	db := r.Db(ctx).Model(&TaskExecution{})

	if filter.StartTime.IsPresent() {
		db = db.Where("scheduled_time >= ?", filter.StartTime.MustGet())
	}
	if filter.EndTime.IsPresent() {
		db = db.Where("scheduled_time <= ?", filter.EndTime.MustGet())
	}
	if filter.TaskID.IsPresent() {
		db = db.Where("task_id = ?", filter.TaskID.MustGet())
	}
	if filter.Status.IsPresent() {
		db = db.Where("status = ?", filter.Status.MustGet())
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	var pos []*TaskExecution
	if err := db.Order("scheduled_time DESC").Limit(limit).Offset(offset).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	domains := make([]*domain.TaskExecution, len(pos))
	for i := range pos {
		domains[i] = pos[i].ToDomain()
	}
	return domains, count, nil
}
