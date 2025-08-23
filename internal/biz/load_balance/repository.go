package load_balance

import (
	"context"

	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

type Repo interface {
	commonrepo.Transaction
	
	// GetByTaskID 通过任务ID获取负载均衡状态
	GetByTaskID(ctx context.Context, taskID uint64) (*LoadBalanceState, error)
	
	// Save 保存负载均衡状态
	Save(ctx context.Context, state *LoadBalanceState) error
	
	// Create 创建负载均衡状态
	Create(ctx context.Context, state *LoadBalanceState) error
}