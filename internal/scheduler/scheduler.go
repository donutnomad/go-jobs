package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/biz/scheduler_instance"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"github.com/jobs/scheduler/internal/loadbalance"
	"github.com/jobs/scheduler/pkg/config"
	"github.com/robfig/cron/v3"
	"github.com/yitter/idgenerator-go/idgen"
	"go.uber.org/zap"
)

// Scheduler 任务调度器
type Scheduler struct {
	config        config.SchedulerConfig
	sqlDB         *sql.DB
	locker        *Locker
	cron          *cron.Cron
	lbManager     *loadbalance.Manager
	healthChecker *HealthChecker
	logger        *zap.Logger

	instanceID string
	isLeader   bool
	leaderMu   sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup

	// 任务执行器
	taskRunner *TaskRunner

	// repositories
	taskRepo              task.Repo
	executionRepo         execution.Repo
	schedulerInstanceRepo scheduler_instance.Repo

	// redis pub/sub listener
	rdb *redis.Client
	sub *redis.PubSub
}

// New 创建调度器
func New(
	cfg config.Config,
	db commonrepo.DB,
	logger *zap.Logger,

	taskRunner *TaskRunner,
	lbManager *loadbalance.Manager,
	checker *HealthChecker,

	taskRepo task.Repo,
	executionRepo execution.Repo,
	schedulerInstanceRepo scheduler_instance.Repo,
	rdb *redis.Client,
) (*Scheduler, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	s := &Scheduler{
		config:                cfg.Scheduler,
		sqlDB:                 sqlDB,
		logger:                logger,
		instanceID:            cfg.Scheduler.InstanceID,
		isLeader:              false,
		stopCh:                make(chan struct{}),
		lbManager:             lbManager,
		cron:                  cron.New(cron.WithSeconds()),
		taskRepo:              taskRepo,
		executionRepo:         executionRepo,
		schedulerInstanceRepo: schedulerInstanceRepo,
		taskRunner:            taskRunner,
		healthChecker:         checker,
	}

	// 创建分布式锁
	s.locker = NewLocker(sqlDB, cfg.Scheduler.LockKey, cfg.Scheduler.LockTimeout, logger)

	// 注册调度器实例
	if err := s.registerInstance(); err != nil {
		return nil, fmt.Errorf("failed to register scheduler instance: %w", err)
	}

	// inject redis client
	s.rdb = rdb

	return s, nil
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.logger.Info("starting scheduler",
		zap.String("instance_id", s.instanceID))

	// 启动健康检查
	s.healthChecker.Start()

	// 启动任务执行器
	s.taskRunner.Start()

	// 启动心跳goroutine
	s.wg.Add(1)
	go s.heartbeatLoop()

	// 启动领导者选举
	s.wg.Add(1)
	go s.leaderElection()

	// 订阅事件通道（即使不是leader，也要监听；只有leader处理）
	s.startEventSubscriber()

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	s.logger.Info("stopping scheduler",
		zap.String("instance_id", s.instanceID))

	close(s.stopCh)

	// 停止cron
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
	}

	// 释放锁
	if s.locker.IsLocked() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.locker.Unlock(ctx); err != nil {
			s.logger.Error("failed to release lock", zap.Error(err))
		}
	}

	// 停止健康检查
	s.healthChecker.Stop()

	// 停止任务执行器
	s.taskRunner.Stop()

	// 等待所有goroutine退出
	s.wg.Wait()

	// 更新实例状态
	s.updateInstanceStatus(false)

	// 关闭redis订阅
	if s.sub != nil {
		_ = s.sub.Close()
	}
	if s.rdb != nil {
		_ = s.rdb.Close()
	}

	s.logger.Info("scheduler stopped",
		zap.String("instance_id", s.instanceID))

	return nil
}

// startEventSubscriber subscribes to Redis events and processes them only when leader.
func (s *Scheduler) startEventSubscriber() {
    if s.rdb == nil {
        s.logger.Warn("redis client nil, event subscriber not started")
        return
    }
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        backoff := 1 * time.Second
        for {
            // establish subscription
            sub := s.rdb.Subscribe(context.Background(), redisChannel)
            s.sub = sub
            ch := sub.Channel()

            // consume loop
            for {
                select {
                case msg, ok := <-ch:
                    if !ok {
                        // channel closed (network or server side) -> reconnect
                        goto reconnect
                    }
                    var ev RedisEvent
                    if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
                        s.logger.Error("failed to unmarshal event", zap.Error(err))
                        continue
                    }
                    if !s.IsLeader() {
                        // Only leader processes events
                        continue
                    }
                    s.handleEvent(ev)
                case <-s.stopCh:
                    _ = sub.Close()
                    return
                }
            }

        reconnect:
            _ = sub.Close()
            // wait then retry unless stopping
            select {
            case <-time.After(backoff):
                if backoff < 30*time.Second {
                    backoff *= 2
                }
            case <-s.stopCh:
                return
            }
        }
    }()
}

func (s *Scheduler) handleEvent(ev RedisEvent) {
    if !s.IsLeader() {
        return
    }
    switch ev.Type {
    case EventSubmitNewTask:
        if err := s.ScheduleNow(ev.TaskID, ev.Parameters); err != nil {
            s.logger.Error("failed to schedule now", zap.Error(err))
        }
	case EventReloadTasks:
		if err := s.ReloadTasks(); err != nil {
			s.logger.Error("failed to reload tasks", zap.Error(err))
		}
	case EventCancelExecutionTimer:
		s.CancelExecutionTimeout(ev.ExecutionID)
	default:
		// ignore unknown
	}
}

func (s *Scheduler) GetTaskRunner() *TaskRunner {
	return s.taskRunner
}

// registerInstance 注册调度器实例
func (s *Scheduler) registerInstance() error {
	ctx := context.Background()
	instance := &scheduler_instance.SchedulerInstance{
		ID:         uint64(idgen.NextId()),
		InstanceID: s.instanceID,
		Host:       "localhost", // TODO: 获取真实主机名
		Port:       s.config.MaxWorkers,
		IsLeader:   false,
	}

	// 检查实例是否已存在
	existing, err := s.schedulerInstanceRepo.GetByInstanceID(ctx, s.instanceID)
	if err != nil {
		return fmt.Errorf("failed to query scheduler instance: %w", err)
	}

	if existing == nil {
		// 创建新实例
		if err := s.schedulerInstanceRepo.Create(ctx, instance); err != nil {
			return fmt.Errorf("failed to create scheduler instance: %w", err)
		}
		s.logger.Info("scheduler instance registered", zap.String("instance_id", s.instanceID))
	} else {
		// 更新现有实例
		existing.IsLeader = false
		if err := s.schedulerInstanceRepo.Save(ctx, existing); err != nil {
			return fmt.Errorf("failed to update scheduler instance: %w", err)
		}
		s.logger.Info("scheduler instance updated", zap.String("instance_id", s.instanceID))
	}

	return nil
}

// heartbeatLoop 定期更新实例心跳
func (s *Scheduler) heartbeatLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.HeartbeatInterval / 2) // 心跳频率为选举间隔的一半
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if err := s.schedulerInstanceRepo.UpdateHeartbeat(ctx, s.instanceID); err != nil {
				s.logger.Error("failed to update heartbeat",
					zap.String("instance_id", s.instanceID),
					zap.Error(err))
			}
		case <-s.stopCh:
			return
		}
	}
}

// leaderElection 领导者选举
func (s *Scheduler) leaderElection() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tryBecomeLeader()
		case <-s.stopCh:
			return
		}
	}
}

// tryBecomeLeader 尝试成为领导者
func (s *Scheduler) tryBecomeLeader() {
	// 检查是否正在关闭
	select {
	case <-s.stopCh:
		return
	default:
	}

	// 创建可以被stopCh取消的context
	ctx, cancel := context.WithTimeout(context.Background(), s.config.LockTimeout)
	defer cancel()

	// 启动goroutine监听stopCh，如果关闭则取消context
	go func() {
		select {
		case <-s.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	if !s.IsLeader() {
		s.logger.Debug("attempting to acquire leader lock",
			zap.String("instance_id", s.instanceID))

		// 尝试获取锁
		locked, err := s.locker.TryLock(ctx)
		if err != nil {
			// 如果是context取消（服务正在关闭），不记录为错误
			if ctx.Err() == context.Canceled {
				s.logger.Debug("leader lock acquisition cancelled due to shutdown",
					zap.String("instance_id", s.instanceID))
				return
			}
			s.logger.Error("failed to acquire leader lock",
				zap.String("instance_id", s.instanceID),
				zap.Error(err))
			return
		}

		if locked {
			s.setLeader(true)
			s.updateInstanceStatus(true)
			s.logger.Info("became leader",
				zap.String("instance_id", s.instanceID))

			// 恢复遗留的执行任务（服务重启后的清理工作）
			if err := s.recoverOrphanedExecutions(); err != nil {
				s.logger.Error("failed to recover orphaned executions", zap.Error(err))
			}

			// 加载并调度任务
			if err := s.loadAndScheduleTasks(); err != nil {
				s.logger.Error("failed to load and schedule tasks", zap.Error(err))
			}

			// 启动cron调度器
			s.cron.Start()
			s.logger.Info("started cron scheduler as leader",
				zap.String("instance_id", s.instanceID))
		} else {
			s.logger.Debug("could not acquire leader lock, will retry",
				zap.String("instance_id", s.instanceID))
		}
    } else {
		s.logger.Debug("renewing leader lock",
			zap.String("instance_id", s.instanceID))

        // 续约锁
        if err := s.locker.Renew(ctx); err != nil {
			// 如果是context取消（服务正在关闭），不记录为错误
			if ctx.Err() == context.Canceled {
				s.logger.Debug("leader lock renewal cancelled due to shutdown",
					zap.String("instance_id", s.instanceID))
				return
			}
            s.logger.Error("failed to renew leader lock, stepping down",
				zap.String("instance_id", s.instanceID),
				zap.Error(err))
            s.setLeader(false)
            s.updateInstanceStatus(false)

            // 停止cron调度器
            stopCtx := s.cron.Stop()
            <-stopCtx.Done()
			s.logger.Info("stopped cron scheduler, no longer leader",
				zap.String("instance_id", s.instanceID))
        } else {
			s.logger.Debug("successfully renewed leader lock",
				zap.String("instance_id", s.instanceID))
		}
    }
}

// updateInstanceStatus 更新实例状态
func (s *Scheduler) updateInstanceStatus(isLeader bool) {
	ctx := context.Background()
	err := s.schedulerInstanceRepo.UpdateLeaderStatus(ctx, s.instanceID, isLeader)
	if err != nil {
		s.logger.Error("failed to update instance status",
			zap.Error(err))
	}
}

// IsLeader safely reports current leadership state.
func (s *Scheduler) IsLeader() bool {
	s.leaderMu.RLock()
	defer s.leaderMu.RUnlock()
	return s.isLeader
}

// setLeader safely updates leadership state.
func (s *Scheduler) setLeader(v bool) {
	s.leaderMu.Lock()
	s.isLeader = v
	s.leaderMu.Unlock()
}

// loadAndScheduleTasks 加载并调度任务
func (s *Scheduler) loadAndScheduleTasks() error {
	ctx := context.Background()

	// 清除所有现有的cron任务
	entries := s.cron.Entries()
	for _, entry := range entries {
		s.cron.Remove(entry.ID)
	}

	// 加载所有活跃任务
	tasks, err := s.taskRepo.FindActiveTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// 为每个任务添加cron调度
    for _, t := range tasks {
        // capture loop variable for closure safety
        tt := t
        entryID, err := s.cron.AddFunc(tt.CronExpression, func() {
            s.scheduleTask(tt)
        })

		if err != nil {
			s.logger.Error("failed to add cron job",
				zap.Uint64("task_id", t.ID),
				zap.String("task_name", t.Name),
				zap.Error(err))
			continue
		}

		s.logger.Info("scheduled task",
			zap.Uint64("task_id", t.ID),
			zap.String("task_name", t.Name),
			zap.String("cron", t.CronExpression),
			zap.Int("entry_id", int(entryID)))
	}

	s.logger.Info("loaded and scheduled tasks",
		zap.Int("count", len(tasks)))

	return nil
}

// ReloadTasks 重新加载和调度任务，用于暂停/恢复功能
func (s *Scheduler) ReloadTasks() error {
    if !s.IsLeader() {
        s.logger.Debug("ignore reload tasks: not leader")
        return ErrNotLeader
    }
    return s.loadAndScheduleTasks()
}

// recoverOrphanedExecutions 恢复遗留的执行任务（服务重启后的清理工作）
func (s *Scheduler) recoverOrphanedExecutions() error {
	ctx := context.Background()

	// 查询所有 pending 和 running 状态的执行记录
	orphanedExecutions, err := s.executionRepo.FindByStatuses(ctx, []execution.ExecutionStatus{
		execution.ExecutionStatusPending,
		execution.ExecutionStatusRunning,
	})
	if err != nil {
		return fmt.Errorf("failed to find orphaned executions: %w", err)
	}

	if len(orphanedExecutions) == 0 {
		s.logger.Info("no orphaned executions found")
		return nil
	}

	s.logger.Info("recovering orphaned executions",
		zap.Int("count", len(orphanedExecutions)))

	// 将这些任务标记为失败
	failedCount := 0
	for _, exec := range orphanedExecutions {
		exec.MarkFailed("Service restarted, execution was interrupted")
		if err := s.executionRepo.Save(ctx, exec); err != nil {
			s.logger.Error("failed to mark orphaned execution as failed",
				zap.Uint64("execution_id", exec.ID),
				zap.Error(err))
			continue
		}
		failedCount++

		// 取消超时定时器（如果存在）
		s.taskRunner.CancelTimeout(exec.ID)
	}

	s.logger.Info("orphaned executions recovered",
		zap.Int("total", len(orphanedExecutions)),
		zap.Int("failed", failedCount))

	return nil
}

func (s *Scheduler) CancelExecutionTimeout(executionID uint64) {
	s.taskRunner.CancelTimeout(executionID)
}

func (s *Scheduler) ScheduleNow(taskID uint64, parameters map[string]any) error {
    if !s.IsLeader() {
        s.logger.Debug("ignore schedule now: not leader",
            zap.Uint64("task_id", taskID))
        return ErrNotLeader
    }

	ctx := context.Background()

	// 加载任务信息
	task_, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}
	if task_ == nil {
		return fmt.Errorf("task not found: %d", taskID)
	}

	// 检查执行模式
	shouldExecute, err := s.checkExecutionMode(ctx, task_)
	if err != nil {
		s.logger.Error("failed to check execution mode",
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		return fmt.Errorf("failed to check execution mode: %w", err)
	}

	if !shouldExecute {
		s.logger.Info("skipping task execution due to execution mode",
			zap.Uint64("task_id", taskID),
			zap.String("execution_mode", string(task_.ExecutionMode)))
		return nil
	}

    execution_ := execution.TaskExecution{
        ID:            uint64(idgen.NextId()),
        TaskID:        taskID,
        ScheduledTime: time.Now(),
        Status:        execution.ExecutionStatusPending,
	}
	err = s.executionRepo.Create(ctx, &execution_)
	if err != nil {
		return err
	}
	s.taskRunner.Submit(taskID, parameters, execution_.ID)
	return nil
}

// scheduleTask 调度任务执行
func (s *Scheduler) scheduleTask(task *task.Task) {
	ctx := context.Background()

	s.logger.Info("scheduling task",
		zap.Uint64("task_id", task.ID),
		zap.String("task_name", task.Name))

	// 检查执行模式
	shouldExecute, err := s.checkExecutionMode(ctx, task)
	if err != nil {
		s.logger.Error("failed to check execution mode",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
		return
	}

	if !shouldExecute {
		s.logger.Info("skipping task execution",
			zap.Uint64("task_id", task.ID),
			zap.String("reason", "execution mode check"))
		return
	}

	// 创建执行记录
	exec := &execution.TaskExecution{
		ID:            uint64(idgen.NextId()),
		TaskID:        task.ID,
		ScheduledTime: time.Now(),
		Status:        execution.ExecutionStatusPending,
	}

	if err := s.executionRepo.Create(ctx, exec); err != nil {
		s.logger.Error("failed to create execution record",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
		return
	}

	// 提交到任务执行器
	s.taskRunner.submit(task, exec)
}

// checkExecutionMode 检查执行模式
func (s *Scheduler) checkExecutionMode(ctx context.Context, task_ *task.Task) (bool, error) {
	switch task_.ExecutionMode {
	case task.ExecutionModeParallel:
		// 并行模式，总是执行
		return true, nil

	case task.ExecutionModeSequential:
		// 串行模式，检查是否有正在运行的任务
		count, err := s.executionRepo.CountByTaskAndStatus(ctx, task_.ID, []execution.ExecutionStatus{
			execution.ExecutionStatusPending,
			execution.ExecutionStatusRunning,
		})
		if err != nil {
			return false, err
		}
		return count == 0, nil

	case task.ExecutionModeSkip:
		// 跳过模式，如果有正在运行的任务则跳过
		count, err := s.executionRepo.CountByTaskAndStatus(ctx, task_.ID, []execution.ExecutionStatus{
			execution.ExecutionStatusPending,
			execution.ExecutionStatusRunning,
		})
		if err != nil {
			return false, err
		}

		if count > 0 {
			// 有任务正在执行，返回false表示跳过，不创建任何记录
			return false, nil
		}
		return true, nil

	default:
		return true, nil
	}
}
