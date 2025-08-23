package schedulerinstancerepo

import (
	"context"
	"errors"

	domain "github.com/jobs/scheduler/internal/biz/scheduler_instance"
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

func (r *MysqlRepositoryImpl) GetByInstanceID(ctx context.Context, instanceID string) (*domain.SchedulerInstance, error) {
	var po SchedulerInstancePO
	err := r.Db(ctx).Where("instance_id = ?", instanceID).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *MysqlRepositoryImpl) Create(ctx context.Context, instance *domain.SchedulerInstance) error {
	po := new(SchedulerInstancePO).FromDomain(instance)
	return r.Db(ctx).Create(po).Error
}

func (r *MysqlRepositoryImpl) Save(ctx context.Context, instance *domain.SchedulerInstance) error {
	//po := new(SchedulerInstancePO).FromDomain(instance)
	return r.Db(ctx).Model(&SchedulerInstancePO{}).Where("instance_id = ?", instance.InstanceID).Update("is_leader", instance.IsLeader).Error
}

func (r *MysqlRepositoryImpl) UpdateLeaderStatus(ctx context.Context, instanceID string, isLeader bool) error {
	return r.Db(ctx).Model(&SchedulerInstancePO{}).
		Where("instance_id = ?", instanceID).
		Update("is_leader", isLeader).Error
}

func (r *MysqlRepositoryImpl) DeleteExpired(ctx context.Context, maxAge int64) error {
	return r.Db(ctx).
		Where("updated_at < FROM_UNIXTIME(?)", maxAge).
		Delete(&SchedulerInstancePO{}).Error
}

func (r *MysqlRepositoryImpl) List(ctx context.Context) ([]*domain.SchedulerInstance, error) {
	var pos []*SchedulerInstancePO
	if err := r.Db(ctx).Find(&pos).Error; err != nil {
		return nil, err
	}
	return lo.Map(pos, func(po *SchedulerInstancePO, _ int) *domain.SchedulerInstance {
		return po.ToDomain()
	}), nil
}
