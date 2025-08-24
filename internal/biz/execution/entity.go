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

// Domain behavior helpers to encapsulate state transitions.

// StartNow marks execution as running and sets StartTime.
func (e *TaskExecution) StartNow() *TaskExecution {
    now := time.Now()
    e.Status = ExecutionStatusRunning
    e.StartTime = &now
    return e
}

// AssignExecutor sets the executor and retry attempt.
func (e *TaskExecution) AssignExecutor(executorID uint64, attempt int) *TaskExecution {
    e.ExecutorID = executorID
    e.RetryCount = attempt
    return e
}

// MarkFailed marks execution failed and captures end time and reason.
func (e *TaskExecution) MarkFailed(reason string) *TaskExecution {
    now := time.Now()
    e.Status = ExecutionStatusFailed
    e.EndTime = &now
    e.Logs = reason
    return e
}

// MarkTimeout marks execution timed out and sets end time and reason.
func (e *TaskExecution) MarkTimeout() *TaskExecution {
    now := time.Now()
    e.Status = ExecutionStatusTimeout
    e.EndTime = &now
    e.Logs = "Execution timeout"
    return e
}
