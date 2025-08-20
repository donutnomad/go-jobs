package executor

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/app/types"
)

// Executor 执行器领域实体
type Executor struct {
	// 基础属性
	id           types.ID
	name         string
	instanceID   string
	config       ExecutorConfig
	status       ExecutorStatus
	healthStatus HealthStatus
	metadata     ExecutorMetadata

	// 时间戳
	createdAt time.Time
	updatedAt time.Time

	// 领域事件
	domainEvents []interface{}
}

// NewExecutor 创建新执行器（工厂方法）
func NewExecutor(name, instanceID, baseURL string) (*Executor, error) {
	// 验证参数
	if name == "" {
		return nil, fmt.Errorf("executor name cannot be empty")
	}
	if instanceID == "" {
		return nil, fmt.Errorf("instance ID cannot be empty")
	}

	// 创建配置
	config, err := NewExecutorConfig(baseURL, "")
	if err != nil {
		return nil, fmt.Errorf("invalid executor config: %w", err)
	}

	// 生成唯一ID
	id := types.ID(uuid.New().String())

	now := time.Now()
	executor := &Executor{
		id:           id,
		name:         name,
		instanceID:   instanceID,
		config:       config,
		status:       ExecutorStatusOnline,
		healthStatus: NewHealthStatus(),
		metadata:     ExecutorMetadata{},
		createdAt:    now,
		updatedAt:    now,
		domainEvents: make([]interface{}, 0),
	}

	// 添加领域事件
	executor.addDomainEvent(ExecutorRegisteredEvent{
		ExecutorID: id,
		Name:       name,
		InstanceID: instanceID,
		CreatedAt:  now,
	})

	return executor, nil
}

// ID 获取执行器ID
func (e *Executor) ID() types.ID {
	return e.id
}

// Name 获取执行器名称
func (e *Executor) Name() string {
	return e.name
}

// InstanceID 获取实例ID
func (e *Executor) InstanceID() string {
	return e.instanceID
}

// Config 获取配置
func (e *Executor) Config() ExecutorConfig {
	return e.config
}

// Status 获取状态
func (e *Executor) Status() ExecutorStatus {
	return e.status
}

// HealthStatus 获取健康状态
func (e *Executor) HealthStatus() HealthStatus {
	return e.healthStatus
}

// Metadata 获取元数据
func (e *Executor) Metadata() ExecutorMetadata {
	return e.metadata
}

// CreatedAt 获取创建时间
func (e *Executor) CreatedAt() time.Time {
	return e.createdAt
}

// UpdatedAt 获取更新时间
func (e *Executor) UpdatedAt() time.Time {
	return e.updatedAt
}

// 业务方法

// IsAvailable 判断是否可用
func (e *Executor) IsAvailable() bool {
	return e.status.IsAvailable() && e.healthStatus.IsHealthy
}

// CanAcceptTasks 判断是否可以接受任务
func (e *Executor) CanAcceptTasks() bool {
	return e.status.CanAcceptTasks() && e.healthStatus.IsHealthy
}

// IsHealthy 判断是否健康
func (e *Executor) IsHealthy() bool {
	return e.healthStatus.IsHealthy
}

// IsOnline 判断是否在线
func (e *Executor) IsOnline() bool {
	return e.status == ExecutorStatusOnline
}

// IsOffline 判断是否离线
func (e *Executor) IsOffline() bool {
	return e.status == ExecutorStatusOffline
}

// IsInMaintenance 判断是否在维护中
func (e *Executor) IsInMaintenance() bool {
	return e.status == ExecutorStatusMaintenance
}

// NeedsHealthCheck 判断是否需要健康检查
func (e *Executor) NeedsHealthCheck(interval time.Duration) bool {
	return e.healthStatus.NeedsHealthCheck(interval)
}

// GetExecuteURL 获取执行任务的URL
func (e *Executor) GetExecuteURL() string {
	return e.config.GetExecuteURL()
}

// GetStopURL 获取停止任务的URL
func (e *Executor) GetStopURL() string {
	return e.config.GetStopURL()
}

// HasTag 检查是否有特定标签
func (e *Executor) HasTag(tag string) bool {
	return e.metadata.HasTag(tag)
}

// GetCapacity 获取容量
func (e *Executor) GetCapacity() int {
	return e.metadata.GetCapacity()
}

// 状态变更方法

// GoOnline 上线
func (e *Executor) GoOnline() error {
	if e.status == ExecutorStatusOnline {
		return nil // 已经在线
	}

	previousStatus := e.status
	e.status = ExecutorStatusOnline
	e.updatedAt = time.Now()

	// 标记为健康
	e.healthStatus.MarkHealthy()

	// 添加领域事件
	e.addDomainEvent(ExecutorStatusChangedEvent{
		ExecutorID:     e.id,
		PreviousStatus: previousStatus,
		CurrentStatus:  e.status,
		ChangedAt:      e.updatedAt,
	})

	return nil
}

// GoOffline 离线
func (e *Executor) GoOffline(reason string) error {
	if e.status == ExecutorStatusOffline {
		return nil // 已经离线
	}

	previousStatus := e.status
	e.status = ExecutorStatusOffline
	e.updatedAt = time.Now()

	// 标记为不健康
	e.healthStatus.MarkUnhealthy()

	// 添加领域事件
	e.addDomainEvent(ExecutorStatusChangedEvent{
		ExecutorID:     e.id,
		PreviousStatus: previousStatus,
		CurrentStatus:  e.status,
		ChangedAt:      e.updatedAt,
		Reason:         reason,
	})

	return nil
}

// EnterMaintenance 进入维护模式
func (e *Executor) EnterMaintenance(reason string) error {
	if e.status == ExecutorStatusMaintenance {
		return nil // 已经在维护中
	}

	previousStatus := e.status
	e.status = ExecutorStatusMaintenance
	e.updatedAt = time.Now()

	// 添加领域事件
	e.addDomainEvent(ExecutorStatusChangedEvent{
		ExecutorID:     e.id,
		PreviousStatus: previousStatus,
		CurrentStatus:  e.status,
		ChangedAt:      e.updatedAt,
		Reason:         reason,
	})

	return nil
}

// UpdateStatus 更新状态
func (e *Executor) UpdateStatus(status ExecutorStatus, reason string) error {
	switch status {
	case ExecutorStatusOnline:
		return e.GoOnline()
	case ExecutorStatusOffline:
		return e.GoOffline(reason)
	case ExecutorStatusMaintenance:
		return e.EnterMaintenance(reason)
	default:
		return fmt.Errorf("invalid executor status: %s", status)
	}
}

// 健康检查方法

// MarkHealthy 标记为健康
func (e *Executor) MarkHealthy() {
	wasUnhealthy := !e.healthStatus.IsHealthy
	e.healthStatus.MarkHealthy()
	e.updatedAt = time.Now()

	// 如果从不健康恢复到健康，添加事件
	if wasUnhealthy {
		e.addDomainEvent(ExecutorHealthRecoveredEvent{
			ExecutorID:  e.id,
			RecoveredAt: e.updatedAt,
		})
	}
}

// MarkUnhealthy 标记为不健康
func (e *Executor) MarkUnhealthy() {
	wasHealthy := e.healthStatus.IsHealthy
	e.healthStatus.MarkUnhealthy()
	e.updatedAt = time.Now()

	// 如果从健康变为不健康，添加事件
	if wasHealthy {
		e.addDomainEvent(ExecutorHealthDegradedEvent{
			ExecutorID: e.id,
			Failures:   e.healthStatus.HealthCheckFailures,
			DegradedAt: e.updatedAt,
		})
	}

	// 检查是否应该标记为离线
	if e.healthStatus.ShouldMarkOffline() {
		e.GoOffline("health check failed too many times")
	}
}

// 更新方法

// UpdateName 更新名称
func (e *Executor) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("executor name cannot be empty")
	}
	if name == e.name {
		return nil // 没有变化
	}

	oldName := e.name
	e.name = name
	e.updatedAt = time.Now()

	// 添加领域事件
	e.addDomainEvent(ExecutorUpdatedEvent{
		ExecutorID: e.id,
		Field:      "name",
		OldValue:   oldName,
		NewValue:   name,
		UpdatedAt:  e.updatedAt,
	})

	return nil
}

// UpdateConfig 更新配置
func (e *Executor) UpdateConfig(baseURL, healthCheckURL string) error {
	newConfig, err := NewExecutorConfig(baseURL, healthCheckURL)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if newConfig == e.config {
		return nil // 没有变化
	}

	oldConfig := e.config
	e.config = newConfig
	e.updatedAt = time.Now()

	// 添加领域事件
	e.addDomainEvent(ExecutorUpdatedEvent{
		ExecutorID: e.id,
		Field:      "config",
		OldValue:   oldConfig,
		NewValue:   newConfig,
		UpdatedAt:  e.updatedAt,
	})

	return nil
}

// UpdateMetadata 更新元数据
func (e *Executor) UpdateMetadata(metadata ExecutorMetadata) error {
	oldMetadata := e.metadata
	e.metadata = metadata
	e.updatedAt = time.Now()

	// 添加领域事件
	e.addDomainEvent(ExecutorUpdatedEvent{
		ExecutorID: e.id,
		Field:      "metadata",
		OldValue:   oldMetadata,
		NewValue:   metadata,
		UpdatedAt:  e.updatedAt,
	})

	return nil
}

// 领域事件处理

// GetDomainEvents 获取领域事件
func (e *Executor) GetDomainEvents() []interface{} {
	events := make([]interface{}, len(e.domainEvents))
	copy(events, e.domainEvents)
	return events
}

// ClearDomainEvents 清除领域事件
func (e *Executor) ClearDomainEvents() {
	e.domainEvents = make([]interface{}, 0)
}

// addDomainEvent 添加领域事件
func (e *Executor) addDomainEvent(event interface{}) {
	e.domainEvents = append(e.domainEvents, event)
}

// AddDomainEvent 添加领域事件（公共方法，供UseCase使用）
func (e *Executor) AddDomainEvent(event interface{}) {
	e.addDomainEvent(event)
}

// 验证方法

// Validate 验证执行器的完整性
func (e *Executor) Validate() error {
	if e.id.IsZero() {
		return fmt.Errorf("executor ID cannot be empty")
	}
	if e.name == "" {
		return fmt.Errorf("executor name cannot be empty")
	}
	if e.instanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}
	if !e.config.IsValid() {
		return fmt.Errorf("invalid executor config")
	}
	return nil
}
