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
	return executors[rand.Intn(len(executors))], nil
}

func (s *RandomStrategy) Name() string {
	return "random"
}
