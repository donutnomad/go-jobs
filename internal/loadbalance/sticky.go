package loadbalance

import (
	"context"
	"fmt"
	"sync"

	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
	"gorm.io/gorm"
)

// StickyStrategy 粘性策略 - 始终选择同一个执行器
type StickyStrategy struct {
	storage *orm.Storage
	mu      sync.Mutex
}

func NewStickyStrategy(storage *orm.Storage) *StickyStrategy {
	return &StickyStrategy{
		storage: storage,
	}
}

func (s *StickyStrategy) Select(ctx context.Context, taskID string, executors []*models.Executor) (*models.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取或创建负载均衡状态
	var state models.LoadBalanceState
	err := s.storage.DB().Where("task_id = ?", taskID).First(&state).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新状态，选择第一个执行器作为粘性执行器
		selected := executors[0]
		state = models.LoadBalanceState{
			TaskID:           taskID,
			StickyExecutorID: &selected.ID,
			LastExecutorID:   &selected.ID,
		}
		if err := s.storage.DB().Create(&state).Error; err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
		return selected, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get load balance state: %w", err)
	}

	// 检查粘性执行器是否仍然可用（基于名称匹配，允许同名执行器替代）
	if state.StickyExecutorID != nil {
		// 获取原粘性执行器的名称
		var stickyExec models.Executor
		if err := s.storage.DB().Where("id = ?", *state.StickyExecutorID).First(&stickyExec).Error; err == nil {
			// 查找当前可用执行器中是否有同名的
			for _, exec := range executors {
				if exec.Name == stickyExec.Name {
					// 找到同名执行器，使用它并更新状态
					state.LastExecutorID = &exec.ID
					if err := s.storage.DB().Save(&state).Error; err != nil {
						return nil, fmt.Errorf("failed to update load balance state: %w", err)
					}
					return exec, nil
				}
			}
		}
	}

	// 粘性执行器不可用，选择新的粘性执行器
	selected := executors[0]
	state.StickyExecutorID = &selected.ID
	state.LastExecutorID = &selected.ID
	if err := s.storage.DB().Save(&state).Error; err != nil {
		return nil, fmt.Errorf("failed to update load balance state: %w", err)
	}

	return selected, nil
}

func (s *StickyStrategy) Name() string {
	return "sticky"
}
