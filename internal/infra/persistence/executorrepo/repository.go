package executorrepo

import (
	"context"
	"errors"

	domain "github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type MysqlRepositoryImpl struct {
	commonrepo.DefaultRepo
}

func NewMysqlRepositoryImpl(db commonrepo.DB) domain.Repo {
	return &MysqlRepositoryImpl{
		DefaultRepo: commonrepo.NewDefaultRepo(db),
	}
}

func (m *MysqlRepositoryImpl) Save(ctx context.Context, executor *domain.Executor) error {
	po := new(Executor).FromDomain(executor)
	return m.Db(ctx).Save(po).Error
}

func (m *MysqlRepositoryImpl) Create(ctx context.Context, executor *domain.Executor) error {
	po := new(Executor).FromDomain(executor)
	return m.Db(ctx).Create(po).Error
}

func (m *MysqlRepositoryImpl) Delete(ctx context.Context, id uint64) error {
	return m.Db(ctx).Delete(&Executor{}, id).Error
}

func (m *MysqlRepositoryImpl) GetByID(ctx context.Context, id uint64) (*domain.Executor, error) {
	var po Executor
	if err := m.Db(ctx).First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (m *MysqlRepositoryImpl) GetByInstanceID(ctx context.Context, instanceID uint64) (*domain.Executor, error) {
	var po Executor
	if err := m.Db(ctx).Where("id = ?", instanceID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (m *MysqlRepositoryImpl) GetByName(ctx context.Context, name string) (*domain.Executor, error) {
	var po Executor
	if err := m.Db(ctx).Where("name = ?", name).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (m *MysqlRepositoryImpl) Update(ctx context.Context, id uint64, patch *domain.ExecutorPatch) error {
	values := patchToMap(patch)
	if len(values) == 0 {
		return nil
	}
	return m.Db(ctx).Model(&Executor{}).Where("id = ?", id).Updates(values).Error
}

func (m *MysqlRepositoryImpl) List(ctx context.Context, offset, limit int) ([]*domain.Executor, error) {
	var pos []*Executor
	if err := m.Db(ctx).Offset(offset).Limit(limit).Find(&pos).Error; err != nil {
		return nil, err
	}
	return lo.Map(pos, func(po *Executor, _ int) *domain.Executor {
		return po.ToDomain()
	}), nil
}

func (m *MysqlRepositoryImpl) FindByStatus(ctx context.Context, status []domain.ExecutorStatus) ([]*domain.Executor, error) {
	var pos []*Executor
	if err := m.Db(ctx).Where("status IN (?)", status).Find(&pos).Error; err != nil {
		return nil, err
	}
	return lo.Map(pos, func(po *Executor, _ int) *domain.Executor {
		return po.ToDomain()
	}), nil
}

func (m *MysqlRepositoryImpl) GetHealthyExecutorsForTask(ctx context.Context, taskID uint64) ([]*domain.Executor, error) {
	var pos []*Executor
	if err := m.Db(ctx).Table(Executor{}.TableName()+" as e").
		Joins("INNER JOIN jobs_task_assignment as ta ON ta.executor_name = e.name").
		Where("ta.task_id = ?", taskID).
		Where("e.is_healthy = ?", true).
		Where("e.status = ?", domain.ExecutorStatusOnline).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	return lo.Map(pos, func(po *Executor, _ int) *domain.Executor {
		return po.ToDomain()
	}), nil
}
