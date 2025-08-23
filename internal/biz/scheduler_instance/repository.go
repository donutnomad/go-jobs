package scheduler_instance

import (
	"context"

	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

type Repo interface {
	commonrepo.Transaction

	// Create 创建调度器实例
	Create(ctx context.Context, instance *SchedulerInstance) error

	// GetByInstanceID 通过实例ID获取
	GetByInstanceID(ctx context.Context, instanceID string) (*SchedulerInstance, error)

	// UpdateLeaderStatus 更新领导者状态
	UpdateLeaderStatus(ctx context.Context, instanceID string, isLeader bool) error

	// Save 保存调度器实例
	Save(ctx context.Context, instance *SchedulerInstance) error

	List(ctx context.Context) ([]*SchedulerInstance, error)
}
