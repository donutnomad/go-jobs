package loadbalance

import (
	"context"
	"fmt"
	"sync"

	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/load_balance"
)

// StickyStrategy 粘性策略 - 始终选择同一个执行器
type StickyStrategy struct {
	loadBalanceRepo load_balance.Repo
	mu              sync.Mutex
}

func NewStickyStrategy(loadBalanceRepo load_balance.Repo) *StickyStrategy {
	return &StickyStrategy{
		loadBalanceRepo: loadBalanceRepo,
	}
}

func (s *StickyStrategy) Select(ctx context.Context, taskID uint64, executors []*executor.Executor) (*executor.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取或创建负载均衡状态
	state, err := s.loadBalanceRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get load balance state: %w", err)
	}
	
	if state == nil {
		// 创建新状态，选择第一个执行器作为粘性执行器
		selected := executors[0]
		state = &load_balance.LoadBalanceState{
			TaskID:           taskID,
			StickyExecutorID: &selected.ID,
			LastExecutorID:   &selected.ID,
		}
		if err := s.loadBalanceRepo.Create(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
		return selected, nil
	}

	// 检查粘性执行器是否仍然可用
	if state.StickyExecutorID != nil {
		// 查找当前可用执行器中是否有匹配的
		for _, exec := range executors {
			if exec.ID == *state.StickyExecutorID {
				// 找到粘性执行器
				state.LastExecutorID = &exec.ID
				if err := s.loadBalanceRepo.Save(ctx, state); err != nil {
					return nil, fmt.Errorf("failed to update load balance state: %w", err)
				}
				return exec, nil
			}
		}
	}

	// 粘性执行器不可用，选择新的粘性执行器
	selected := executors[0]
	state.StickyExecutorID = &selected.ID
	state.LastExecutorID = &selected.ID
	if err := s.loadBalanceRepo.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to update load balance state: %w", err)
	}

	return selected, nil
}

func (s *StickyStrategy) Name() string {
	return "sticky"
}
