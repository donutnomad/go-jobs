package api

import "github.com/jobs/scheduler/internal/scheduler"

type IEmitter interface {
	SubmitNewTask(taskID string, executionID string) error
	ReloadTasks() error
	CancelExecutionTimer(executionID string) error
}

type EventBus struct {
	scheduler  *scheduler.Scheduler
	taskRunner *scheduler.TaskRunner
}

func NewEventBus(scheduler *scheduler.Scheduler, taskRunner *scheduler.TaskRunner) *EventBus {
	return &EventBus{scheduler: scheduler, taskRunner: taskRunner}
}

func (e *EventBus) SubmitNewTask(taskID string, executionID string) error {
	e.taskRunner.Submit2(taskID, executionID)
	return nil
}

func (e *EventBus) ReloadTasks() error {
	return e.scheduler.ReloadTasks()
}

func (e *EventBus) CancelExecutionTimer(executionID string) error {
	e.taskRunner.CancelTimeout(executionID)
	return nil
}
