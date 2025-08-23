package schedulerinstancerepo

import (
	domain "github.com/jobs/scheduler/internal/biz/scheduler_instance"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

type SchedulerInstancePO struct {
	commonrepo.Mode
	InstanceID string `gorm:"column:instance_id;size:64;uniqueIndex"`
	Host       string `gorm:"column:host;size:255"`
	Port       int    `gorm:"column:port"`
	IsLeader   bool   `gorm:"column:is_leader;default:false"`
}

func (SchedulerInstancePO) TableName() string {
	return "scheduler_instances"
}

func (po *SchedulerInstancePO) ToDomain() *domain.SchedulerInstance {
	return &domain.SchedulerInstance{
		ID:         po.InstanceID,
		InstanceID: po.InstanceID,
		Host:       po.Host,
		Port:       po.Port,
		IsLeader:   po.IsLeader,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func (po *SchedulerInstancePO) FromDomain(d *domain.SchedulerInstance) *SchedulerInstancePO {
	po.InstanceID = d.InstanceID
	po.Host = d.Host
	po.Port = d.Port
	po.IsLeader = d.IsLeader
	if !d.CreatedAt.IsZero() {
		po.CreatedAt = d.CreatedAt
	}
	if !d.UpdatedAt.IsZero() {
		po.UpdatedAt = d.UpdatedAt
	}
	return po
}