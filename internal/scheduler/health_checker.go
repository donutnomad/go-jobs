package scheduler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/infra/persistence/executorrepo"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/pkg/config"
	"go.uber.org/zap"
)

type ITaskRunner interface {
	RemoveBreaker(executorID uint64)
	ResetBreaker(executorID uint64)
}

type HealthChecker struct {
	storage      *orm.Storage
	logger       *zap.Logger
	config       config.HealthCheckConfig
	httpClient   *http.Client
	stopCh       chan struct{}
	wg           sync.WaitGroup
	taskRunner   ITaskRunner // 添加TaskRunner引用
	executorRepo executor.Repo
}

func NewHealthChecker(storage *orm.Storage, logger *zap.Logger, config config.HealthCheckConfig,
	taskRunner ITaskRunner, executorRepo executor.Repo) *HealthChecker {
	return &HealthChecker{
		storage: storage,
		logger:  logger,
		config:  config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stopCh:       make(chan struct{}),
		taskRunner:   taskRunner,
		executorRepo: executorRepo,
	}
}

func (h *HealthChecker) Start() {
	if !h.config.Enabled {
		h.logger.Info("health checker is disabled")
		return
	}
	h.wg.Add(1)
	go h.run()
	h.logger.Info("health checker started",
		zap.Duration("interval", h.config.Interval))
}

func (h *HealthChecker) Stop() {
	close(h.stopCh)
	h.wg.Wait()
	h.logger.Info("health checker stopped")
}

func (h *HealthChecker) run() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	h.checkAll()

	for {
		select {
		case <-ticker.C:
			h.checkAll()
		case <-h.stopCh:
			return
		}
	}
}

func (h *HealthChecker) checkAll() {
	executors, err := h.executorRepo.FindByStatus(context.Background(), []executor.ExecutorStatus{
		executor.ExecutorStatusOnline,
		executor.ExecutorStatusOffline,
	})
	if err != nil {
		h.logger.Error("failed to get executors for health check", zap.Error(err))
		return
	}

	// 并发检查所有执行器
	var wg sync.WaitGroup
	for _, exec := range executors {
		wg.Add(1)
		go func(exec *executor.Executor) {
			defer wg.Done()
			h.checkExecutor(exec)
		}(exec)
	}
	wg.Wait()
}

func (h *HealthChecker) checkExecutor(exe *executor.Executor) {
	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout)
	defer cancel()

	isHealthy := h.ping(ctx, exe)
	now := time.Now()
	exe.LastHealthCheck = &now

	// 更新健康状态
	if isHealthy {
		// 健康检查成功 - 立即恢复
		exe.HealthCheckFailures = 0

		if !exe.IsHealthy {
			// 从不健康恢复
			exe.IsHealthy = true
			h.logger.Info("executor recovered to healthy",
				zap.Uint64("executor_id", exe.ID),
				zap.String("instance_id", exe.InstanceID))
		}

		// 如果之前是离线状态，立即恢复为在线
		if exe.Status == executor.ExecutorStatusOffline {
			exe.Status = executor.ExecutorStatusOnline
			h.logger.Info("executor recovered to online",
				zap.Uint64("executor_id", exe.ID),
				zap.String("instance_id", exe.InstanceID))

			// 重置熔断器状态
			if h.taskRunner != nil {
				h.taskRunner.ResetBreaker(exe.ID)
			}
		}
	} else {
		// 健康检查失败

		// 如果执行器已经离线，不要继续累加失败次数
		if exe.Status == executor.ExecutorStatusOffline {
			// 已经离线，保持当前状态，不累加错误计数
			h.logger.Debug("executor is already offline, skip failure count increment",
				zap.Uint64("executor_id", exe.ID),
				zap.String("instance_id", exe.InstanceID),
				zap.Int("current_failures", exe.HealthCheckFailures))
		} else {
			// 执行器还未离线，累加失败次数
			exe.HealthCheckFailures++

			// 检查是否达到失败阈值
			if exe.HealthCheckFailures >= h.config.FailureThreshold {
				// 标记为不健康
				if exe.IsHealthy {
					exe.IsHealthy = false
					h.logger.Warn("executor marked as unhealthy",
						zap.Uint64("executor_id", exe.ID),
						zap.String("instance_id", exe.InstanceID),
						zap.Int("failures", exe.HealthCheckFailures))
				}

				// 标记为离线
				exe.Status = executor.ExecutorStatusOffline
				h.logger.Warn("executor marked as offline due to health check failures",
					zap.Uint64("executor_id", exe.ID),
					zap.String("instance_id", exe.InstanceID),
					zap.Int("failures", exe.HealthCheckFailures))

				// 清理熔断器，避免错误计数累积
				if h.taskRunner != nil {
					h.taskRunner.RemoveBreaker(exe.ID)
				}
			}
		}
	}

	// 保存更新
	if err := executorrepo.NewMysqlRepositoryImpl(h.storage.DB()).Save(ctx, exe); err != nil {
		h.logger.Error("failed to update executor health status",
			zap.Uint64("executor_id", exe.ID),
			zap.Error(err))
	}
}

func (h *HealthChecker) ping(ctx context.Context, executor *executor.Executor) bool {
	u := executor.GetHealthCheckURL()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		h.logger.Error("failed to create health check request", zap.Uint64("executor_id", executor.ID), zap.Error(err))
		return false
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Debug("health check failed", zap.Uint64("executor_id", executor.ID), zap.String("url", u), zap.Error(err))
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logger.Debug("health check returned non-200 status", zap.Uint64("executor_id", executor.ID), zap.Int("status_code", resp.StatusCode))
		return false
	}

	return true
}
