package entity

import (
	"errors"
	"time"
)

// Task 任务领域实体
type Task struct {
	ID                  string
	Name                string
	CronExpression      string
	Parameters          map[string]any
	ExecutionMode       ExecutionMode
	LoadBalanceStrategy LoadBalanceStrategy
	MaxRetry            int
	TimeoutSeconds      int
	Status              TaskStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	TaskExecutors       []TaskExecutor
}

// TaskExecutor 任务执行器关联
type TaskExecutor struct {
	ID           string
	TaskID       string
	ExecutorName string
	Priority     int
	Weight       int
	Task         *Task // 关联的任务信息（可选）
}

// ExecutionMode 执行模式
type ExecutionMode string

const (
	ExecutionModeSequential ExecutionMode = "sequential"
	ExecutionModeParallel   ExecutionMode = "parallel"
	ExecutionModeSkip       ExecutionMode = "skip"
)

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin         LoadBalanceStrategy = "round_robin"
	LoadBalanceWeightedRoundRobin LoadBalanceStrategy = "weighted_round_robin"
	LoadBalanceRandom             LoadBalanceStrategy = "random"
	LoadBalanceSticky             LoadBalanceStrategy = "sticky"
	LoadBalanceLeastLoaded        LoadBalanceStrategy = "least_loaded"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusActive  TaskStatus = "active"
	TaskStatusPaused  TaskStatus = "paused"
	TaskStatusDeleted TaskStatus = "deleted"
)

// NewTask 创建新任务
func NewTask(name, cronExpression string) (*Task, error) {
	if name == "" {
		return nil, errors.New("任务名称不能为空")
	}
	if cronExpression == "" {
		return nil, errors.New("Cron表达式不能为空")
	}

	return &Task{
		Name:                name,
		CronExpression:      cronExpression,
		Parameters:          make(map[string]any),
		ExecutionMode:       ExecutionModeParallel,
		LoadBalanceStrategy: LoadBalanceRoundRobin,
		MaxRetry:            3,
		TimeoutSeconds:      300,
		Status:              TaskStatusActive,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}, nil
}

// Pause 暂停任务
func (t *Task) Pause() error {
	if t.Status == TaskStatusPaused {
		return errors.New("任务已经暂停")
	}
	if t.Status == TaskStatusDeleted {
		return errors.New("无法暂停已删除的任务")
	}
	t.Status = TaskStatusPaused
	t.UpdatedAt = time.Now()
	return nil
}

// Resume 恢复任务
func (t *Task) Resume() error {
	if t.Status == TaskStatusActive {
		return errors.New("任务已经激活")
	}
	if t.Status == TaskStatusDeleted {
		return errors.New("无法恢复已删除的任务")
	}
	t.Status = TaskStatusActive
	t.UpdatedAt = time.Now()
	return nil
}

// Delete 删除任务（软删除）
func (t *Task) Delete() {
	t.Status = TaskStatusDeleted
	t.UpdatedAt = time.Now()
}

// Update 更新任务
func (t *Task) Update(name, cronExpression string, parameters map[string]any, executionMode ExecutionMode,
	loadBalanceStrategy LoadBalanceStrategy, maxRetry, timeoutSeconds int) {
	if name != "" {
		t.Name = name
	}
	if cronExpression != "" {
		t.CronExpression = cronExpression
	}
	if parameters != nil {
		t.Parameters = parameters
	}
	if executionMode != "" {
		t.ExecutionMode = executionMode
	}
	if loadBalanceStrategy != "" {
		t.LoadBalanceStrategy = loadBalanceStrategy
	}
	if maxRetry > 0 {
		t.MaxRetry = maxRetry
	}
	if timeoutSeconds > 0 {
		t.TimeoutSeconds = timeoutSeconds
	}
	t.UpdatedAt = time.Now()
}

// IsActive 判断任务是否激活
func (t *Task) IsActive() bool {
	return t.Status == TaskStatusActive
}

// CanBeExecuted 判断任务是否可以执行
func (t *Task) CanBeExecuted() bool {
	return t.Status == TaskStatusActive
}
