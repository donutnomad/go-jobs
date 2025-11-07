package schedulerinstancerepo

import (
	"context"
	"errors"

	"github.com/google/wire"
	domain "github.com/jobs/scheduler/internal/biz/scheduler_instance"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

var Provider = wire.NewSet(NewMysqlRepositoryImpl)

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
		Updates(map[string]interface{}{
			"is_leader":  isLeader,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *MysqlRepositoryImpl) UpdateHeartbeat(ctx context.Context, instanceID string) error {
	return r.Db(ctx).Model(&SchedulerInstancePO{}).
		Where("instance_id = ?", instanceID).
		Update("updated_at", gorm.Expr("NOW()")).Error
}

func (r *MysqlRepositoryImpl) ListActive(ctx context.Context, maxIdleSeconds int) ([]*domain.SchedulerInstance, error) {
	var pos []*SchedulerInstancePO
	// 查询最近更新时间在maxIdleSeconds秒内的实例
	// 排序：1. Leader在前 2. 按创建时间升序
	if err := r.Db(ctx).
		Where("updated_at >= DATE_SUB(NOW(), INTERVAL ? SECOND)", maxIdleSeconds).
		Order("is_leader DESC, created_at ASC").
		Find(&pos).Error; err != nil {
		return nil, err
	}
	return lo.Map(pos, func(po *SchedulerInstancePO, _ int) *domain.SchedulerInstance {
		return po.ToDomain()
	}), nil
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
