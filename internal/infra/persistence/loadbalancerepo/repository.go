package loadbalancerepo

import (
	"context"
	"errors"

	domain "github.com/jobs/scheduler/internal/biz/load_balance"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
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

func (r *MysqlRepositoryImpl) GetByTaskID(ctx context.Context, taskID uint64) (*domain.LoadBalanceState, error) {
	var po LoadBalanceStatePO
	err := r.Db(ctx).Where("task_id = ?", taskID).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *MysqlRepositoryImpl) Save(ctx context.Context, state *domain.LoadBalanceState) error {
	po := new(LoadBalanceStatePO).FromDomain(state)
	return r.Db(ctx).Save(po).Error
}

func (r *MysqlRepositoryImpl) Create(ctx context.Context, state *domain.LoadBalanceState) error {
	po := new(LoadBalanceStatePO).FromDomain(state)
	err := r.Db(ctx).Create(po).Error
	if err != nil {
		return err
	}
	state.UpdatedAt = po.UpdatedAt
	return nil
}