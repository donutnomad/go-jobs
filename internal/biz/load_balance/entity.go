package load_balance

import (
	"time"
)

type LoadBalanceState struct {
	ID               uint64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	TaskID           uint64
	LastExecutorID   *uint64
	RoundRobinIndex  int
	StickyExecutorID *uint64
}
