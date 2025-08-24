package executor

import (
	"context"

	"github.com/google/wire"
)

var Provider = wire.NewSet(NewUsecase)

type Usecase struct {
	executorRepo Repo
}

func NewUsecase(executorRepo Repo) *Usecase {
	return &Usecase{executorRepo: executorRepo}
}

func (u *Usecase) Update(ctx context.Context, id uint64, patch *ExecutorPatch) (*Executor, error) {
	exec, err := u.executorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	err = u.executorRepo.Update(ctx, id, patch)
	if err != nil {
		return nil, err
	}

	return exec, nil
}

func (u *Usecase) UpdateStatus(ctx context.Context, id uint64, status ExecutorStatus) (*Executor, error) {
	exec, err := u.executorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	patch := exec.SetStatus(status)
	err = u.executorRepo.Update(ctx, id, patch)
	if err != nil {
		return nil, err
	}
	return exec, nil
}
