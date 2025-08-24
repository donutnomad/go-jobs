package schedulerinstancerepo

import (
	domain "github.com/jobs/scheduler/internal/biz/scheduler_instance"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

func (po *SchedulerInstancePO) ToDomain() *domain.SchedulerInstance {
	return &domain.SchedulerInstance{
		ID:         po.ID,
		InstanceID: po.InstanceID,
		Host:       po.Host,
		Port:       po.Port,
		IsLeader:   po.IsLeader,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func (po *SchedulerInstancePO) FromDomain(d *domain.SchedulerInstance) *SchedulerInstancePO {
	return &SchedulerInstancePO{
		Mode: commonrepo.Mode{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		InstanceID: d.InstanceID,
		Host:       d.Host,
		Port:       d.Port,
		IsLeader:   d.IsLeader,
	}
}
