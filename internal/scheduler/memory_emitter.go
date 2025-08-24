package scheduler

type EventBus struct {
	scheduler *Scheduler
}

func NewEventBus(scheduler *Scheduler) *EventBus {
	return &EventBus{scheduler: scheduler}
}

func (e *EventBus) SubmitNewTask(taskID uint64, parameters map[string]any) error {
	return e.scheduler.ScheduleNow(taskID, parameters)
}

func (e *EventBus) ReloadTasks() error {
	return e.scheduler.ReloadTasks()
}

func (e *EventBus) CancelExecutionTimer(executionID uint64) error {
	e.scheduler.CancelExecutionTimeout(executionID)
	return nil
}
