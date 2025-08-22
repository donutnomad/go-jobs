package models

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

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

func (e *Executor) GetStopURL() string {
	return fmt.Sprintf("%s/stop", e.BaseURL)
}

func (e *Executor) SetStatus(status ExecutorStatus) {
	e.Status = status
	if status == ExecutorStatusOnline {
		e.SetToOnline()
	}
}

func (e *Executor) SetToOnline() {
	e.Status = ExecutorStatusOnline
	e.IsHealthy = true
	e.HealthCheckFailures = 0
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

type ExecutorRepo struct {
	db *gorm.DB
}

func NewExecutorRepo(db *gorm.DB) *ExecutorRepo {
	return &ExecutorRepo{db: db}
}

func (r *ExecutorRepo) GetHealthyExecutors(ctx context.Context, taskID string) ([]*Executor, error) {
	var executors []*Executor
	query := r.db.WithContext(ctx).
		Joins("JOIN task_executors ON task_executors.executor_id = executors.id").
		Where("task_executors.task_id = ?", taskID).
		Where("executors.status = ?", ExecutorStatusOnline).
		Where("executors.is_healthy = ?", true).
		Order("task_executors.priority DESC")
	if err := query.Find(&executors).Error; err != nil {
		return nil, fmt.Errorf("failed to get healthy executors: %w", err)
	}
	return executors, nil
}
