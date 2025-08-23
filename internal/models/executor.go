package models

import (
	"context"
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/infra/persistence/taskrepo"
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

	// 在应用层手动填充的关联字段（不使用GORM关联）
	TaskExecutors []taskrepo.TaskAssignmentPo `gorm:"-" json:"task_executors,omitempty"`
}

type ExecutorRepo struct {
	db *gorm.DB
}

func NewExecutorRepo(db *gorm.DB) *ExecutorRepo {
	return &ExecutorRepo{db: db}
}

func (r *ExecutorRepo) GetHealthyExecutors(ctx context.Context, taskID uint64) ([]*Executor, error) {
	var executors []*Executor
	query := r.db.WithContext(ctx).
		Joins("JOIN task_executors ON task_executors.executor_name = executors.name").
		Where("task_executors.task_id = ?", taskID).
		Where("executors.status = ?", ExecutorStatusOnline).
		Where("executors.is_healthy = ?", true).
		Order("task_executors.priority DESC")
	if err := query.Find(&executors).Error; err != nil {
		return nil, fmt.Errorf("failed to get healthy executors: %w", err)
	}
	return executors, nil
}
