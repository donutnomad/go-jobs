package scheduler

import "errors"

// ErrNotLeader indicates the current scheduler instance is not the leader
// and therefore will not process leader-only operations.
var ErrNotLeader = errors.New("not leader")

