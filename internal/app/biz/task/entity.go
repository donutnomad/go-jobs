package task

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/app/types"
)

// Task 任务领域实体
// 遵循DDD原则，包含业务逻辑和状态管理
type Task struct {
	// 基础属性
	id                  types.ID
	name                string
	cronExpression      CronExpression
	parameters          types.JSONMap
	executionMode       ExecutionMode
	loadBalanceStrategy LoadBalanceStrategy
	configuration       TaskConfiguration
	status              TaskStatus

	// 时间戳
	createdAt time.Time
	updatedAt time.Time

	// 领域事件（可选，用于事件驱动）
	domainEvents []interface{}
}

// NewTask 创建新任务（工厂方法）
func NewTask(name string, cronExpr string) (*Task, error) {
	// 验证参数
	if name == "" {
		return nil, fmt.Errorf("task name cannot be empty")
	}

	// 创建和验证cron表达式
	cronExpression, err := NewCronExpression(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// 生成唯一ID
	id := types.ID(uuid.New().String())

	now := time.Now()
	task := &Task{
		id:                  id,
		name:                name,
		cronExpression:      cronExpression,
		parameters:          make(types.JSONMap),
		executionMode:       ExecutionModeParallel,        // 默认并行执行
		loadBalanceStrategy: LoadBalanceRoundRobin,        // 默认轮询策略
		configuration:       NewTaskConfiguration(3, 300), // 默认配置
		status:              TaskStatusActive,             // 默认激活状态
		createdAt:           now,
		updatedAt:           now,
		domainEvents:        make([]interface{}, 0),
	}

	// 添加领域事件：任务已创建
	task.addDomainEvent(TaskCreatedEvent{
		TaskID:    id,
		Name:      name,
		CreatedAt: now,
	})

	return task, nil
}

// ID 获取任务ID
func (t *Task) ID() types.ID {
	return t.id
}

// Name 获取任务名称
func (t *Task) Name() string {
	return t.name
}

// CronExpression 获取Cron表达式
func (t *Task) CronExpression() CronExpression {
	return t.cronExpression
}

// Parameters 获取参数
func (t *Task) Parameters() types.JSONMap {
	return t.parameters
}

// ExecutionMode 获取执行模式
func (t *Task) ExecutionMode() ExecutionMode {
	return t.executionMode
}

// LoadBalanceStrategy 获取负载均衡策略
func (t *Task) LoadBalanceStrategy() LoadBalanceStrategy {
	return t.loadBalanceStrategy
}

// Configuration 获取配置
func (t *Task) Configuration() TaskConfiguration {
	return t.configuration
}

// Status 获取状态
func (t *Task) Status() TaskStatus {
	return t.status
}

// CreatedAt 获取创建时间
func (t *Task) CreatedAt() time.Time {
	return t.createdAt
}

// UpdatedAt 获取更新时间
func (t *Task) UpdatedAt() time.Time {
	return t.updatedAt
}

// 业务方法

// CanBeScheduled 判断任务是否可以被调度
func (t *Task) CanBeScheduled() bool {
	return t.status.CanBeScheduled()
}

// CanBeExecuted 判断任务是否可以被执行
func (t *Task) CanBeExecuted() bool {
	return t.status.IsActive()
}

// IsActive 判断任务是否为活跃状态
func (t *Task) IsActive() bool {
	return t.status.IsActive()
}

// IsPaused 判断任务是否已暂停
func (t *Task) IsPaused() bool {
	return t.status == TaskStatusPaused
}

// IsDeleted 判断任务是否已删除
func (t *Task) IsDeleted() bool {
	return t.status == TaskStatusDeleted
}

// AllowsConcurrentExecution 判断是否允许并发执行
func (t *Task) AllowsConcurrentExecution() bool {
	return t.executionMode.AllowsConcurrent()
}

// ShouldSkipOnRunning 判断运行时是否应该跳过
func (t *Task) ShouldSkipOnRunning() bool {
	return t.executionMode.ShouldSkipOnRunning()
}

// RequiresWeightedBalance 判断是否需要加权负载均衡
func (t *Task) RequiresWeightedBalance() bool {
	return t.loadBalanceStrategy.UsesWeight()
}

// HasTimeout 判断是否设置了超时
func (t *Task) HasTimeout() bool {
	return t.configuration.HasTimeout()
}

// GetTimeoutDuration 获取超时时长
func (t *Task) GetTimeoutDuration() time.Duration {
	return t.configuration.GetTimeoutDuration()
}

// GetNextExecutionTime 获取下次执行时间
func (t *Task) GetNextExecutionTime(from time.Time) (time.Time, error) {
	if !t.CanBeScheduled() {
		return time.Time{}, fmt.Errorf("task cannot be scheduled in current status: %s", t.status)
	}
	return t.cronExpression.NextTime(from)
}

// 状态变更方法

// Pause 暂停任务
func (t *Task) Pause() error {
	if t.status == TaskStatusPaused {
		return fmt.Errorf("task is already paused")
	}
	if t.status == TaskStatusDeleted {
		return fmt.Errorf("cannot pause deleted task")
	}

	previousStatus := t.status
	t.status = TaskStatusPaused
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskStatusChangedEvent{
		TaskID:         t.id,
		PreviousStatus: previousStatus,
		CurrentStatus:  t.status,
		ChangedAt:      t.updatedAt,
	})

	return nil
}

// Resume 恢复任务
func (t *Task) Resume() error {
	if t.status == TaskStatusActive {
		return fmt.Errorf("task is already active")
	}
	if t.status == TaskStatusDeleted {
		return fmt.Errorf("cannot resume deleted task")
	}

	previousStatus := t.status
	t.status = TaskStatusActive
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskStatusChangedEvent{
		TaskID:         t.id,
		PreviousStatus: previousStatus,
		CurrentStatus:  t.status,
		ChangedAt:      t.updatedAt,
	})

	return nil
}

// Delete 删除任务（软删除）
func (t *Task) Delete() error {
	if t.status == TaskStatusDeleted {
		return fmt.Errorf("task is already deleted")
	}

	previousStatus := t.status
	t.status = TaskStatusDeleted
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskDeletedEvent{
		TaskID:    t.id,
		Name:      t.name,
		DeletedAt: t.updatedAt,
	})

	return nil
}

// 更新方法

// UpdateName 更新任务名称
func (t *Task) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if name == t.name {
		return nil // 没有变化，直接返回
	}

	oldName := t.name
	t.name = name
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskUpdatedEvent{
		TaskID:    t.id,
		Field:     "name",
		OldValue:  oldName,
		NewValue:  name,
		UpdatedAt: t.updatedAt,
	})

	return nil
}

// UpdateCronExpression 更新Cron表达式
func (t *Task) UpdateCronExpression(cronExpr string) error {
	newCronExpression, err := NewCronExpression(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	if newCronExpression == t.cronExpression {
		return nil // 没有变化，直接返回
	}

	oldExpr := t.cronExpression
	t.cronExpression = newCronExpression
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskUpdatedEvent{
		TaskID:    t.id,
		Field:     "cron_expression",
		OldValue:  oldExpr.String(),
		NewValue:  newCronExpression.String(),
		UpdatedAt: t.updatedAt,
	})

	return nil
}

// UpdateParameters 更新参数
func (t *Task) UpdateParameters(parameters types.JSONMap) error {
	if parameters == nil {
		parameters = make(types.JSONMap)
	}

	t.parameters = parameters
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskUpdatedEvent{
		TaskID:    t.id,
		Field:     "parameters",
		OldValue:  t.parameters,
		NewValue:  parameters,
		UpdatedAt: t.updatedAt,
	})

	return nil
}

// UpdateExecutionMode 更新执行模式
func (t *Task) UpdateExecutionMode(mode ExecutionMode) error {
	if !mode.IsValid() {
		return fmt.Errorf("invalid execution mode: %s", mode)
	}

	if mode == t.executionMode {
		return nil // 没有变化，直接返回
	}

	oldMode := t.executionMode
	t.executionMode = mode
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskUpdatedEvent{
		TaskID:    t.id,
		Field:     "execution_mode",
		OldValue:  oldMode.String(),
		NewValue:  mode.String(),
		UpdatedAt: t.updatedAt,
	})

	return nil
}

// UpdateLoadBalanceStrategy 更新负载均衡策略
func (t *Task) UpdateLoadBalanceStrategy(strategy LoadBalanceStrategy) error {
	if !strategy.IsValid() {
		return fmt.Errorf("invalid load balance strategy: %s", strategy)
	}

	if strategy == t.loadBalanceStrategy {
		return nil // 没有变化，直接返回
	}

	oldStrategy := t.loadBalanceStrategy
	t.loadBalanceStrategy = strategy
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskUpdatedEvent{
		TaskID:    t.id,
		Field:     "load_balance_strategy",
		OldValue:  oldStrategy.String(),
		NewValue:  strategy.String(),
		UpdatedAt: t.updatedAt,
	})

	return nil
}

// UpdateConfiguration 更新配置
func (t *Task) UpdateConfiguration(config TaskConfiguration) error {
	if !config.IsValid() {
		return fmt.Errorf("invalid task configuration")
	}

	oldConfig := t.configuration
	t.configuration = config
	t.updatedAt = time.Now()

	// 添加领域事件
	t.addDomainEvent(TaskUpdatedEvent{
		TaskID:    t.id,
		Field:     "configuration",
		OldValue:  oldConfig,
		NewValue:  config,
		UpdatedAt: t.updatedAt,
	})

	return nil
}

// UpdateAll 批量更新（用于UpdateTaskRequest）
func (t *Task) UpdateAll(req UpdateTaskRequest) error {
	// 验证请求
	if err := req.Validate(); err != nil {
		return err
	}

	// 更新名称
	if req.Name != "" {
		if err := t.UpdateName(req.Name); err != nil {
			return err
		}
	}

	// 更新Cron表达式
	if req.CronExpression != "" {
		if err := t.UpdateCronExpression(req.CronExpression); err != nil {
			return err
		}
	}

	// 更新参数
	if req.Parameters != nil {
		if err := t.UpdateParameters(req.Parameters); err != nil {
			return err
		}
	}

	// 更新执行模式
	if req.ExecutionMode != "" {
		if err := t.UpdateExecutionMode(req.ExecutionMode); err != nil {
			return err
		}
	}

	// 更新负载均衡策略
	if req.LoadBalanceStrategy != "" {
		if err := t.UpdateLoadBalanceStrategy(req.LoadBalanceStrategy); err != nil {
			return err
		}
	}

	// 更新配置
	if req.MaxRetry > 0 || req.TimeoutSeconds > 0 {
		maxRetry := t.configuration.MaxRetry
		timeoutSeconds := t.configuration.TimeoutSeconds

		if req.MaxRetry > 0 {
			maxRetry = req.MaxRetry
		}
		if req.TimeoutSeconds > 0 {
			timeoutSeconds = req.TimeoutSeconds
		}

		newConfig := NewTaskConfiguration(maxRetry, timeoutSeconds)
		if err := t.UpdateConfiguration(newConfig); err != nil {
			return err
		}
	}

	// 更新状态
	if req.Status != 0 && req.Status != t.status {
		switch req.Status {
		case TaskStatusActive:
			return t.Resume()
		case TaskStatusPaused:
			return t.Pause()
		case TaskStatusDeleted:
			return t.Delete()
		}
	}

	return nil
}

// 领域事件处理

// GetDomainEvents 获取领域事件
func (t *Task) GetDomainEvents() []interface{} {
	events := make([]interface{}, len(t.domainEvents))
	copy(events, t.domainEvents)
	return events
}

// ClearDomainEvents 清除领域事件
func (t *Task) ClearDomainEvents() {
	t.domainEvents = make([]interface{}, 0)
}

// addDomainEvent 添加领域事件
func (t *Task) addDomainEvent(event interface{}) {
	t.domainEvents = append(t.domainEvents, event)
}

// AddDomainEvent 添加领域事件（公共方法，供UseCase使用）
func (t *Task) AddDomainEvent(event interface{}) {
	t.addDomainEvent(event)
}

// 验证方法

// Validate 验证任务的完整性
func (t *Task) Validate() error {
	if t.id.IsZero() {
		return fmt.Errorf("task ID cannot be empty")
	}
	if t.name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if t.cronExpression.IsEmpty() {
		return fmt.Errorf("cron expression cannot be empty")
	}
	if !t.executionMode.IsValid() {
		return fmt.Errorf("invalid execution mode: %s", t.executionMode)
	}
	if !t.loadBalanceStrategy.IsValid() {
		return fmt.Errorf("invalid load balance strategy: %s", t.loadBalanceStrategy)
	}
	if !t.configuration.IsValid() {
		return fmt.Errorf("invalid task configuration")
	}
	return nil
}
