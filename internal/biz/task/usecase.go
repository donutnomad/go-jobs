package task

import (
	"context"
	"errors"

	"github.com/samber/mo"
)

type Usecase struct {
	repo Repo
}

func NewUsecase(repo Repo) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Create(ctx context.Context, task *Task) error {
	return u.repo.Create(ctx, task)
}

type UpdateRequest struct {
	Name                mo.Option[string]
	CronExpression      mo.Option[string]
	Parameters          mo.Option[map[string]any]
	ExecutionMode       mo.Option[ExecutionMode]
	LoadBalanceStrategy mo.Option[LoadBalanceStrategy]
	MaxRetry            mo.Option[int]
	TimeoutSeconds      mo.Option[int]
	Status              mo.Option[TaskStatus]
}

func (u *Usecase) Update(ctx context.Context, id uint64, req *UpdateRequest) error {
	task, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	} else if task == nil {
		return errors.New("task not found")
	}

	patch := NewTaskPatch()
	patch.Name = req.Name.ToPointer()
	patch.CronExpression = req.CronExpression.ToPointer()
	patch.Parameters = req.Parameters.ToPointer()
	patch.ExecutionMode = req.ExecutionMode.ToPointer()
	patch.LoadBalanceStrategy = req.LoadBalanceStrategy.ToPointer()
	patch.MaxRetry = req.MaxRetry.ToPointer()
	patch.TimeoutSeconds = req.TimeoutSeconds.ToPointer()
	patch.Status = req.Status.ToPointer()

	return u.repo.Update(ctx, task.ID, patch)
}

func (u *Usecase) Delete(ctx context.Context, id uint64) error {
	task, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	} else if task == nil {
		return errors.New("task not found")
	}

	return u.repo.Delete(ctx, id)
}

func (u *Usecase) Pause(ctx context.Context, id uint64) error {
	task, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	} else if task == nil {
		return errors.New("task not found")
	}
	patch, err := task.Pause()
	if err != nil {
		return err
	}
	return u.repo.Update(ctx, id, patch)
}

func (u *Usecase) Resume(ctx context.Context, id uint64) error {
	task, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	} else if task == nil {
		return errors.New("task not found")
	}
	patch, err := task.Resume()
	if err != nil {
		return err
	}
	return u.repo.Update(ctx, id, patch)
}

func (u *Usecase) AssignExecutor(ctx context.Context, id uint64, executorName string, priority int, weight int) (*TaskAssignment, error) {
	task, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	} else if task == nil {
		return nil, errors.New("task not found")
	}
	newAssignment := &TaskAssignment{
		TaskID:       id,
		ExecutorName: executorName,
		Priority:     priority,
		Weight:       weight,
	}
	return newAssignment, u.repo.CreateAssignment(ctx, newAssignment)
}

func (u *Usecase) UpdateAssignment(ctx context.Context, taskID uint64, executorName string, priority mo.Option[int], weight mo.Option[int]) (*TaskAssignment, error) {
	assignment, err := u.repo.GetAssignmentByTaskIDAndExecutorName(ctx, taskID, executorName)
	if err != nil {
		return nil, err
	} else if assignment == nil {
		return nil, errors.New("assignment not found")
	}

	patch := NewTaskAssignmentPatch()
	patch.Priority = priority.ToPointer()
	patch.Weight = weight.ToPointer()
	if patch.Priority != nil {
		assignment.Priority = *patch.Priority
	}
	if patch.Weight != nil {
		assignment.Weight = *patch.Weight
	}

	return assignment, u.repo.UpdateAssignment(ctx, assignment.ID, patch)
}

func (u *Usecase) UnassignExecutor(ctx context.Context, taskID uint64, executorName string) error {
	assignment, err := u.repo.GetAssignmentByTaskIDAndExecutorName(ctx, taskID, executorName)
	if err != nil {
		return err
	} else if assignment == nil {
		return errors.New("assignment not found")
	}
	return u.repo.DeleteAssignment(ctx, assignment.ID)
}
