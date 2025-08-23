package scheduler_instance

import (
	"time"
)

type SchedulerInstance struct {
	ID         string
	InstanceID string
	Host       string
	Port       int
	IsLeader   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}