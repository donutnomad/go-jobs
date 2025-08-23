package models

import (
	"database/sql/driver"
	"encoding/json"
)

type ExecutionMode string

const (
	ExecutionModeSequential ExecutionMode = "sequential"
	ExecutionModeParallel   ExecutionMode = "parallel"
	ExecutionModeSkip       ExecutionMode = "skip"
)

type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin         LoadBalanceStrategy = "round_robin"
	LoadBalanceWeightedRoundRobin LoadBalanceStrategy = "weighted_round_robin"
	LoadBalanceRandom             LoadBalanceStrategy = "random"
	LoadBalanceSticky             LoadBalanceStrategy = "sticky"
	LoadBalanceLeastLoaded        LoadBalanceStrategy = "least_loaded"
)

type ExecutorStatus string

const (
	ExecutorStatusOnline      ExecutorStatus = "online"
	ExecutorStatusOffline     ExecutorStatus = "offline"
	ExecutorStatusMaintenance ExecutorStatus = "maintenance"
)

func (s ExecutorStatus) ToInt() int {
	switch s {
	case ExecutorStatusOnline:
		return 1
	case ExecutorStatusOffline:
		return 3
	case ExecutorStatusMaintenance:
		return 2
	}
	return 0
}

type ExecutionStatus string

const (
	ExecutionStatusPending ExecutionStatus = "pending"
	ExecutionStatusRunning ExecutionStatus = "running"
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusFailed  ExecutionStatus = "failed"
	ExecutionStatusTimeout ExecutionStatus = "timeout"
	ExecutionStatusSkipped ExecutionStatus = "skipped"
)

type JSONMap map[string]any

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}
