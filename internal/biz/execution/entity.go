package execution

import "time"

type TaskExecution struct {
	ID        uint64
	CreatedAt time.Time
	UpdatedAt time.Time

	TaskID        uint64
	ExecutorID    uint64
	ScheduledTime time.Time
	StartTime     *time.Time
	EndTime       *time.Time
	Status        ExecutionStatus
	Result        map[string]any
	Logs          string
	RetryCount    int
}

type TaskExecutionPatch struct {
	StartTime  *time.Time
	EndTime    *time.Time
	Status     *ExecutionStatus
	Result     *map[string]any
	Logs       *string
	RetryCount *int
}
