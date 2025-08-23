package taskrepo

import (
	domain "github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"gorm.io/datatypes"
)

type TaskPo struct {
	commonrepo.Mode
	Name                string                     `gorm:"column:name;uniqueIndex;size:255;not null"`                           // name唯一
	CronExpression      string                     `gorm:"column:cron_expression;size:100;not null"`                            // corn表达式
	Parameters          datatypes.JSONMap          `gorm:"column:parameters;type:json"`                                         // 任务输入的参数
	ExecutionMode       domain.ExecutionMode       `gorm:"column:execution_mode;size:50;not null;default:'parallel'"`           // 任务执行模式
	LoadBalanceStrategy domain.LoadBalanceStrategy `gorm:"column:load_balance_strategy;size:50;not null;default:'round_robin'"` // 负载均衡策略
	Status              domain.TaskStatus          `gorm:"column:status;size:50;not null;index"`                                // 任务状态

	MaxRetry       int `gorm:"column:max_retry;default:3"`         // 最大重试次数
	TimeoutSeconds int `gorm:"column:timeout_seconds;default:300"` // 任务超时时间
}

func (t *TaskPo) TableName() string {
	return "jobs_task"
}
