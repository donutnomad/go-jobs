package scheduler

import (
    "context"
    "encoding/json"
    "time"

    redis "github.com/go-redis/redis/v8"
)

// EventBus now publishes events via Redis pub/sub.
// It keeps a fallback to direct calls if Redis is disabled/not reachable.
var _ IEmitter = (*EventBus)(nil)

type EventBus struct {
    scheduler *Scheduler
    rdb       *redis.Client
}

// NewEventBus constructs an event bus with an injected Redis client.
// If rdb is nil, it will fallback to direct in-process calls.
func NewEventBus(scheduler *Scheduler, rdb *redis.Client) *EventBus {
    return &EventBus{scheduler: scheduler, rdb: rdb}
}

func (e *EventBus) publish(ctx context.Context, ev RedisEvent) error {
    if e.rdb == nil { // fallback when redis disabled
        switch ev.Type {
        case EventSubmitNewTask:
            return e.scheduler.ScheduleNow(ev.TaskID, ev.Parameters)
        case EventReloadTasks:
            return e.scheduler.ReloadTasks()
        case EventCancelExecutionTimer:
            e.scheduler.CancelExecutionTimeout(ev.ExecutionID)
            return nil
        default:
            return nil
        }
    }

    payload, err := json.Marshal(ev)
    if err != nil {
        return err
    }
    return e.rdb.Publish(ctx, redisChannel, payload).Err()
}

func (e *EventBus) SubmitNewTask(taskID uint64, parameters map[string]any) error {
    ev := RedisEvent{
        Type:       EventSubmitNewTask,
        TaskID:     taskID,
        Parameters: parameters,
        Source:     e.scheduler.instanceID,
        Timestamp:  time.Now().UnixMilli(),
    }
    return e.publish(context.Background(), ev)
}

func (e *EventBus) ReloadTasks() error {
    ev := RedisEvent{
        Type:      EventReloadTasks,
        Source:    e.scheduler.instanceID,
        Timestamp: time.Now().UnixMilli(),
    }
    return e.publish(context.Background(), ev)
}

func (e *EventBus) CancelExecutionTimer(executionID uint64) error {
    ev := RedisEvent{
        Type:        EventCancelExecutionTimer,
        ExecutionID: executionID,
        Source:      e.scheduler.instanceID,
        Timestamp:   time.Now().UnixMilli(),
    }
    return e.publish(context.Background(), ev)
}
