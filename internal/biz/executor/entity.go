package executor

import (
	"fmt"
	"time"
)

type Executor struct {
	ID                  uint64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Name                string
	InstanceID          string
	BaseURL             string
	HealthCheckURL      string
	Status              ExecutorStatus
	IsHealthy           bool
	LastHealthCheck     *time.Time
	HealthCheckFailures int
	Metadata            map[string]any
}

func (e *Executor) GetLastHealthCheck() int64 {
	if e.LastHealthCheck == nil {
		return 0
	}
	return e.LastHealthCheck.Unix()
}

func (e *Executor) GetHealthCheckURL() string {
	if e.HealthCheckURL != "" {
		return e.HealthCheckURL
	}
	return fmt.Sprintf("%s/health", e.BaseURL)
}

func (e *Executor) GetStopURL() string {
	return fmt.Sprintf("%s/stop", e.BaseURL)
}

func (e *Executor) GetExecURL() string {
	return fmt.Sprintf("%s/execute", e.BaseURL)
}

func (e *Executor) SetStatus(status ExecutorStatus) *ExecutorPatch {
	e.Status = status

	patch := NewExecutorPatch()
	patch.WithStatus(status)

	if status == ExecutorStatusOnline {
		patch.With(e.SetToOnline())
	}

	return patch
}

func (e *Executor) SetToOnline() *ExecutorPatch {
	e.Status = ExecutorStatusOnline
	e.IsHealthy = true
	e.HealthCheckFailures = 0
	return NewExecutorPatch().WithStatus(e.Status).WithIsHealthy(e.IsHealthy).WithHealthCheckFailures(e.HealthCheckFailures)
}

type ExecutorPatch struct {
	Name                *string
	InstanceID          *string
	BaseURL             *string
	HealthCheckURL      *string
	Status              *ExecutorStatus
	IsHealthy           *bool
	LastHealthCheck     *time.Time
	HealthCheckFailures *int
	Metadata            *map[string]any
}

func NewExecutorPatch() *ExecutorPatch {
	return new(ExecutorPatch)
}

func (e *ExecutorPatch) With(other *ExecutorPatch) *ExecutorPatch {
	if other.Name != nil {
		e.Name = other.Name
	}
	if other.InstanceID != nil {
		e.InstanceID = other.InstanceID
	}
	if other.BaseURL != nil {
		e.BaseURL = other.BaseURL
	}
	if other.HealthCheckURL != nil {
		e.HealthCheckURL = other.HealthCheckURL
	}
	if other.Status != nil {
		e.Status = other.Status
	}
	if other.IsHealthy != nil {
		e.IsHealthy = other.IsHealthy
	}
	if other.LastHealthCheck != nil {
		e.LastHealthCheck = other.LastHealthCheck
	}
	if other.HealthCheckFailures != nil {
		e.HealthCheckFailures = other.HealthCheckFailures
	}
	if other.Metadata != nil {
		e.Metadata = other.Metadata
	}
	return e
}

func (e *ExecutorPatch) WithName(name string) *ExecutorPatch {
	e.Name = &name
	return e
}

func (e *ExecutorPatch) WithInstanceID(instanceID string) *ExecutorPatch {
	e.InstanceID = &instanceID
	return e
}

func (e *ExecutorPatch) WithBaseURL(baseURL string) *ExecutorPatch {
	e.BaseURL = &baseURL
	return e
}

func (e *ExecutorPatch) WithHealthCheckURL(healthCheckURL string) *ExecutorPatch {
	e.HealthCheckURL = &healthCheckURL
	return e
}

func (e *ExecutorPatch) WithStatus(status ExecutorStatus) *ExecutorPatch {
	e.Status = &status
	return e
}

func (e *ExecutorPatch) WithIsHealthy(isHealthy bool) *ExecutorPatch {
	e.IsHealthy = &isHealthy
	return e
}

func (e *ExecutorPatch) WithLastHealthCheck(lastHealthCheck time.Time) *ExecutorPatch {
	e.LastHealthCheck = &lastHealthCheck
	return e
}

func (e *ExecutorPatch) WithHealthCheckFailures(healthCheckFailures int) *ExecutorPatch {
	e.HealthCheckFailures = &healthCheckFailures
	return e
}

func (e *ExecutorPatch) WithMetadata(metadata map[string]any) *ExecutorPatch {
	e.Metadata = &metadata
	return e
}
