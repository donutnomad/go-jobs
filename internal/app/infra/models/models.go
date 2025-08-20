package models

import (
	"time"

	executionbiz "github.com/jobs/scheduler/internal/app/biz/execution"
	executorbiz "github.com/jobs/scheduler/internal/app/biz/executor"
	schedulerbiz "github.com/jobs/scheduler/internal/app/biz/scheduler"
	taskbiz "github.com/jobs/scheduler/internal/app/biz/task"
	"github.com/jobs/scheduler/internal/app/types"
	"gorm.io/gorm"
)

// TaskModel 任务数据模型
type TaskModel struct {
	ID                  string        `gorm:"primaryKey;type:varchar(64)" json:"id"`
	Name                string        `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	CronExpression      string        `gorm:"type:varchar(100);not null" json:"cron_expression"`
	Parameters          types.JSONMap `gorm:"type:json" json:"parameters"`
	ExecutionMode       string        `gorm:"type:varchar(20);not null;default:'parallel'" json:"execution_mode"`
	LoadBalanceStrategy string        `gorm:"type:varchar(20);not null;default:'round_robin'" json:"load_balance_strategy"`
	MaxRetry            int           `gorm:"not null;default:3" json:"max_retry"`
	TimeoutSeconds      int           `gorm:"not null;default:300" json:"timeout_seconds"`
	Status              int           `gorm:"not null;default:1" json:"status"`
	CreatedAt           time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time     `gorm:"not null" json:"updated_at"`

	// 关联关系
	Executors  []ExecutorModel      `gorm:"many2many:task_executors;" json:"executors,omitempty"`
	Executions []TaskExecutionModel `gorm:"foreignKey:TaskID" json:"executions,omitempty"`
}

// TableName 表名
func (TaskModel) TableName() string {
	return "tasks"
}

// ToEntity 转换为领域实体
func (m *TaskModel) ToEntity() (*taskbiz.Task, error) {
	// 这里需要调用领域实体的工厂方法
	task, err := taskbiz.NewTask(m.Name, m.CronExpression)
	if err != nil {
		return nil, err
	}

	// 设置其他属性（需要通过反射或其他方式，这里简化处理）
	// 实际实现中可能需要使用更复杂的映射逻辑

	return task, nil
}

// FromEntity 从领域实体转换
func (m *TaskModel) FromEntity(task *taskbiz.Task) {
	m.ID = string(task.ID())
	m.Name = task.Name()
	m.CronExpression = task.CronExpression().String()
	m.Parameters = task.Parameters()
	m.ExecutionMode = string(task.ExecutionMode())
	m.LoadBalanceStrategy = string(task.LoadBalanceStrategy())
	m.MaxRetry = task.Configuration().MaxRetry
	m.TimeoutSeconds = task.Configuration().TimeoutSeconds
	m.Status = int(task.Status())
	m.CreatedAt = task.CreatedAt()
	m.UpdatedAt = task.UpdatedAt()
}

// ExecutorModel 执行器数据模型
type ExecutorModel struct {
	ID                  string        `gorm:"primaryKey;type:varchar(64)" json:"id"`
	Name                string        `gorm:"type:varchar(100);not null" json:"name"`
	InstanceID          string        `gorm:"type:varchar(100);uniqueIndex;not null" json:"instance_id"`
	BaseURL             string        `gorm:"type:varchar(255);not null" json:"base_url"`
	HealthCheckURL      string        `gorm:"type:varchar(255)" json:"health_check_url"`
	Status              int           `gorm:"not null;default:1" json:"status"`
	IsHealthy           bool          `gorm:"not null;default:true" json:"is_healthy"`
	HealthCheckFailures int           `gorm:"not null;default:0" json:"health_check_failures"`
	LastHealthCheckAt   *time.Time    `json:"last_health_check_at"`
	Tags                types.JSONMap `gorm:"type:json" json:"tags"`
	Capacity            int           `gorm:"not null;default:10" json:"capacity"`
	CreatedAt           time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time     `gorm:"not null" json:"updated_at"`

	// 关联关系
	Tasks      []TaskModel          `gorm:"many2many:task_executors;" json:"tasks,omitempty"`
	Executions []TaskExecutionModel `gorm:"foreignKey:ExecutorID" json:"executions,omitempty"`
}

// TableName 表名
func (ExecutorModel) TableName() string {
	return "executors"
}

// ToEntity 转换为领域实体
func (m *ExecutorModel) ToEntity() (*executorbiz.Executor, error) {
	executor, err := executorbiz.NewExecutor(m.Name, m.InstanceID, m.BaseURL)
	if err != nil {
		return nil, err
	}

	// 设置其他属性（简化处理）
	return executor, nil
}

// FromEntity 从领域实体转换
func (m *ExecutorModel) FromEntity(executor *executorbiz.Executor) {
	m.ID = string(executor.ID())
	m.Name = executor.Name()
	m.InstanceID = executor.InstanceID()
	m.BaseURL = executor.Config().BaseURL
	m.HealthCheckURL = executor.Config().HealthCheckURL
	m.Status = int(executor.Status())
	m.IsHealthy = executor.IsHealthy()
	m.HealthCheckFailures = executor.HealthStatus().HealthCheckFailures
	m.LastHealthCheckAt = executor.HealthStatus().LastCheckTime
	m.Tags = types.JSONMap{"tags": executor.Metadata().Tags}
	m.Capacity = executor.Metadata().Capacity
	m.CreatedAt = executor.CreatedAt()
	m.UpdatedAt = executor.UpdatedAt()
}

// TaskExecutionModel 任务执行数据模型
type TaskExecutionModel struct {
	ID            string        `gorm:"primaryKey;type:varchar(64)" json:"id"`
	TaskID        string        `gorm:"type:varchar(64);not null;index" json:"task_id"`
	ExecutorID    *string       `gorm:"type:varchar(64);index" json:"executor_id"`
	Status        int           `gorm:"not null;default:0" json:"status"`
	Parameters    types.JSONMap `gorm:"type:json" json:"parameters"`
	Result        types.JSONMap `gorm:"type:json" json:"result"`
	Logs          string        `gorm:"type:text" json:"logs"`
	ErrorMessage  string        `gorm:"type:text" json:"error_message"`
	CurrentRetry  int           `gorm:"not null;default:0" json:"current_retry"`
	MaxRetries    int           `gorm:"not null;default:3" json:"max_retries"`
	ScheduledTime time.Time     `gorm:"not null;index" json:"scheduled_time"`
	StartTime     *time.Time    `json:"start_time"`
	EndTime       *time.Time    `json:"end_time"`
	Duration      *int64        `json:"duration"` // 毫秒
	CallbackURL   string        `gorm:"type:varchar(255)" json:"callback_url"`
	CreatedAt     time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time     `gorm:"not null" json:"updated_at"`

	// 关联关系
	Task     TaskModel      `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Executor *ExecutorModel `gorm:"foreignKey:ExecutorID" json:"executor,omitempty"`
}

// TableName 表名
func (TaskExecutionModel) TableName() string {
	return "task_executions"
}

// ToEntity 转换为领域实体
func (m *TaskExecutionModel) ToEntity() (*executionbiz.TaskExecution, error) {
	taskID := types.ID(m.TaskID)
	execution, err := executionbiz.NewTaskExecution(taskID, m.Parameters, m.ScheduledTime)
	if err != nil {
		return nil, err
	}

	// 设置其他属性（简化处理）
	return execution, nil
}

// FromEntity 从领域实体转换
func (m *TaskExecutionModel) FromEntity(execution *executionbiz.TaskExecution) {
	m.ID = string(execution.ID())
	m.TaskID = string(execution.TaskID())

	if execution.HasExecutor() {
		executorID := string(execution.GetExecutorID())
		m.ExecutorID = &executorID
	}

	m.Status = int(execution.Status())
	m.Parameters = execution.Context().Parameters
	m.Result = execution.Result().Data
	m.Logs = execution.Result().Logs
	m.ErrorMessage = execution.Result().ErrorMessage
	m.CurrentRetry = execution.CurrentRetry()
	m.MaxRetries = execution.RetryPolicy().MaxRetries
	m.ScheduledTime = execution.GetScheduledTime()
	m.StartTime = execution.Result().StartTime
	m.EndTime = execution.Result().EndTime

	if execution.Result().Duration != nil {
		duration := int64(*execution.Result().Duration / time.Millisecond)
		m.Duration = &duration
	}

	m.CallbackURL = execution.Context().CallbackURL
	m.CreatedAt = execution.CreatedAt()
	m.UpdatedAt = execution.UpdatedAt()
}

// SchedulerInstanceModel 调度器实例数据模型
type SchedulerInstanceModel struct {
	ID               string        `gorm:"primaryKey;type:varchar(64)" json:"id"`
	InstanceID       string        `gorm:"type:varchar(100);uniqueIndex;not null" json:"instance_id"`
	Status           int           `gorm:"not null;default:1" json:"status"`
	LeadershipStatus int           `gorm:"not null;default:0" json:"leadership_status"`
	LeadershipLock   string        `gorm:"type:varchar(100);not null" json:"leadership_lock"`
	ClusterConfig    types.JSONMap `gorm:"type:json" json:"cluster_config"`
	LastHeartbeat    time.Time     `gorm:"not null" json:"last_heartbeat"`
	LeaderElectedAt  *time.Time    `json:"leader_elected_at"`
	CreatedAt        time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt        time.Time     `gorm:"not null" json:"updated_at"`
}

// TableName 表名
func (SchedulerInstanceModel) TableName() string {
	return "scheduler_instances"
}

// ToEntity 转换为领域实体
func (m *SchedulerInstanceModel) ToEntity() (*schedulerbiz.Scheduler, error) {
	// 构造集群配置
	config := schedulerbiz.ClusterConfig{} // 从JSON解析

	scheduler, err := schedulerbiz.NewScheduler(m.InstanceID, config)
	if err != nil {
		return nil, err
	}

	// 设置其他属性（简化处理）
	return scheduler, nil
}

// FromEntity 从领域实体转换
func (m *SchedulerInstanceModel) FromEntity(scheduler *schedulerbiz.Scheduler) {
	m.ID = string(scheduler.ID())
	m.InstanceID = scheduler.InstanceID()
	m.Status = int(scheduler.Status())
	m.LeadershipStatus = int(scheduler.LeadershipStatus())
	m.LeadershipLock = scheduler.LeadershipLock()
	m.ClusterConfig = types.JSONMap{"config": scheduler.ClusterConfig()}
	m.LastHeartbeat = scheduler.LastHeartbeat()
	m.LeaderElectedAt = scheduler.LeaderElectedAt()
	m.CreatedAt = scheduler.CreatedAt()
	m.UpdatedAt = scheduler.UpdatedAt()
}

// TaskExecutorModel 任务执行器关联模型
type TaskExecutorModel struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TaskID     string    `gorm:"type:varchar(64);not null;index" json:"task_id"`
	ExecutorID string    `gorm:"type:varchar(64);not null;index" json:"executor_id"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`

	// 关联关系
	Task     TaskModel     `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Executor ExecutorModel `gorm:"foreignKey:ExecutorID" json:"executor,omitempty"`
}

// TableName 表名
func (TaskExecutorModel) TableName() string {
	return "task_executors"
}

// LoadBalanceStateModel 负载均衡状态模型
type LoadBalanceStateModel struct {
	ID           uint          `gorm:"primaryKey" json:"id"`
	TaskID       string        `gorm:"type:varchar(64);not null;uniqueIndex" json:"task_id"`
	Strategy     string        `gorm:"type:varchar(20);not null" json:"strategy"`
	CurrentIndex int           `gorm:"not null;default:0" json:"current_index"`
	State        types.JSONMap `gorm:"type:json" json:"state"`
	UpdatedAt    time.Time     `gorm:"not null" json:"updated_at"`

	// 关联关系
	Task TaskModel `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

// TableName 表名
func (LoadBalanceStateModel) TableName() string {
	return "load_balance_states"
}
