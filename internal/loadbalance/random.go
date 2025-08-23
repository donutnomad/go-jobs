package loadbalance

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/jobs/scheduler/internal/biz/executor"
)

// RandomStrategy 随机策略
type RandomStrategy struct{}

func NewRandomStrategy() *RandomStrategy {
	return &RandomStrategy{}
}

func (s *RandomStrategy) Select(ctx context.Context, taskID uint64, executors []*executor.Executor) (*executor.Executor, error) {
	if len(executors) == 0 {
		return nil, fmt.Errorf("no available executors")
	}

	// 随机选择一个执行器
	index := rand.Intn(len(executors))
	return executors[index], nil
}

func (s *RandomStrategy) Name() string {
	return "random"
}
