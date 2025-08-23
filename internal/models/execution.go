package models

import (
	"time"
)

type TaskExecution struct {
	ID            string          `gorm:"primaryKey;size:64" json:"id"`
	TaskID        string          `gorm:"size:64;not null;index:idx_task_status" json:"task_id"`
	ExecutorID    *string         `gorm:"size:64" json:"executor_id"`
	ScheduledTime time.Time       `gorm:"not null;index" json:"scheduled_time"`
	StartTime     *time.Time      `gorm:"" json:"start_time"`
	EndTime       *time.Time      `gorm:"" json:"end_time"`
	Status        ExecutionStatus `gorm:"type:enum('pending','running','success','failed','timeout','skipped','cancelled');default:'pending';index:idx_task_status;index" json:"status"`
	Result        JSONMap         `gorm:"type:json" json:"result"`
	Logs          string          `gorm:"type:text" json:"logs"`
	RetryCount    int             `gorm:"default:0" json:"retry_count"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`

	// 在应用层手动填充的关联字段（不使用GORM关联）
	Task     *Task     `gorm:"-" json:"task,omitempty"`
	Executor *Executor `gorm:"-" json:"executor,omitempty"`
}

func (TaskExecution) TableName() string {
	return "task_executions"
}

type LoadBalanceState struct {
	TaskID           string    `gorm:"primaryKey;size:64" json:"task_id"`
	LastExecutorID   *string   `gorm:"size:64" json:"last_executor_id"`
	RoundRobinIndex  int       `gorm:"default:0" json:"round_robin_index"`
	StickyExecutorID *string   `gorm:"size:64" json:"sticky_executor_id"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 移除数据库关联，在应用层实现
	// Task           *Task     `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"task,omitempty"`
	// LastExecutor   *Executor `gorm:"foreignKey:LastExecutorID;constraint:OnDelete:SET NULL" json:"last_executor,omitempty"`
	// StickyExecutor *Executor `gorm:"foreignKey:StickyExecutorID;constraint:OnDelete:SET NULL" json:"sticky_executor,omitempty"`
}

func (LoadBalanceState) TableName() string {
	return "load_balance_state"
}

type SchedulerInstance struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	InstanceID string    `gorm:"size:255;not null;uniqueIndex" json:"instance_id"`
	Host       string    `gorm:"size:255;not null" json:"host"`
	Port       int       `gorm:"not null" json:"port"`
	IsLeader   bool      `gorm:"default:false;index" json:"is_leader"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SchedulerInstance) TableName() string {
	return "scheduler_instances"
}
