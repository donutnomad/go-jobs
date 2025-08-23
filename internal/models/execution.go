package models

import (
	"time"
)

type LoadBalanceState struct {
	TaskID           uint64    `gorm:"primaryKey;size:64" json:"task_id"`
	LastExecutorID   *uint64   `gorm:"size:64" json:"last_executor_id"`
	RoundRobinIndex  int       `gorm:"default:0" json:"round_robin_index"`
	StickyExecutorID *uint64   `gorm:"size:64" json:"sticky_executor_id"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (LoadBalanceState) TableName() string {
	return "load_balance_state"
}

type SchedulerInstance struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	InstanceID string    `gorm:"size:255;not null;uniqueIndex" json:"instance_id"`
	Host       string    `gorm:"size:255;not null" json:"host"`
	Port       int       `gorm:"not null" json:"port"`
	IsLeader   bool      `gorm:"default:false;index" json:"is_leader"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SchedulerInstance) TableName() string {
	return "scheduler_instances"
}
