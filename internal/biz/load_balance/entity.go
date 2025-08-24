package load_balance

import (
	"time"

	"github.com/yitter/idgenerator-go/idgen"
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

// NewLoadBalanceStateForTask constructs a default state for a task
// with a generated ID and round-robin index set to 0.
func NewLoadBalanceStateForTask(taskID uint64) *LoadBalanceState {
	return &LoadBalanceState{
		ID:              uint64(idgen.NextId()),
		TaskID:          taskID,
		RoundRobinIndex: 0,
	}
}

// SetLastExecutorID sets the last selected executor id.
func (s *LoadBalanceState) SetLastExecutorID(id uint64) {
	s.LastExecutorID = &id
}

// SetStickyExecutorID sets the sticky executor id.
func (s *LoadBalanceState) SetStickyExecutorID(id uint64) {
	s.StickyExecutorID = &id
}

// AdvanceRoundRobin advances the round-robin index by 1 modulo total.
func (s *LoadBalanceState) AdvanceRoundRobin(total int) {
	if total <= 0 {
		return
	}
	s.RoundRobinIndex = (s.RoundRobinIndex + 1) % total
}

// CurrentIndex returns the current round-robin index modulo total.
func (s *LoadBalanceState) CurrentIndex(total int) int {
	if total <= 0 {
		return 0
	}
	return s.RoundRobinIndex % total
}
