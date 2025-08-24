package loadbalance

import (
	"context"
	"fmt"

	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/load_balance"
	"github.com/jobs/scheduler/internal/utils/loExt"
)

// LeastLoadedStrategy 最少负载策略
type LeastLoadedStrategy struct {
	executionRepo   execution.Repo
	loadBalanceRepo load_balance.Repo
}

func NewLeastLoadedStrategy(executionRepo execution.Repo, loadBalanceRepo load_balance.Repo) *LeastLoadedStrategy {
	return &LeastLoadedStrategy{
		executionRepo:   executionRepo,
		loadBalanceRepo: loadBalanceRepo,
	}
}

func (s *LeastLoadedStrategy) Select(ctx context.Context, taskID uint64, executors []*executor.Executor) (*executor.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	// 统计每个执行器的运行任务数
	type executorLoad struct {
		executor *executor.Executor
		load     int64
	}

	loads := make([]executorLoad, 0, len(executors))
	for _, exec := range executors {
		// 统计该执行器的运行任务数
		count, err := s.executionRepo.CountByExecutorAndStatus(ctx, exec.ID, loExt.DefSlice(execution.ExecutionStatusRunning))
		if err != nil {
			return nil, fmt.Errorf("failed to count running tasks: %w", err)
		}

		loads = append(loads, executorLoad{
			executor: exec,
			load:     count,
		})
	}

	// 选择负载最小的执行器
	minLoad := loads[0]
	for _, load := range loads[1:] {
		if load.load < minLoad.load {
			minLoad = load
		}
	}

	// 更新负载均衡状态
	state, err := s.loadBalanceRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		// 创建新状态
		state = load_balance.NewLoadBalanceStateForTask(taskID)
		state.SetLastExecutorID(minLoad.executor.ID)
		if err := s.loadBalanceRepo.Create(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
	} else {
		state.SetLastExecutorID(minLoad.executor.ID)
		if err := s.loadBalanceRepo.Save(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to update load balance state: %w", err)
		}
	}

	return minLoad.executor, nil
}

func (s *LeastLoadedStrategy) Name() string {
	return "least_loaded"
}
