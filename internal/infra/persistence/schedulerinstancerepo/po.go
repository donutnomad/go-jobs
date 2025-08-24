package schedulerinstancerepo

import (
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
	return "jobs_scheduler_instances"
}
