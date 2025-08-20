package loadbalance

import (
	"context"
	"fmt"
	"sync"

	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/storage"
	"gorm.io/gorm"
)

// RoundRobinStrategy 轮询策略
type RoundRobinStrategy struct {
	storage *storage.Storage
	mu      sync.Mutex
}

func NewRoundRobinStrategy(storage *storage.Storage) *RoundRobinStrategy {
	return &RoundRobinStrategy{
		storage: storage,
	}
}

func (s *RoundRobinStrategy) Select(ctx context.Context, taskID string, executors []*models.Executor) (*models.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取或创建负载均衡状态
	var state models.LoadBalanceState
	err := s.storage.DB().Where("task_id = ?", taskID).First(&state).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新状态
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

func (s *RoundRobinStrategy) Name() string {
	return "round_robin"
}
