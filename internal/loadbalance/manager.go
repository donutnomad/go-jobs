package loadbalance

import (
	"context"
	"fmt"

	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/storage"
)

// Manager 负载均衡管理器
type Manager struct {
	storage    *storage.Storage
	strategies map[models.LoadBalanceStrategy]Strategy
}

// NewManager 创建负载均衡管理器
func NewManager(storage *storage.Storage) *Manager {
	m := &Manager{
		storage:    storage,
		strategies: make(map[models.LoadBalanceStrategy]Strategy),
	}

	// 注册所有策略
	m.strategies[models.LoadBalanceRoundRobin] = NewRoundRobinStrategy(storage)
	m.strategies[models.LoadBalanceWeightedRoundRobin] = NewWeightedRoundRobinStrategy(storage)
	m.strategies[models.LoadBalanceRandom] = NewRandomStrategy()
	m.strategies[models.LoadBalanceSticky] = NewStickyStrategy(storage)
	m.strategies[models.LoadBalanceLeastLoaded] = NewLeastLoadedStrategy(storage)

	return m
}

// SelectExecutor 根据任务的负载均衡策略选择执行器
func (m *Manager) SelectExecutor(ctx context.Context, task *models.Task, executors []*models.Executor) (*models.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors for task %s", task.Name)
	}

	// 使用任务的负载均衡策略
	strategy, ok := m.strategies[task.LoadBalanceStrategy]
	if !ok {
		// 默认使用轮询策略
		strategy = m.strategies[models.LoadBalanceRoundRobin]
	}

	// 使用策略选择执行器
	executor, err := strategy.Select(ctx, task.ID, executors)
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
