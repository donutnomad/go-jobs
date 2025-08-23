package entity

import (
	"errors"
	"time"
)

// Executor 执行器领域实体
type Executor struct {
	ID                  string
	Name                string
	InstanceID          string
	BaseURL             string
	HealthCheckURL      string
	Status              ExecutorStatus
	IsHealthy           bool
	HealthCheckFailures int
	LastHealthCheck     *time.Time
	Metadata            map[string]any
	CreatedAt           time.Time
	UpdatedAt           time.Time
	TaskExecutors       []TaskExecutor
}

// ExecutorStatus 执行器状态
type ExecutorStatus string

const (
	ExecutorStatusOnline      ExecutorStatus = "online"
	ExecutorStatusOffline     ExecutorStatus = "offline"
	ExecutorStatusMaintenance ExecutorStatus = "maintenance"
	ExecutorStatusError       ExecutorStatus = "error"
)

// NewExecutor 创建新执行器
func NewExecutor(name, instanceID, baseURL, healthCheckURL string, metadata map[string]any) (*Executor, error) {
	if name == "" {
		return nil, errors.New("执行器名称不能为空")
	}
	if instanceID == "" {
		return nil, errors.New("实例ID不能为空")
	}
	if baseURL == "" {
		return nil, errors.New("基础URL不能为空")
	}

	now := time.Now()
	return &Executor{
		Name:                name,
		InstanceID:          instanceID,
		BaseURL:             baseURL,
		HealthCheckURL:      healthCheckURL,
		Status:              ExecutorStatusOnline,
		IsHealthy:           true,
		HealthCheckFailures: 0,
		LastHealthCheck:     &now,
		Metadata:            metadata,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// UpdateStatus 更新执行器状态
func (e *Executor) UpdateStatus(status ExecutorStatus) {
	e.Status = status
	e.UpdatedAt = time.Now()
}

// SetHealthy 设置健康状态
func (e *Executor) SetHealthy(isHealthy bool) {
	e.IsHealthy = isHealthy
	now := time.Now()
	e.LastHealthCheck = &now

	if isHealthy {
		e.HealthCheckFailures = 0
		if e.Status == ExecutorStatusError {
			e.Status = ExecutorStatusOnline
		}
	} else {
		e.HealthCheckFailures++
		if e.HealthCheckFailures >= 3 {
			e.Status = ExecutorStatusError
		}
	}
	e.UpdatedAt = now
}

// Update 更新执行器信息
func (e *Executor) Update(name, baseURL, healthCheckURL string) {
	if name != "" {
		e.Name = name
	}
	if baseURL != "" {
		e.BaseURL = baseURL
	}
	if healthCheckURL != "" {
		e.HealthCheckURL = healthCheckURL
	}
	e.UpdatedAt = time.Now()
}

// IsOnline 判断执行器是否在线
func (e *Executor) IsOnline() bool {
	return e.Status == ExecutorStatusOnline && e.IsHealthy
}

// GetStopURL 获取停止任务的URL
func (e *Executor) GetStopURL() string {
	return e.BaseURL + "/stop"
}

// GetExecuteURL 获取执行任务的URL
func (e *Executor) GetExecuteURL() string {
	return e.BaseURL + "/execute"
}
