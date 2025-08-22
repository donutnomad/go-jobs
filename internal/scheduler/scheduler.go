package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/loadbalance"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/pkg/config"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Scheduler 任务调度器
type Scheduler struct {
	config        config.SchedulerConfig
	storage       *orm.Storage
	sqlDB         *sql.DB
	locker        *Locker
	cron          *cron.Cron
	lbManager     *loadbalance.Manager
	healthChecker *HealthChecker
	logger        *zap.Logger

	instanceID string
	isLeader   bool
	stopCh     chan struct{}
	wg         sync.WaitGroup

	// 任务执行器
	taskRunner *TaskRunner
}

// New 创建调度器
func New(cfg config.Config, storage *orm.Storage, logger *zap.Logger, callbackURL func(string) string) (*Scheduler, error) {
	sqlDB, err := storage.DB().DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	s := &Scheduler{
		config:     cfg.Scheduler,
		storage:    storage,
		sqlDB:      sqlDB,
		logger:     logger,
		instanceID: cfg.Scheduler.InstanceID,
		isLeader:   false,
		stopCh:     make(chan struct{}),
		lbManager:  loadbalance.NewManager(storage),
		cron:       cron.New(cron.WithSeconds()),
	}

	// 创建分布式锁
	s.locker = NewLocker(sqlDB, cfg.Scheduler.LockKey, cfg.Scheduler.LockTimeout, logger)

	// 创建任务执行器
	s.taskRunner = NewTaskRunner(storage, s.lbManager, logger, cfg.Scheduler.MaxWorkers, callbackURL)

	// Executor健康检查
	s.healthChecker = NewHealthChecker(storage, logger, cfg.HealthCheck, s.taskRunner)

	// 注册调度器实例
	if err := s.registerInstance(); err != nil {
		return nil, fmt.Errorf("failed to register scheduler instance: %w", err)
	}

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

	// 启动领导者选举
	s.wg.Add(1)
	go s.leaderElection()

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

	s.logger.Info("scheduler stopped",
		zap.String("instance_id", s.instanceID))

	return nil
}

func (s *Scheduler) GetTaskRunner() *TaskRunner {
	return s.taskRunner
}

// registerInstance 注册调度器实例
func (s *Scheduler) registerInstance() error {
	instance := models.SchedulerInstance{
		ID:         uuid.New().String(),
		InstanceID: s.instanceID,
		Host:       "localhost", // TODO: 获取真实主机名
		Port:       s.config.MaxWorkers,
		IsLeader:   false,
	}

	// 检查实例是否已存在
	var existing models.SchedulerInstance
	err := s.storage.DB().Where("instance_id = ?", s.instanceID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建新实例
		if err := s.storage.DB().Create(&instance).Error; err != nil {
			return fmt.Errorf("failed to create scheduler instance: %w", err)
		}
	} else if err == nil {
		// 更新现有实例
		existing.IsLeader = false
		if err := s.storage.DB().Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update scheduler instance: %w", err)
		}
	} else {
		return fmt.Errorf("failed to query scheduler instance: %w", err)
	}

	return nil
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
	ctx, cancel := context.WithTimeout(context.Background(), s.config.LockTimeout)
	defer cancel()

	if !s.isLeader {
		// 尝试获取锁
		locked, err := s.locker.TryLock(ctx)
		if err != nil {
			s.logger.Error("failed to acquire leader lock", zap.Error(err))
			return
		}

		if locked {
			s.isLeader = true
			s.updateInstanceStatus(true)
			s.logger.Info("became leader",
				zap.String("instance_id", s.instanceID))

			// 加载并调度任务
			if err := s.loadAndScheduleTasks(); err != nil {
				s.logger.Error("failed to load and schedule tasks", zap.Error(err))
			}

			// 启动cron调度器
			s.cron.Start()
		}
	} else {
		// 续约锁
		if err := s.locker.Renew(ctx); err != nil {
			s.logger.Error("failed to renew leader lock", zap.Error(err))
			s.isLeader = false
			s.updateInstanceStatus(false)

			// 停止cron调度器
			s.cron.Stop()
		}
	}
}

// updateInstanceStatus 更新实例状态
func (s *Scheduler) updateInstanceStatus(isLeader bool) {
	result := s.storage.DB().
		Model(&models.SchedulerInstance{}).
		Where("instance_id = ?", s.instanceID).
		Update("is_leader", isLeader)

	if result.Error != nil {
		s.logger.Error("failed to update instance status",
			zap.Error(result.Error))
	}
}

// loadAndScheduleTasks 加载并调度任务
func (s *Scheduler) loadAndScheduleTasks() error {
	// 清除所有现有的cron任务
	entries := s.cron.Entries()
	for _, entry := range entries {
		s.cron.Remove(entry.ID)
	}

	// 加载所有活跃任务
	var tasks []models.Task
	err := s.storage.DB().
		Where("status = ?", models.TaskStatusActive).
		Find(&tasks).Error
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// 为每个任务添加cron调度
	for _, task := range tasks {
		t := task // 创建副本避免闭包问题
		entryID, err := s.cron.AddFunc(t.CronExpression, func() {
			s.scheduleTask(&t)
		})

		if err != nil {
			s.logger.Error("failed to add cron job",
				zap.String("task_id", t.ID),
				zap.String("task_name", t.Name),
				zap.Error(err))
			continue
		}

		s.logger.Info("scheduled task",
			zap.String("task_id", t.ID),
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
	return s.loadAndScheduleTasks()
}

// scheduleTask 调度任务执行
func (s *Scheduler) scheduleTask(task *models.Task) {
	ctx := context.Background()

	s.logger.Info("scheduling task",
		zap.String("task_id", task.ID),
		zap.String("task_name", task.Name))

	// 检查执行模式
	shouldExecute, err := s.checkExecutionMode(ctx, task)
	if err != nil {
		s.logger.Error("failed to check execution mode",
			zap.String("task_id", task.ID),
			zap.Error(err))
		return
	}

	if !shouldExecute {
		s.logger.Info("skipping task execution",
			zap.String("task_id", task.ID),
			zap.String("reason", "execution mode check"))
		return
	}

	// 创建执行记录
	execution := &models.TaskExecution{
		ID:            uuid.New().String(),
		TaskID:        task.ID,
		ScheduledTime: time.Now(),
		Status:        models.ExecutionStatusPending,
	}

	if err := s.storage.DB().Create(execution).Error; err != nil {
		s.logger.Error("failed to create execution record",
			zap.String("task_id", task.ID),
			zap.Error(err))
		return
	}

	// 提交到任务执行器
	s.taskRunner.Submit(task, execution)
}

// checkExecutionMode 检查执行模式
func (s *Scheduler) checkExecutionMode(ctx context.Context, task *models.Task) (bool, error) {
	switch task.ExecutionMode {
	case models.ExecutionModeParallel:
		// 并行模式，总是执行
		return true, nil

	case models.ExecutionModeSequential:
		// 串行模式，检查是否有正在运行的任务
		var count int64
		err := s.storage.DB().
			Model(&models.TaskExecution{}).
			Where("task_id = ? AND status IN ?", task.ID,
				[]models.ExecutionStatus{models.ExecutionStatusPending, models.ExecutionStatusRunning}).
			Count(&count).Error
		if err != nil {
			return false, err
		}
		return count == 0, nil

	case models.ExecutionModeSkip:
		// 跳过模式，如果有正在运行的任务则跳过
		var count int64
		err := s.storage.DB().
			Model(&models.TaskExecution{}).
			Where("task_id = ? AND status IN ?", task.ID,
				[]models.ExecutionStatus{models.ExecutionStatusPending, models.ExecutionStatusRunning}).
			Count(&count).Error
		if err != nil {
			return false, err
		}

		if count > 0 {
			// 创建跳过记录
			execution := &models.TaskExecution{
				ID:            uuid.New().String(),
				TaskID:        task.ID,
				ScheduledTime: time.Now(),
				Status:        models.ExecutionStatusSkipped,
				Logs:          "Skipped due to execution mode",
			}
			s.storage.DB().Create(execution)
			return false, nil
		}
		return true, nil

	default:
		return true, nil
	}
}
