package executorrepo

import (
	"time"

	domain "github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"gorm.io/datatypes"
)

type Executor struct {
	commonrepo.Mode
	Name                string                `gorm:"column:name;size:255;not null;index:idx_name_instance"`
	InstanceID          string                `gorm:"column:instance_id;size:255;not null;uniqueIndex;index:idx_name_instance"`
	BaseURL             string                `gorm:"column:base_url;size:500;not null"`
	HealthCheckURL      string                `gorm:"column:health_check_url;size:500"`
	Status              domain.ExecutorStatus `gorm:"column:status;size:50;not null;index:idx_status_healthy"`
	IsHealthy           bool                  `gorm:"column:is_healthy;default:true;index:idx_status_healthy"`
	LastHealthCheck     *time.Time            `gorm:"column:last_health_check"`
	HealthCheckFailures int                   `gorm:"column:health_check_failures;default:0"`
	Metadata            datatypes.JSONMap     `gorm:"column:metadata;type:json"`
}

func (Executor) TableName() string {
	return "jobs_executors"
}
