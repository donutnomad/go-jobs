package models

import (
	"time"
)

type Task struct {
	ID                  string              `gorm:"primaryKey;size:64" json:"id"`
	Name                string              `gorm:"uniqueIndex;size:255;not null" json:"name"`
	CronExpression      string              `gorm:"size:100;not null" json:"cron_expression"`
	Parameters          JSONMap             `gorm:"type:json" json:"parameters"`
	ExecutionMode       ExecutionMode       `gorm:"type:enum('sequential','parallel','skip');default:'parallel'" json:"execution_mode"`
	LoadBalanceStrategy LoadBalanceStrategy `gorm:"type:enum('round_robin','weighted_round_robin','random','sticky','least_loaded');default:'round_robin'" json:"load_balance_strategy"`
	MaxRetry            int                 `gorm:"default:3" json:"max_retry"`
	TimeoutSeconds      int                 `gorm:"default:300" json:"timeout_seconds"`
	Status              TaskStatus          `gorm:"type:enum('active','paused','deleted');default:'active';index" json:"status"`
	CreatedAt           time.Time           `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time           `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联关系
	TaskExecutors []TaskExecutor  `gorm:"foreignKey:TaskID" json:"task_executors,omitempty"`
	Executions    []TaskExecution `gorm:"foreignKey:TaskID" json:"executions,omitempty"`
}

func (Task) TableName() string {
	return "tasks"
}
