package loadbalance

import (
	"context"
	"fmt"
	"sync"

	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/storage"
	"gorm.io/gorm"
)

// WeightedRoundRobinStrategy 加权轮询策略
type WeightedRoundRobinStrategy struct {
	storage *storage.Storage
	mu      sync.Mutex
}

func NewWeightedRoundRobinStrategy(storage *storage.Storage) *WeightedRoundRobinStrategy {
	return &WeightedRoundRobinStrategy{
		storage: storage,
	}
}

func (s *WeightedRoundRobinStrategy) Select(ctx context.Context, taskID string, executors []*models.Executor) (*models.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取执行器权重信息
	var taskExecutors []models.TaskExecutor
	err := s.storage.DB().
		Where("task_id = ?", taskID).
		Find(&taskExecutors).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get task executors: %w", err)
	}

	// 构建权重映射
	weightMap := make(map[string]int)
	totalWeight := 0
	for _, te := range taskExecutors {
		weight := te.Weight
		if weight <= 0 {
			weight = 1
		}
		weightMap[te.ExecutorID] = weight

		// 只计算可用执行器的权重
		for _, exec := range executors {
			if exec.ID == te.ExecutorID {
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
	var state models.LoadBalanceState
	err = s.storage.DB().Where("task_id = ?", taskID).First(&state).Error
	if err == gorm.ErrRecordNotFound {
		state = models.LoadBalanceState{
			TaskID:          taskID,
			RoundRobinIndex: 0,
		}
		if err := s.storage.DB().Create(&state).Error; err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get load balance state: %w", err)
	}

	// 基于权重选择执行器
	targetWeight := state.RoundRobinIndex % totalWeight
	currentWeight := 0

	for _, exec := range executors {
		weight := weightMap[exec.ID]
		if weight <= 0 {
			weight = 1
		}
		currentWeight += weight
		if currentWeight > targetWeight {
			// 更新状态
			state.RoundRobinIndex = (state.RoundRobinIndex + 1) % totalWeight
			state.LastExecutorID = &exec.ID
			if err := s.storage.DB().Save(&state).Error; err != nil {
				return nil, fmt.Errorf("failed to update load balance state: %w", err)
			}
			return exec, nil
		}
	}

	// 默认返回第一个
	return executors[0], nil
}

func (s *WeightedRoundRobinStrategy) selectDefault(ctx context.Context, taskID string, executors []*models.Executor) (*models.Executor, error) {
	// 获取或创建负载均衡状态
	var state models.LoadBalanceState
	err := s.storage.DB().Where("task_id = ?", taskID).First(&state).Error
	if err == gorm.ErrRecordNotFound {
		state = models.LoadBalanceState{
			TaskID:          taskID,
			RoundRobinIndex: 0,
		}
		if err := s.storage.DB().Create(&state).Error; err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get load balance state: %w", err)
	}

	// 选择下一个执行器
	index := state.RoundRobinIndex % len(executors)
	selected := executors[index]

	// 更新索引
	state.RoundRobinIndex = (state.RoundRobinIndex + 1) % len(executors)
	state.LastExecutorID = &selected.ID
	if err := s.storage.DB().Save(&state).Error; err != nil {
		return nil, fmt.Errorf("failed to update load balance state: %w", err)
	}

	return selected, nil
}

func (s *WeightedRoundRobinStrategy) Name() string {
	return "weighted_round_robin"
}
