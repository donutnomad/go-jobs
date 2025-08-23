package entity

import (
	"errors"
	"time"
)

// TaskExecution 任务执行记录领域实体
type TaskExecution struct {
	ID            string
	TaskID        string
	ExecutorID    *string
	ScheduledTime time.Time
	StartTime     *time.Time
	EndTime       *time.Time
	Status        ExecutionStatus
	Result        map[string]any
	Logs          string
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// 关联实体
	Task     *Task
	Executor *Executor
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusSuccess   ExecutionStatus = "success"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusTimeout   ExecutionStatus = "timeout"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// NewTaskExecution 创建新的任务执行记录
func NewTaskExecution(taskID string) *TaskExecution {
	now := time.Now()
	return &TaskExecution{
		TaskID:        taskID,
		ScheduledTime: now,
		Status:        ExecutionStatusPending,
		Result:        make(map[string]any),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// Start 开始执行
func (te *TaskExecution) Start(executorID string) error {
	if te.Status != ExecutionStatusPending {
		return errors.New("只能开始待执行状态的任务")
	}

	now := time.Now()
	te.Status = ExecutionStatusRunning
	te.ExecutorID = &executorID
	te.StartTime = &now
	te.UpdatedAt = now
	return nil
}

// Complete 完成执行
func (te *TaskExecution) Complete(status ExecutionStatus, result map[string]any, logs string) error {
	if te.Status != ExecutionStatusRunning {
		return errors.New("只能完成正在运行的任务")
	}

	if status != ExecutionStatusSuccess && status != ExecutionStatusFailed && status != ExecutionStatusTimeout {
		return errors.New("无效的完成状态")
	}

	now := time.Now()
	te.Status = status
	te.EndTime = &now
	te.Result = result
	te.Logs = logs
	te.UpdatedAt = now
	return nil
}

// Cancel 取消执行
func (te *TaskExecution) Cancel() error {
	if te.Status != ExecutionStatusPending && te.Status != ExecutionStatusRunning {
		return errors.New("只能取消待执行或正在运行的任务")
	}

	now := time.Now()
	te.Status = ExecutionStatusCancelled
	if te.StartTime == nil {
		te.StartTime = &now
	}
	te.EndTime = &now
	te.UpdatedAt = now
	return nil
}

// IsCompleted 判断是否已完成
func (te *TaskExecution) IsCompleted() bool {
	return te.Status == ExecutionStatusSuccess ||
		te.Status == ExecutionStatusFailed ||
		te.Status == ExecutionStatusTimeout ||
		te.Status == ExecutionStatusCancelled
}

// Duration 获取执行时长
func (te *TaskExecution) Duration() time.Duration {
	if te.StartTime == nil {
		return 0
	}
	endTime := time.Now()
	if te.EndTime != nil {
		endTime = *te.EndTime
	}
	return endTime.Sub(*te.StartTime)
}

// IsSuccess 判断是否成功
func (te *TaskExecution) IsSuccess() bool {
	return te.Status == ExecutionStatusSuccess
}
