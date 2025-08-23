package loadbalance

import (
	"context"
	"fmt"

	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/load_balance"
	"github.com/jobs/scheduler/internal/biz/task"
)

// Manager 负载均衡管理器
type Manager struct {
	strategies         map[task.LoadBalanceStrategy]Strategy
	loadBalanceRepo    load_balance.Repo
	taskRepo          task.Repo
	executionRepo     execution.Repo
}

// NewManager 创建负载均衡管理器
func NewManager(loadBalanceRepo load_balance.Repo, taskRepo task.Repo, executionRepo execution.Repo) *Manager {
	m := &Manager{
		strategies:      make(map[task.LoadBalanceStrategy]Strategy),
		loadBalanceRepo: loadBalanceRepo,
		taskRepo:        taskRepo,
		executionRepo:   executionRepo,
	}

	// 注册所有策略
	m.strategies[task.LoadBalanceRoundRobin] = NewRoundRobinStrategy(loadBalanceRepo)
	m.strategies[task.LoadBalanceWeightedRoundRobin] = NewWeightedRoundRobinStrategy(taskRepo, loadBalanceRepo)
	m.strategies[task.LoadBalanceRandom] = NewRandomStrategy()
	m.strategies[task.LoadBalanceSticky] = NewStickyStrategy(loadBalanceRepo)
	m.strategies[task.LoadBalanceLeastLoaded] = NewLeastLoadedStrategy(executionRepo, loadBalanceRepo)

	return m
}

// SelectExecutor 根据任务的负载均衡策略选择执行器
func (m *Manager) SelectExecutor(ctx context.Context, task_ *task.Task, executors []*executor.Executor) (*executor.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors for task %s", task_.Name)
	}

	// 使用任务的负载均衡策略
	strategy, ok := m.strategies[task_.LoadBalanceStrategy]
	if !ok {
		// 默认使用轮询策略
		strategy = m.strategies[task.LoadBalanceRoundRobin]
	}

	// 使用策略选择执行器
	exec, err := strategy.Select(ctx, task_.ID, executors)
	if err != nil {
		return nil, fmt.Errorf("failed to select executor using %s strategy: %w", strategy.Name(), err)
	}

	return exec, nil
}

// GetStrategy 获取指定的负载均衡策略
func (m *Manager) GetStrategy(strategyType task.LoadBalanceStrategy) (Strategy, error) {
	strategy, ok := m.strategies[strategyType]
	if !ok {
		return nil, fmt.Errorf("unknown load balance strategy: %s", strategyType)
	}
	return strategy, nil
}
