package api

import (
	"time"

	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

type CreateTaskReq struct {
	Name                string                   `json:"name" binding:"required"`
	CronExpression      string                   `json:"cron_expression" binding:"required"`
	Parameters          map[string]any           `json:"parameters"`
	ExecutionMode       task.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy task.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                      `json:"max_retry"`
	TimeoutSeconds      int                      `json:"timeout_seconds"`
}

func (r *CreateTaskReq) GetExecutionMode() task.ExecutionMode {
	if r.ExecutionMode == "" {
		return task.ExecutionModeParallel
	}
	return r.ExecutionMode
}

func (r *CreateTaskReq) GetLoadBalanceStrategy() task.LoadBalanceStrategy {
	if r.LoadBalanceStrategy == "" {
		return task.LoadBalanceRoundRobin
	}
	return r.LoadBalanceStrategy
}

func (r *CreateTaskReq) GetMaxRetry() int {
	if r.MaxRetry == 0 {
		return 3
	}
	return r.MaxRetry
}

func (r *CreateTaskReq) GetTimeoutSeconds() int {
	if r.TimeoutSeconds == 0 {
		return 300
	}
	return r.TimeoutSeconds
}

type TaskResp struct {
	ID                  uint64                   `json:"id"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           time.Time                `json:"updated_at"`
	Name                string                   `json:"name"`
	CronExpression      string                   `json:"cron_expression"`
	Parameters          map[string]any           `json:"parameters"`
	ExecutionMode       task.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy task.LoadBalanceStrategy `json:"load_balance_strategy"`
	Status              task.TaskStatus          `json:"status"`
	MaxRetry            int                      `json:"max_retry"`
	TimeoutSeconds      int                      `json:"timeout_seconds"`
}

type TaskWithAssignmentsResp struct {
	*TaskResp
	Assignments []TaskAssignmentResp `json:"task_executors"`
}

func (t *TaskWithAssignmentsResp) FromDomain(in *task.Task) *TaskWithAssignmentsResp {
	return &TaskWithAssignmentsResp{
		TaskResp: t.TaskResp.FromDomain(in),
		Assignments: lo.Map(in.Assignments, func(assignment *task.TaskAssignment, _ int) TaskAssignmentResp {
			return *new(TaskAssignmentResp).FromDomain(assignment)
		}),
	}
}

type TaskAssignmentResp struct {
	ID           uint64        `json:"id"`
	CreatedAt    time.Time     `json:"created_at"`
	TaskID       uint64        `json:"task_id"`
	ExecutorName string        `json:"executor_name"`
	Priority     int           `json:"priority"`
	Weight       int           `json:"weight"`
	Executor     *ExecutorResp `json:"executor"`
}

func (t *TaskAssignmentResp) FromDomain(in *task.TaskAssignment) *TaskAssignmentResp {
	return &TaskAssignmentResp{
		ID:           in.ID,
		CreatedAt:    in.CreatedAt,
		TaskID:       in.TaskID,
		ExecutorName: in.ExecutorName,
		Priority:     in.Priority,
		Weight:       in.Weight,
	}
}

func (t *TaskResp) FromDomain(in *task.Task) *TaskResp {
	return &TaskResp{
		ID:                  in.ID,
		Name:                in.Name,
		CronExpression:      in.CronExpression,
		Parameters:          in.Parameters,
		ExecutionMode:       in.ExecutionMode,
		LoadBalanceStrategy: in.LoadBalanceStrategy,
		Status:              in.Status,
		MaxRetry:            in.MaxRetry,
		TimeoutSeconds:      in.TimeoutSeconds,
		CreatedAt:           in.CreatedAt,
		UpdatedAt:           in.UpdatedAt,
	}
}

type UpdateTaskReq struct {
	Name                string                   `json:"name"`
	CronExpression      string                   `json:"cron_expression"`
	Parameters          map[string]any           `json:"parameters"`
	ExecutionMode       task.ExecutionMode       `json:"execution_mode"`
	LoadBalanceStrategy task.LoadBalanceStrategy `json:"load_balance_strategy"`
	MaxRetry            int                      `json:"max_retry"`
	TimeoutSeconds      int                      `json:"timeout_seconds"`
	Status              task.TaskStatus          `json:"status"`
}

type AssignExecutorReq struct {
	ExecutorID string `json:"executor_id" binding:"required"`
	Priority   int    `json:"priority"`
	Weight     int    `json:"weight"`
}

func (r AssignExecutorReq) GetExecutorID() uint64 {
	return cast.ToUint64(r.ExecutorID)
}

type UpdateExecutorAssignmentReq struct {
	Priority int `json:"priority"`
	Weight   int `json:"weight"`
}

type TaskStatsResp struct {
	SuccessRate24h   float64            `json:"success_rate_24h"`
	Total24h         int64              `json:"total_24h"`
	Success24h       int64              `json:"success_24h"`
	Health90d        HealthStatus       `json:"health_90d"`
	RecentExecutions []RecentExecutions `json:"recent_executions"`
	DailyStats90d    []map[string]any   `json:"daily_stats_90d"`
}

type RecentExecutions struct {
	Date        string  `json:"date"`
	Total       int     `json:"total"`
	Success     int     `json:"success"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

type TriggerTaskRequest struct {
	Parameters map[string]any `json:"parameters"`
}

////// executor API  //////

type ListExecutorReq struct {
}

type UpdateExecutorReq struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	HealthCheckURL string `json:"health_check_url"`
}

type RegisterExecutorReq struct {
	ExecutorID     string           `json:"executor_id" binding:"required"`   // 执行器唯一ID
	ExecutorName   string           `json:"executor_name" binding:"required"` // 执行器名称
	ExecutorURL    string           `json:"executor_url" binding:"required"`  // 执行器URL
	HealthCheckURL string           `json:"health_check_url"`                 // 健康检查URL（可选）
	Tasks          []TaskDefinition `json:"tasks"`                            // 任务定义列表
	Metadata       map[string]any   `json:"metadata"`                         // 元数据
}

type TaskDefinition struct {
	Name                string                   `json:"name" binding:"required"`
	ExecutionMode       task.ExecutionMode       `json:"execution_mode" binding:"required"`
	CronExpression      string                   `json:"cron_expression" binding:"required"`
	LoadBalanceStrategy task.LoadBalanceStrategy `json:"load_balance_strategy" binding:"required"`
	MaxRetry            int                      `json:"max_retry"`
	TimeoutSeconds      int                      `json:"timeout_seconds"`
	Parameters          map[string]any           `json:"parameters"`
	Status              task.TaskStatus          `json:"status"` // 初始状态，可以是 active 或 paused
}

func (d *TaskDefinition) GetMaxRetry() int {
	if d.MaxRetry == 0 {
		return 3
	}
	return d.MaxRetry
}

func (d *TaskDefinition) GetTimeoutSeconds() int {
	if d.TimeoutSeconds == 0 {
		return 300
	}
	return d.TimeoutSeconds
}

func (d *TaskDefinition) GetStatus() task.TaskStatus {
	if d.Status == "" {
		return task.TaskStatusPaused
	}
	return d.Status
}

func (d *TaskDefinition) GetParameters() map[string]any {
	if d.Parameters == nil {
		return make(map[string]any)
	}
	return d.Parameters
}

func (d *TaskDefinition) GetExecutionMode() task.ExecutionMode {
	if d.ExecutionMode == "" {
		return task.ExecutionModeParallel
	}
	return d.ExecutionMode
}

func (d *TaskDefinition) GetLoadBalanceStrategy() task.LoadBalanceStrategy {
	if d.LoadBalanceStrategy == "" {
		return task.LoadBalanceRoundRobin
	}
	return d.LoadBalanceStrategy
}

type UpdateExecutorStatusReq struct {
	Status executor.ExecutorStatus `json:"status" binding:"required, oneof=online offline maintenance"`
	Reason string                  `json:"reason"`
}

///// execution API //////

type ExecutionCallbackRequest struct {
	ExecutionID string                    `json:"execution_id" binding:"required"`
	Status      execution.ExecutionStatus `json:"status" binding:"required"`
	Result      map[string]any            `json:"result"`
	Logs        string                    `json:"logs"`
}

type ExecutionStatsReq struct {
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	TaskID    uint64 `form:"task_id"`
}

func (r ExecutionStatsReq) GetStartTime() int64 {
	// 2025-08-22T16:00:00.000Z
	startTime, err := time.Parse(time.RFC3339, r.StartTime)
	if err != nil {
		return 0
	}
	return startTime.Unix()
}

func (r ExecutionStatsReq) GetEndTime() int64 {
	endTime, err := time.Parse(time.RFC3339, r.EndTime)
	if err != nil {
		return 0
	}
	return endTime.Unix()
}

type ExecutionStatsResp struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
	Running int64 `json:"running"`
	Pending int64 `json:"pending"`
}

type PageAndSize struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (p PageAndSize) GetOffset() int {
	return (max(1, p.Page) - 1) * p.GetLimit()
}

func (p PageAndSize) GetLimit() int {
	if p.PageSize == 0 {
		return 20
	}
	return p.PageSize
}
func (p PageAndSize) GetTotalPages(total int64) int {
	totalPages := int(total) / p.GetLimit()
	if int(total)%p.GetLimit() > 0 {
		totalPages++
	}
	return totalPages
}

type ListExecutionReq struct {
	PageAndSize
	TaskID    uint64                    `form:"task_id"`
	Status    execution.ExecutionStatus `form:"status"`
	StartTime int64                     `form:"start_time"`
	EndTime   int64                     `form:"end_time"`
}

type ListExecutionResp struct {
	Data       []*TaskExecutionResp `json:"data"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}

////// common ///////

type SchedulerStatsResp struct {
	Instances []SchedulerInstanceResp `json:"instances"`
	Time      time.Time               `json:"time"`
}
type SchedulerInstanceResp struct {
	ID         uint64    `json:"id"`
	InstanceID string    `json:"instance_id"`
	Host       string    `json:"host"`
	Port       int       `json:"port"`
	IsLeader   bool      `json:"is_leader"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TaskExecutionResp struct {
	ID            uint64                    `json:"id"`
	TaskID        uint64                    `json:"task_id"`
	ExecutorID    uint64                    `json:"executor_id"`
	ScheduledTime time.Time                 `json:"scheduled_time"`
	StartTime     *time.Time                `json:"start_time"`
	EndTime       *time.Time                `json:"end_time"`
	Status        execution.ExecutionStatus `json:"status"`
	Result        map[string]any            `json:"result"`
	Logs          string                    `json:"logs"`
	RetryCount    int                       `json:"retry_count"`
	CreatedAt     time.Time                 `json:"created_at"`

	// 在应用层手动填充的关联字段（不使用GORM关联）
	Task     *TaskResp     `json:"task,omitempty"`
	Executor *ExecutorResp `json:"executor,omitempty"`
}

func (t *TaskExecutionResp) FromDomain(in *execution.TaskExecution) *TaskExecutionResp {
	return &TaskExecutionResp{
		ID:            in.ID,
		TaskID:        in.TaskID,
		ExecutorID:    in.ExecutorID,
		ScheduledTime: in.ScheduledTime,
		StartTime:     in.StartTime,
		EndTime:       in.EndTime,
		Status:        in.Status,
		Result:        in.Result,
		Logs:          in.Logs,
		RetryCount:    in.RetryCount,
		CreatedAt:     in.CreatedAt,
		Task:          nil,
		Executor:      nil,
	}
}

type GetTasksReq struct {
	Status task.TaskStatus `form:"status" binding:"omitempty,oneof=active paused deleted"`
}

type ExecutorResp struct {
	ID                  uint64                  `json:"id"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
	Name                string                  `json:"name"`
	InstanceID          string                  `json:"instance_id"`
	BaseURL             string                  `json:"base_url"`
	HealthCheckURL      string                  `json:"health_check_url"`
	Status              executor.ExecutorStatus `json:"status"`
	IsHealthy           bool                    `json:"is_healthy"`
	LastHealthCheck     *time.Time              `json:"last_health_check"`
	HealthCheckFailures int                     `json:"health_check_failures"`
	Metadata            map[string]any          `json:"metadata"`
	TaskAssignments     []*TaskAssignmentResp2  `json:"task_executors,omitempty"`
}

func (t *ExecutorResp) FromDomain(in *executor.Executor) *ExecutorResp {
	return &ExecutorResp{
		ID:                  in.ID,
		CreatedAt:           in.CreatedAt,
		UpdatedAt:           in.UpdatedAt,
		Name:                in.Name,
		BaseURL:             in.BaseURL,
		HealthCheckURL:      in.HealthCheckURL,
		Status:              in.Status,
		IsHealthy:           in.IsHealthy,
		LastHealthCheck:     in.LastHealthCheck,
		HealthCheckFailures: in.HealthCheckFailures,
		Metadata:            in.Metadata,
	}
}

type TaskAssignmentResp2 struct {
	TaskAssignmentResp
	Task *TaskResp `json:"task"`
}

func (t *TaskAssignmentResp2) FromDomain(in *task.TaskAssignment) *TaskAssignmentResp2 {
	return &TaskAssignmentResp2{
		TaskAssignmentResp: *new(TaskAssignmentResp).FromDomain(in),
	}
}
