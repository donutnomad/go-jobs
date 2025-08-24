package scheduler

// EventType represents the type of events flowing through the bus.
type EventType string

const (
    EventSubmitNewTask       EventType = "submit_new_task"
    EventReloadTasks         EventType = "reload_tasks"
    EventCancelExecutionTimer EventType = "cancel_execution_timer"
)

// RedisEvent is the message payload for pub/sub.
type RedisEvent struct {
    Type        EventType        `json:"type"`
    TaskID      uint64           `json:"task_id,omitempty"`
    Parameters  map[string]any   `json:"parameters,omitempty"`
    ExecutionID uint64           `json:"execution_id,omitempty"`
    Source      string           `json:"source,omitempty"`
    Timestamp   int64            `json:"ts,omitempty"`
}

const redisChannel = "go-jobs:scheduler-events"

