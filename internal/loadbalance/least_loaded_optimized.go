package loadbalance

import (
	"context"
	"fmt"
	"sync"

	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
)

// OptimizedLeastLoadedStrategy 优化的最少负载策略
type OptimizedLeastLoadedStrategy struct {
	storage *orm.Storage
	cache   *LoadCache
}

// LoadCache 负载缓存
type LoadCache struct {
	mu    sync.RWMutex
	loads map[string]int64
}

func NewOptimizedLeastLoadedStrategy(storage *orm.Storage) *OptimizedLeastLoadedStrategy {
	return &OptimizedLeastLoadedStrategy{
		storage: storage,
		cache: &LoadCache{
			loads: make(map[string]int64),
		},
	}
}

func (s *OptimizedLeastLoadedStrategy) Select(ctx context.Context, taskID string, executors []*models.Executor) (*models.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	// 批量查询所有执行器的负载
	executorIDs := make([]string, 0, len(executors))
	executorMap := make(map[string]*models.Executor)
	for _, exec := range executors {
		executorIDs = append(executorIDs, exec.ID)
		executorMap[exec.ID] = exec
	}

	// 使用单个查询获取所有执行器的负载
	type ExecutorLoad struct {
		ExecutorID string
		Count      int64
	}

	var loads []ExecutorLoad
	err := s.storage.DB().
		Model(&models.TaskExecution{}).
		Select("executor_id, COUNT(*) as count").
		Where("executor_id IN ? AND status = ?", executorIDs, models.ExecutionStatusRunning).
		Group("executor_id").
		Scan(&loads).Error

	if err != nil {
		return nil, fmt.Errorf("failed to count running tasks: %w", err)
	}

	// 构建负载映射
	loadMap := make(map[string]int64)
	for _, load := range loads {
		loadMap[load.ExecutorID] = load.Count
	}

	// 为没有运行任务的执行器设置负载为0
	for _, execID := range executorIDs {
		if _, exists := loadMap[execID]; !exists {
			loadMap[execID] = 0
		}
	}

	// 选择负载最小的执行器
	var minExecutor *models.Executor
	minLoad := int64(^uint64(0) >> 1) // MaxInt64

	for execID, load := range loadMap {
		if load < minLoad {
			minLoad = load
			minExecutor = executorMap[execID]
		}
	}

	if minExecutor == nil {
		return nil, fmt.Errorf("failed to select executor")
	}

	// 异步更新负载均衡状态，避免阻塞
	go s.updateLoadBalanceState(taskID, minExecutor.ID)

	// 更新缓存
	s.cache.mu.Lock()
	s.cache.loads[minExecutor.ID] = minLoad + 1
	s.cache.mu.Unlock()

	return minExecutor, nil
}

func (s *OptimizedLeastLoadedStrategy) updateLoadBalanceState(taskID, executorID string) {
	var state models.LoadBalanceState
	err := s.storage.DB().Where("task_id = ?", taskID).First(&state).Error
	if err != nil {
		// 创建新状态
		state = models.LoadBalanceState{
			TaskID:         taskID,
			LastExecutorID: &executorID,
		}
		s.storage.DB().Create(&state)
	} else {
		state.LastExecutorID = &executorID
		s.storage.DB().Save(&state)
	}
}

func (s *OptimizedLeastLoadedStrategy) Name() string {
	return "least_loaded_optimized"
}
