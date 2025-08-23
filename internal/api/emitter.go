package api

import (
	"github.com/jobs/scheduler/internal/scheduler"
)

type IEmitter interface {
	SubmitNewTask(taskID uint64, parameters map[string]any, executionID uint64) error
	ReloadTasks() error
	CancelExecutionTimer(executionID uint64) error
}

type EventBus struct {
	scheduler  *scheduler.Scheduler
	taskRunner *scheduler.TaskRunner
}

func NewEventBus(scheduler *scheduler.Scheduler, taskRunner *scheduler.TaskRunner) *EventBus {
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
