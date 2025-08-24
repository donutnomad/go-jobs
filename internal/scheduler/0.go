package scheduler

import "github.com/google/wire"

var Provider = wire.NewSet(
	New,
	NewTaskRunner,
	NewEventBus,
	NewHealthChecker,
)

type ITaskRunner interface {
	RemoveBreaker(executorID uint64)
	ResetBreaker(executorID uint64)
	CancelTimeout(executionID uint64)
	Submit(taskId uint64, parameters map[string]any, executionId uint64)
}

type IEmitter interface {
	SubmitNewTask(taskID uint64, parameters map[string]any) error
	ReloadTasks() error
	CancelExecutionTimer(executionID uint64) error
}
