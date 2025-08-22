package loadbalance

import "github.com/google/wire"

var Provider = wire.NewSet(NewManager, NewRoundRobinStrategy, NewWeightedRoundRobinStrategy, NewRandomStrategy, NewStickyStrategy, NewLeastLoadedStrategy)
