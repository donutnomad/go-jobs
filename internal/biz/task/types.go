package task

type ExecutionMode string

const (
	ExecutionModeSequential ExecutionMode = "sequential"
	ExecutionModeParallel   ExecutionMode = "parallel"
	ExecutionModeSkip       ExecutionMode = "skip"
)

type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin         LoadBalanceStrategy = "round_robin"
	LoadBalanceWeightedRoundRobin LoadBalanceStrategy = "weighted_round_robin"
	LoadBalanceRandom             LoadBalanceStrategy = "random"
	LoadBalanceSticky             LoadBalanceStrategy = "sticky"
	LoadBalanceLeastLoaded        LoadBalanceStrategy = "least_loaded"
)

type TaskStatus string

const (
	TaskStatusActive  TaskStatus = "active"
	TaskStatusPaused  TaskStatus = "paused"
	TaskStatusDeleted TaskStatus = "deleted"
)
