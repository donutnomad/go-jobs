package executor

import (
	"fmt"
	"net/url"
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// ExecutorStatus 执行器状态
type ExecutorStatus int

const (
	ExecutorStatusOnline ExecutorStatus = iota + 1
	ExecutorStatusOffline
	ExecutorStatusMaintenance
)

func (s ExecutorStatus) String() string {
	switch s {
	case ExecutorStatusOnline:
		return "online"
	case ExecutorStatusOffline:
		return "offline"
	case ExecutorStatusMaintenance:
		return "maintenance"
	default:
		return "unknown"
	}
}

// IsAvailable 判断是否可用
func (s ExecutorStatus) IsAvailable() bool {
	return s == ExecutorStatusOnline
}

// CanAcceptTasks 判断是否可以接受任务
func (s ExecutorStatus) CanAcceptTasks() bool {
	return s == ExecutorStatusOnline
}

// ToInt 转换为整数（用于排序）
func (s ExecutorStatus) ToInt() int {
	switch s {
	case ExecutorStatusOnline:
		return 1
	case ExecutorStatusMaintenance:
		return 2
	case ExecutorStatusOffline:
		return 3
	default:
		return 4
	}
}

// ExecutorConfig 执行器配置值对象
type ExecutorConfig struct {
	BaseURL        string `json:"base_url"`
	HealthCheckURL string `json:"health_check_url"`
}

// NewExecutorConfig 创建执行器配置
func NewExecutorConfig(baseURL, healthCheckURL string) (ExecutorConfig, error) {
	if baseURL == "" {
		return ExecutorConfig{}, fmt.Errorf("base URL is required")
	}

	// 验证URL格式
	if _, err := url.Parse(baseURL); err != nil {
		return ExecutorConfig{}, fmt.Errorf("invalid base URL: %w", err)
	}

	// 如果没有提供健康检查URL，使用默认的
	if healthCheckURL == "" {
		healthCheckURL = baseURL + "/health"
	} else {
		// 验证健康检查URL格式
		if _, err := url.Parse(healthCheckURL); err != nil {
			return ExecutorConfig{}, fmt.Errorf("invalid health check URL: %w", err)
		}
	}

	return ExecutorConfig{
		BaseURL:        baseURL,
		HealthCheckURL: healthCheckURL,
	}, nil
}

// IsValid 验证配置是否有效
func (c ExecutorConfig) IsValid() bool {
	if c.BaseURL == "" {
		return false
	}
	if _, err := url.Parse(c.BaseURL); err != nil {
		return false
	}
	if c.HealthCheckURL != "" {
		if _, err := url.Parse(c.HealthCheckURL); err != nil {
			return false
		}
	}
	return true
}

// GetExecuteURL 获取执行任务的URL
func (c ExecutorConfig) GetExecuteURL() string {
	return c.BaseURL + "/execute"
}

// GetStopURL 获取停止任务的URL
func (c ExecutorConfig) GetStopURL() string {
	return c.BaseURL + "/stop"
}

// HealthStatus 健康状态值对象
type HealthStatus struct {
	IsHealthy           bool       `json:"is_healthy"`
	LastHealthCheck     *time.Time `json:"last_health_check"`
	HealthCheckFailures int        `json:"health_check_failures"`
}

// NewHealthStatus 创建健康状态
func NewHealthStatus() HealthStatus {
	return HealthStatus{
		IsHealthy:           true,
		LastHealthCheck:     nil,
		HealthCheckFailures: 0,
	}
}

// MarkHealthy 标记为健康
func (h *HealthStatus) MarkHealthy() {
	now := time.Now()
	h.IsHealthy = true
	h.LastHealthCheck = &now
	h.HealthCheckFailures = 0
}

// MarkUnhealthy 标记为不健康
func (h *HealthStatus) MarkUnhealthy() {
	now := time.Now()
	h.IsHealthy = false
	h.LastHealthCheck = &now
	h.HealthCheckFailures++
}

// ShouldMarkOffline 判断是否应该标记为离线（连续失败3次）
func (h HealthStatus) ShouldMarkOffline() bool {
	return h.HealthCheckFailures >= 3
}

// CanRecover 判断是否可以恢复（连续成功2次）
func (h HealthStatus) CanRecover() bool {
	return h.IsHealthy && h.HealthCheckFailures == 0
}

// NeedsHealthCheck 判断是否需要健康检查
func (h HealthStatus) NeedsHealthCheck(interval time.Duration) bool {
	if h.LastHealthCheck == nil {
		return true
	}
	return time.Since(*h.LastHealthCheck) >= interval
}

// ExecutorMetadata 执行器元数据
type ExecutorMetadata struct {
	Region   string   `json:"region"`
	Version  string   `json:"version"`
	Tags     []string `json:"tags"`
	Capacity int      `json:"capacity"`
}

// HasTag 检查是否有特定标签
func (m ExecutorMetadata) HasTag(tag string) bool {
	for _, t := range m.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// GetCapacity 获取容量，如果没有设置则返回默认值
func (m ExecutorMetadata) GetCapacity() int {
	if m.Capacity <= 0 {
		return 10 // 默认容量
	}
	return m.Capacity
}

// RegisterRequest 执行器注册请求
type RegisterRequest struct {
	Name           string        `json:"name" binding:"required"`
	InstanceID     string        `json:"instance_id" binding:"required"`
	BaseURL        string        `json:"base_url" binding:"required"`
	HealthCheckURL string        `json:"health_check_url"`
	Metadata       types.JSONMap `json:"metadata"`
}

// Validate 验证注册请求
func (r RegisterRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("executor name is required")
	}
	if r.InstanceID == "" {
		return fmt.Errorf("instance ID is required")
	}
	if r.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	// 验证URL格式
	if _, err := url.Parse(r.BaseURL); err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	if r.HealthCheckURL != "" {
		if _, err := url.Parse(r.HealthCheckURL); err != nil {
			return fmt.Errorf("invalid health check URL: %w", err)
		}
	}

	return nil
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status ExecutorStatus `json:"status" binding:"required"`
	Reason string         `json:"reason"`
}

// ExecutorFilter 执行器过滤器
type ExecutorFilter struct {
	types.Filter
	Status       []string `json:"status"`
	InstanceID   *string  `json:"instance_id"`
	IncludeTasks bool     `json:"include_tasks"`
}

// NewExecutorFilter 创建执行器过滤器
func NewExecutorFilter() ExecutorFilter {
	return ExecutorFilter{
		Filter: types.NewFilter(),
	}
}
