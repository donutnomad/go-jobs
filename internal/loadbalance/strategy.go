package loadbalance

import (
	"context"

	"github.com/jobs/scheduler/internal/models"
)

// Strategy 负载均衡策略接口
type Strategy interface {
	// Select 选择一个执行器
	Select(ctx context.Context, taskID string, executors []*models.Executor) (*models.Executor, error)
	// Name 策略名称
	Name() string
}
