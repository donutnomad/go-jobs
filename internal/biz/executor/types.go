package executor

type ExecutorStatus string

const (
	ExecutorStatusOnline      ExecutorStatus = "online"
	ExecutorStatusOffline     ExecutorStatus = "offline"
	ExecutorStatusMaintenance ExecutorStatus = "maintenance"
)

func (s ExecutorStatus) ToInt() int {
	switch s {
	case ExecutorStatusOnline:
		return 1
	case ExecutorStatusOffline:
		return 3
	case ExecutorStatusMaintenance:
		return 2
	}
	return 0
}
