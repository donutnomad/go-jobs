package task

import (
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/app/types"
	"github.com/robfig/cron/v3"
)

// TaskStatus 任务状态（领域特有）
type TaskStatus int

const (
	TaskStatusActive TaskStatus = iota + 1
	TaskStatusPaused
	TaskStatusDeleted
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusActive:
		return "active"
	case TaskStatusPaused:
		return "paused"
	case TaskStatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// IsActive 判断是否为活跃状态
func (s TaskStatus) IsActive() bool {
	return s == TaskStatusActive
}

// CanBeScheduled 判断是否可以被调度
func (s TaskStatus) CanBeScheduled() bool {
	return s == TaskStatusActive
}

// CronExpression cron表达式（领域特有值对象）
type CronExpression string

// NewCronExpression 创建cron表达式
func NewCronExpression(expr string) (CronExpression, error) {
	if expr == "" {
		return "", fmt.Errorf("cron expression cannot be empty")
	}

	// 验证cron表达式格式
	_, err := cron.ParseStandard(expr)
	if err != nil {
		return "", fmt.Errorf("invalid cron expression: %w", err)
	}

	return CronExpression(expr), nil
}

// String 返回字符串表示
func (c CronExpression) String() string {
	return string(c)
}

// IsEmpty 判断是否为空
func (c CronExpression) IsEmpty() bool {
	return string(c) == ""
}

// NextTime 计算下次执行时间
func (c CronExpression) NextTime(from time.Time) (time.Time, error) {
	schedule, err := cron.ParseStandard(string(c))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}
	return schedule.Next(from), nil
}

// ExecutionMode 执行模式（领域特有）
type ExecutionMode string

const (
	ExecutionModeSequential ExecutionMode = "sequential" // 串行执行
	ExecutionModeParallel   ExecutionMode = "parallel"   // 并行执行
	ExecutionModeSkip       ExecutionMode = "skip"       // 跳过执行
)

func (m ExecutionMode) String() string {
	return string(m)
}

// IsValid 验证执行模式是否有效
func (m ExecutionMode) IsValid() bool {
	switch m {
	case ExecutionModeSequential, ExecutionModeParallel, ExecutionModeSkip:
		return true
	default:
		return false
	}
}

// AllowsConcurrent 是否允许并发执行
func (m ExecutionMode) AllowsConcurrent() bool {
	return m == ExecutionModeParallel
}

// ShouldSkipOnRunning 运行时是否应该跳过
func (m ExecutionMode) ShouldSkipOnRunning() bool {
	return m == ExecutionModeSkip
}

// LoadBalanceStrategy 负载均衡策略（领域特有）
type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin         LoadBalanceStrategy = "round_robin"          // 轮询
	LoadBalanceWeightedRoundRobin LoadBalanceStrategy = "weighted_round_robin" // 加权轮询
	LoadBalanceRandom             LoadBalanceStrategy = "random"               // 随机
	LoadBalanceSticky             LoadBalanceStrategy = "sticky"               // 粘性
	LoadBalanceLeastLoaded        LoadBalanceStrategy = "least_loaded"         // 最少负载
)

func (s LoadBalanceStrategy) String() string {
	return string(s)
}

// IsValid 验证策略是否有效
func (s LoadBalanceStrategy) IsValid() bool {
	switch s {
	case LoadBalanceRoundRobin, LoadBalanceWeightedRoundRobin,
		LoadBalanceRandom, LoadBalanceSticky, LoadBalanceLeastLoaded:
		return true
	default:
		return false
	}
}

// UsesWeight 是否使用权重
func (s LoadBalanceStrategy) UsesWeight() bool {
	return s == LoadBalanceWeightedRoundRobin
}

// IsStateful 是否有状态
func (s LoadBalanceStrategy) IsStateful() bool {
	switch s {
	case LoadBalanceRoundRobin, LoadBalanceWeightedRoundRobin, LoadBalanceSticky:
		return true
	default:
		return false
	}
}

// TaskConfiguration 任务配置值对象
type TaskConfiguration struct {
	MaxRetry       int `json:"max_retry"`
	TimeoutSeconds int `json:"timeout_seconds"`
}

// NewTaskConfiguration 创建任务配置
func NewTaskConfiguration(maxRetry, timeoutSeconds int) TaskConfiguration {
	if maxRetry < 0 {
		maxRetry = 3
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}
	return TaskConfiguration{
		MaxRetry:       maxRetry,
		TimeoutSeconds: timeoutSeconds,
	}
}

// IsValid 验证配置是否有效
func (c TaskConfiguration) IsValid() bool {
	return c.MaxRetry >= 0 && c.TimeoutSeconds > 0
}

// HasTimeout 是否设置了超时
func (c TaskConfiguration) HasTimeout() bool {
	return c.TimeoutSeconds > 0
}

// GetTimeoutDuration 获取超时时长
func (c TaskConfiguration) GetTimeoutDuration() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name                string              `json:"name" binding:"required"`
	CronExpression      string              `json:"cron_expression" binding:"required"`
	Parameters          types.JSONMap       `json:"parameters"`
	ExecutionMode       ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                 `json:"max_retry"`
	TimeoutSeconds      int                 `json:"timeout_seconds"`
}

// Validate 验证请求参数
func (r CreateTaskRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("task name is required")
	}

	if _, err := NewCronExpression(r.CronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	if r.ExecutionMode != "" && !r.ExecutionMode.IsValid() {
		return fmt.Errorf("invalid execution mode: %s", r.ExecutionMode)
	}

	if r.LoadBalanceStrategy != "" && !r.LoadBalanceStrategy.IsValid() {
		return fmt.Errorf("invalid load balance strategy: %s", r.LoadBalanceStrategy)
	}

	return nil
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name                string              `json:"name"`
	CronExpression      string              `json:"cron_expression"`
	Parameters          types.JSONMap       `json:"parameters"`
	ExecutionMode       ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                 `json:"max_retry"`
	TimeoutSeconds      int                 `json:"timeout_seconds"`
	Status              TaskStatus          `json:"status"`
}

// Validate 验证更新请求参数
func (r UpdateTaskRequest) Validate() error {
	if r.CronExpression != "" {
		if _, err := NewCronExpression(r.CronExpression); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	if r.ExecutionMode != "" && !r.ExecutionMode.IsValid() {
		return fmt.Errorf("invalid execution mode: %s", r.ExecutionMode)
	}

	if r.LoadBalanceStrategy != "" && !r.LoadBalanceStrategy.IsValid() {
		return fmt.Errorf("invalid load balance strategy: %s", r.LoadBalanceStrategy)
	}

	return nil
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	types.Filter
	Name   *string  `json:"name"`
	Status []string `json:"status"`
	Mode   *string  `json:"mode"`
}

// NewTaskFilter 创建任务过滤器
func NewTaskFilter() TaskFilter {
	return TaskFilter{
		Filter: types.NewFilter(),
	}
}

// ScheduleTaskRequest 调度任务请求
type ScheduleTaskRequest struct {
	TaskID     types.ID      `json:"task_id" binding:"required"`
	Parameters types.JSONMap `json:"parameters"`
	ScheduleAt *time.Time    `json:"schedule_at"`
}

// TriggerTaskRequest 触发任务请求
type TriggerTaskRequest struct {
	Parameters types.JSONMap `json:"parameters"`
}
