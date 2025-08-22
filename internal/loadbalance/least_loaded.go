package loadbalance

import (
	"context"
	"fmt"

	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
)

// LeastLoadedStrategy 最少负载策略
type LeastLoadedStrategy struct {
	storage *orm.Storage
}

func NewLeastLoadedStrategy(storage *orm.Storage) *LeastLoadedStrategy {
	return &LeastLoadedStrategy{
		storage: storage,
	}
}

func (s *LeastLoadedStrategy) Select(ctx context.Context, taskID string, executors []*models.Executor) (*models.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	// 统计每个执行器的运行任务数
	type executorLoad struct {
		executor *models.Executor
		load     int64
	}

	loads := make([]executorLoad, 0, len(executors))
	for _, exec := range executors {
		var count int64
		// 统计所有同名执行器的运行任务总数
		err := s.storage.DB().
			Model(&models.TaskExecution{}).
			Joins("JOIN executors ON executors.id = task_executions.executor_id").
			Where("executors.name = ? AND task_executions.status = ?", exec.Name, models.ExecutionStatusRunning).
			Count(&count).Error
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
	var state models.LoadBalanceState
	err := s.storage.DB().Where("task_id = ?", taskID).First(&state).Error
	if err != nil {
		// 创建新状态
		state = models.LoadBalanceState{
			TaskID:         taskID,
			LastExecutorID: &minLoad.executor.ID,
		}
		if err := s.storage.DB().Create(&state).Error; err != nil {
			return nil, fmt.Errorf("failed to create load balance state: %w", err)
		}
	} else {
		state.LastExecutorID = &minLoad.executor.ID
		if err := s.storage.DB().Save(&state).Error; err != nil {
			return nil, fmt.Errorf("failed to update load balance state: %w", err)
		}
	}

	return minLoad.executor, nil
}

func (s *LeastLoadedStrategy) Name() string {
	return "least_loaded"
}
