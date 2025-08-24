package scheduler

type EventBus struct {
	scheduler  *Scheduler
	taskRunner ITaskRunner
}

func NewEventBus(scheduler *Scheduler, taskRunner ITaskRunner) *EventBus {
	return &EventBus{scheduler: scheduler, taskRunner: taskRunner}
}

func (e *EventBus) SubmitNewTask(taskID uint64, parameters map[string]any, executionID uint64) error {
	e.taskRunner.Submit(taskID, parameters, executionID)
	return nil
}

func (e *EventBus) ReloadTasks() error {
	return e.scheduler.ReloadTasks()
}

func (e *EventBus) CancelExecutionTimer(executionID uint64) error {
	e.taskRunner.CancelTimeout(executionID)
	return nil
}
