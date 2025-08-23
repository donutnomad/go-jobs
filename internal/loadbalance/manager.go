package loadbalance

import (
	"context"
	"fmt"

	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
)

// Manager 负载均衡管理器
type Manager struct {
	storage    *orm.Storage
	strategies map[task.LoadBalanceStrategy]Strategy
}

// NewManager 创建负载均衡管理器
func NewManager(storage *orm.Storage) *Manager {
	m := &Manager{
		storage:    storage,
		strategies: make(map[task.LoadBalanceStrategy]Strategy),
	}

	// 注册所有策略
	m.strategies[task.LoadBalanceRoundRobin] = NewRoundRobinStrategy(storage)
	m.strategies[task.LoadBalanceWeightedRoundRobin] = NewWeightedRoundRobinStrategy(storage)
	m.strategies[task.LoadBalanceRandom] = NewRandomStrategy()
	m.strategies[task.LoadBalanceSticky] = NewStickyStrategy(storage)
	m.strategies[task.LoadBalanceLeastLoaded] = NewLeastLoadedStrategy(storage)

	return m
}

// SelectExecutor 根据任务的负载均衡策略选择执行器
func (m *Manager) SelectExecutor(ctx context.Context, task_ *task.Task, executors []*executor.Executor) (*models.Executor, error) {
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
	executor, err := strategy.Select(ctx, task_.ID, executors)
	if err != nil {
		return nil, fmt.Errorf("failed to select executor using %s strategy: %w", strategy.Name(), err)
	}

	return executor, nil
}

// GetStrategy 获取指定的负载均衡策略
func (m *Manager) GetStrategy(strategyType models.LoadBalanceStrategy) (Strategy, error) {
	strategy, ok := m.strategies[strategyType]
	if !ok {
		return nil, fmt.Errorf("unknown load balance strategy: %s", strategyType)
	}
	return strategy, nil
}
