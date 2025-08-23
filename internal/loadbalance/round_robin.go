package loadbalance

import (
	"context"
	"fmt"
	"sync"

	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/load_balance"
	"github.com/yitter/idgenerator-go/idgen"
)

// RoundRobinStrategy 轮询策略
type RoundRobinStrategy struct {
	loadBalanceRepo load_balance.Repo
	mu              sync.Mutex
}

func NewRoundRobinStrategy(loadBalanceRepo load_balance.Repo) *RoundRobinStrategy {
	return &RoundRobinStrategy{
		loadBalanceRepo: loadBalanceRepo,
	}
}

func (s *RoundRobinStrategy) Select(ctx context.Context, taskID uint64, executors []*executor.Executor) (*executor.Executor, error) {
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
		// 创建新状态
		state = &load_balance.LoadBalanceState{
			ID:              uint64(idgen.NextId()),
			TaskID:          taskID,
			RoundRobinIndex: 0,
		}
		if err := s.loadBalanceRepo.Create(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
	}

	// 选择下一个执行器
	index := state.RoundRobinIndex % len(executors)
	selected := executors[index]

	// 更新索引
	state.RoundRobinIndex = (state.RoundRobinIndex + 1) % len(executors)
	state.LastExecutorID = &selected.ID
	if err := s.loadBalanceRepo.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to update load balance state: %w", err)
	}

	return selected, nil
}

func (s *RoundRobinStrategy) Name() string {
	return "round_robin"
}
