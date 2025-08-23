package loadbalancerepo

import (
	domain "github.com/jobs/scheduler/internal/biz/load_balance"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

type LoadBalanceStatePO struct {
	commonrepo.Mode
	TaskID          uint64  `gorm:"column:task_id;uniqueIndex"`
	RoundRobinIndex int     `gorm:"column:round_robin_index;default:0"`
	LastExecutorID  *uint64 `gorm:"column:last_executor_id"`
}

func (LoadBalanceStatePO) TableName() string {
	return "jobs_load_balance_state"
}

func (po *LoadBalanceStatePO) ToDomain() *domain.LoadBalanceState {
	return &domain.LoadBalanceState{
		ID:              po.ID,
		TaskID:          po.TaskID,
		RoundRobinIndex: po.RoundRobinIndex,
		LastExecutorID:  po.LastExecutorID,
		UpdatedAt:       po.UpdatedAt,
	}
}

func (po *LoadBalanceStatePO) FromDomain(d *domain.LoadBalanceState) *LoadBalanceStatePO {
	po.ID = d.ID
	po.TaskID = d.TaskID
	po.RoundRobinIndex = d.RoundRobinIndex
	po.LastExecutorID = d.LastExecutorID
	if !d.UpdatedAt.IsZero() {
		po.UpdatedAt = d.UpdatedAt
	}
	return po
}
