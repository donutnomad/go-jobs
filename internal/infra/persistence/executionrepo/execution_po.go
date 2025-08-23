package executionrepo

import (
	"time"

	domain "github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"gorm.io/datatypes"
)

type TaskExecution struct {
	commonrepo.Mode
	TaskID        uint64                 `gorm:"column:task_id;not null;index:idx_task_status"`
	ExecutorID    uint64                 `gorm:"column:executor_id;not null;index"`
	ScheduledTime time.Time              `gorm:"column:scheduled_time;not null;index"`
	StartTime     *time.Time             `gorm:"column:start_time"`
	EndTime       *time.Time             `gorm:"column:end_time"`
	Status        domain.ExecutionStatus `gorm:"column:status;size:50;not null;index:idx_task_status;index"`
	Result        datatypes.JSONMap      `gorm:"column:result;type:json"`
	Logs          string                 `gorm:"column:logs;type:text"`
	RetryCount    int                    `gorm:"column:retry_count;default:0"`
}

func (TaskExecution) TableName() string {
	return "task_executions"
}
