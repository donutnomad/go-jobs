package models

import (
	"time"
)

type ExecutorStatus string

const (
	ExecutorStatusOnline      ExecutorStatus = "online"
	ExecutorStatusOffline     ExecutorStatus = "offline"
	ExecutorStatusMaintenance ExecutorStatus = "maintenance"
)

func (s ExecutorStatus) ToInt() int {
	switch s {
	case ExecutorStatusOnline:
		return 1
	case ExecutorStatusOffline:
		return 3
	case ExecutorStatusMaintenance:
		return 2
	}
	return 0
}

type Executor struct {
	ID                  string         `gorm:"primaryKey;size:64" json:"id"`
	Name                string         `gorm:"size:255;not null;index:idx_name_instance" json:"name"`
	InstanceID          string         `gorm:"size:255;not null;uniqueIndex;index:idx_name_instance" json:"instance_id"`
	BaseURL             string         `gorm:"size:500;not null" json:"base_url"`
	HealthCheckURL      string         `gorm:"size:500" json:"health_check_url"`
	Status              ExecutorStatus `gorm:"type:enum('online','offline','maintenance');default:'online';index:idx_status_healthy" json:"status"`
	IsHealthy           bool           `gorm:"default:true;index:idx_status_healthy" json:"is_healthy"`
	LastHealthCheck     *time.Time     `gorm:"" json:"last_health_check"`
	HealthCheckFailures int            `gorm:"default:0" json:"health_check_failures"`
	Metadata            JSONMap        `gorm:"type:json" json:"metadata"`
	CreatedAt           time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	TaskExecutors []TaskExecutor `gorm:"foreignKey:ExecutorID" json:"task_executors,omitempty"`
}

func (Executor) TableName() string {
	return "executors"
}

type TaskExecutor struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	TaskID     string    `gorm:"size:64;not null;uniqueIndex:uk_task_executor;index" json:"task_id"`
	ExecutorID string    `gorm:"size:64;not null;uniqueIndex:uk_task_executor" json:"executor_id"`
	Priority   int       `gorm:"default:0" json:"priority"`
	Weight     int       `gorm:"default:1" json:"weight"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

	Task     *Task     `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"task,omitempty"`
	Executor *Executor `gorm:"foreignKey:ExecutorID;constraint:OnDelete:CASCADE" json:"executor,omitempty"`
}

func (TaskExecutor) TableName() string {
	return "task_executors"
}
