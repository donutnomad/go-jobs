package task

import (
	"errors"
	"time"
)

type Task struct {
	ID                  uint64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Name                string
	CronExpression      string
	Parameters          map[string]any
	ExecutionMode       ExecutionMode
	LoadBalanceStrategy LoadBalanceStrategy
	Status              TaskStatus
	MaxRetry            int
	TimeoutSeconds      int

	Assignments []*TaskAssignment // 默认不加载
}

func (t *Task) Pause() (*TaskPatch, error) {
	if t.Status == TaskStatusPaused {
		return nil, errors.New("task is already paused")
	} else if t.Status == TaskStatusDeleted {
		return nil, errors.New("cannot pause deleted task")
	}
	t.Status = TaskStatusPaused
	return new(TaskPatch).WithStatus(t.Status), nil
}

func (t *Task) Resume() (*TaskPatch, error) {
	if t.Status == TaskStatusActive {
		return nil, errors.New("task is already active")
	} else if t.Status == TaskStatusDeleted {
		return nil, errors.New("cannot resume deleted task")
	}
	t.Status = TaskStatusActive
	return new(TaskPatch).WithStatus(t.Status), nil
}

// MergeParameters merges provided parameters into task's parameters map.
// Initializes the map if nil.
func (t *Task) MergeParameters(extra map[string]any) {
    if extra == nil {
        return
    }
    if t.Parameters == nil {
        t.Parameters = make(map[string]any)
    }
    for k, v := range extra {
        t.Parameters[k] = v
    }
}

type TaskAssignment struct {
	ID           uint64
	CreatedAt    time.Time
	TaskID       uint64
	ExecutorName string
	Priority     int
	Weight       int
}

type TaskPatch struct {
	Name                *string
	CronExpression      *string
	Parameters          *map[string]any
	ExecutionMode       *ExecutionMode
	LoadBalanceStrategy *LoadBalanceStrategy
	Status              *TaskStatus
	MaxRetry            *int
	TimeoutSeconds      *int
}

func NewTaskPatch() *TaskPatch {
	return &TaskPatch{}
}

func (p *TaskPatch) WithName(name string) *TaskPatch {
	p.Name = &name
	return p
}

func (p *TaskPatch) WithCronExpression(cronExpression string) *TaskPatch {
	p.CronExpression = &cronExpression
	return p
}

func (p *TaskPatch) WithParameters(parameters map[string]any) *TaskPatch {
	p.Parameters = &parameters
	return p
}

func (p *TaskPatch) WithExecutionMode(executionMode ExecutionMode) *TaskPatch {
	p.ExecutionMode = &executionMode
	return p
}

func (p *TaskPatch) WithLoadBalanceStrategy(loadBalanceStrategy LoadBalanceStrategy) *TaskPatch {
	p.LoadBalanceStrategy = &loadBalanceStrategy
	return p
}

func (p *TaskPatch) WithStatus(status TaskStatus) *TaskPatch {
	p.Status = &status
	return p
}

func (p *TaskPatch) WithMaxRetry(maxRetry int) *TaskPatch {
	p.MaxRetry = &maxRetry
	return p
}

func (p *TaskPatch) WithTimeoutSeconds(timeoutSeconds int) *TaskPatch {
	p.TimeoutSeconds = &timeoutSeconds
	return p
}

type TaskAssignmentPatch struct {
	Priority *int
	Weight   *int
}

func NewTaskAssignmentPatch() *TaskAssignmentPatch {
	return &TaskAssignmentPatch{}
}

func (p *TaskAssignmentPatch) WithPriority(priority int) *TaskAssignmentPatch {
	p.Priority = &priority
	return p
}

func (p *TaskAssignmentPatch) WithWeight(weight int) *TaskAssignmentPatch {
	p.Weight = &weight
	return p
}
