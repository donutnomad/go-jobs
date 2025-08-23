package load_balance

import (
	"time"
)

type LoadBalanceState struct {
	ID               uint64
	TaskID           uint64
	LastExecutorID   *uint64
	RoundRobinIndex  int
	StickyExecutorID *uint64
	UpdatedAt        time.Time
	CreatedAt        time.Time
}
