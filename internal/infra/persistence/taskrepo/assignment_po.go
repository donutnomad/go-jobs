package taskrepo

import "github.com/jobs/scheduler/internal/infra/persistence/commonrepo"

type TaskAssignmentPo struct {
	commonrepo.Mode
	TaskID       uint64 `gorm:"not null;index;index:idx_task_id_executor_name"`
	ExecutorName string `gorm:"not null;index;index:idx_task_id_executor_name"`
	Priority     int    `gorm:"not null;default:0"`
	Weight       int    `gorm:"not null;default:1"`
}

func (TaskAssignmentPo) TableName() string {
	return "jobs_task_assignments"
}
