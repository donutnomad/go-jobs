package loadbalance

import (
	"context"
	"fmt"
	"sync"

	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/load_balance"
	"github.com/jobs/scheduler/internal/biz/task"
)

// WeightedRoundRobinStrategy 加权轮询策略
type WeightedRoundRobinStrategy struct {
	taskRepo        task.Repo
	loadBalanceRepo load_balance.Repo
	mu              sync.Mutex
}

func NewWeightedRoundRobinStrategy(taskRepo task.Repo, loadBalanceRepo load_balance.Repo) *WeightedRoundRobinStrategy {
	return &WeightedRoundRobinStrategy{
		taskRepo:        taskRepo,
		loadBalanceRepo: loadBalanceRepo,
	}
}

func (s *WeightedRoundRobinStrategy) Select(ctx context.Context, taskID uint64, executors []*executor.Executor) (*executor.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取任务关联的执行器信息
	tsk, err := s.taskRepo.FindByIDWithAssignments(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// 构建权重映射 - 使用执行器名称作为键
	weightMap := make(map[string]int)
	totalWeight := 0
	for _, te := range tsk.Assignments {
		weight := te.Weight
		if weight <= 0 {
			weight = 1
		}
		weightMap[te.ExecutorName] = weight

		// 只计算可用执行器的权重
		for _, exec := range executors {
			if exec.Name == te.ExecutorName {
				totalWeight += weight
				break
			}
		}
	}

	if totalWeight == 0 {
		// 如果没有权重信息，使用普通轮询
		return s.selectDefault(ctx, taskID, executors)
	}

	// 获取或创建负载均衡状态
	state, err := s.loadBalanceRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		state = load_balance.NewLoadBalanceStateForTask(taskID)
		if err := s.loadBalanceRepo.Create(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
	}

	// 基于权重选择执行器
	targetWeight := state.CurrentIndex(totalWeight)
	currentWeight := 0

	for _, exec := range executors {
		weight := weightMap[exec.Name]
		if weight <= 0 {
			weight = 1
		}
		currentWeight += weight
		if currentWeight > targetWeight {
			// 更新状态
			state.AdvanceRoundRobin(totalWeight)
			state.SetLastExecutorID(exec.ID)
			if err := s.loadBalanceRepo.Save(ctx, state); err != nil {
				return nil, fmt.Errorf("failed to update load balance state: %w", err)
			}
			return exec, nil
		}
	}

	// 默认返回第一个
	return executors[0], nil
}

func (s *WeightedRoundRobinStrategy) selectDefault(ctx context.Context, taskID uint64, executors []*executor.Executor) (*executor.Executor, error) {
	// 获取或创建负载均衡状态
	state, err := s.loadBalanceRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		state = load_balance.NewLoadBalanceStateForTask(taskID)
		if err := s.loadBalanceRepo.Create(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
	}

	// 选择下一个执行器
	selected := executors[state.CurrentIndex(len(executors))]

	// 更新索引
	state.AdvanceRoundRobin(len(executors))
	state.SetLastExecutorID(selected.ID)
	if err := s.loadBalanceRepo.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to update load balance state: %w", err)
	}

	return selected, nil
}

func (s *WeightedRoundRobinStrategy) Name() string {
	return "weighted_round_robin"
}
