package executor

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/storage"
	"github.com/jobs/scheduler/pkg/config"
	"go.uber.org/zap"
)

// TaskRunnerInterface 定义TaskRunner的接口，避免循环引用
type TaskRunnerInterface interface {
	RemoveBreaker(executorID string)
	ResetBreaker(executorID string)
}

type HealthChecker struct {
	storage    *storage.Storage
	logger     *zap.Logger
	config     config.HealthCheckConfig
	httpClient *http.Client
	stopCh     chan struct{}
	wg         sync.WaitGroup
	taskRunner TaskRunnerInterface // 添加TaskRunner引用
}

func NewHealthChecker(storage *storage.Storage, logger *zap.Logger, config config.HealthCheckConfig) *HealthChecker {
	return &HealthChecker{
		storage: storage,
		logger:  logger,
		config:  config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stopCh: make(chan struct{}),
	}
}

// SetTaskRunner 设置TaskRunner引用
func (h *HealthChecker) SetTaskRunner(taskRunner TaskRunnerInterface) {
	h.taskRunner = taskRunner
}

// Start 启动健康检查
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

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	close(h.stopCh)
	h.wg.Wait()
	h.logger.Info("health checker stopped")
}

func (h *HealthChecker) run() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	// 立即执行一次健康检查
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
	// 获取所有需要检查的执行器（包括离线的，以便能够恢复）
	var executors []models.Executor
	err := h.storage.DB().
		Where("status IN ?", []models.ExecutorStatus{
			models.ExecutorStatusOnline,
			models.ExecutorStatusOffline,
			// 不检查 maintenance 状态的执行器，因为这是人为设置的维护状态
		}).
		Find(&executors).Error
	if err != nil {
		h.logger.Error("failed to get executors for health check", zap.Error(err))
		return
	}

	// 并发检查所有执行器
	var wg sync.WaitGroup
	for _, executor := range executors {
		wg.Add(1)
		go func(exec models.Executor) {
			defer wg.Done()
			h.checkExecutor(&exec)
		}(executor)
	}
	wg.Wait()
}

func (h *HealthChecker) checkExecutor(executor *models.Executor) {
	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout)
	defer cancel()

	isHealthy := h.ping(ctx, executor)
	now := time.Now()
	executor.LastHealthCheck = &now

	// 更新健康状态
	if isHealthy {
		// 健康检查成功 - 立即恢复
		executor.HealthCheckFailures = 0

		if !executor.IsHealthy {
			// 从不健康恢复
			executor.IsHealthy = true
			h.logger.Info("executor recovered to healthy",
				zap.String("executor_id", executor.ID),
				zap.String("instance_id", executor.InstanceID))
		}

		// 如果之前是离线状态，立即恢复为在线
		if executor.Status == models.ExecutorStatusOffline {
			executor.Status = models.ExecutorStatusOnline
			h.logger.Info("executor recovered to online",
				zap.String("executor_id", executor.ID),
				zap.String("instance_id", executor.InstanceID))

			// 重置熔断器状态
			if h.taskRunner != nil {
				h.taskRunner.ResetBreaker(executor.ID)
			}
		}
	} else {
		// 健康检查失败

		// 如果执行器已经离线，不要继续累加失败次数
		if executor.Status == models.ExecutorStatusOffline {
			// 已经离线，保持当前状态，不累加错误计数
			h.logger.Debug("executor is already offline, skip failure count increment",
				zap.String("executor_id", executor.ID),
				zap.String("instance_id", executor.InstanceID),
				zap.Int("current_failures", executor.HealthCheckFailures))
		} else {
			// 执行器还未离线，累加失败次数
			executor.HealthCheckFailures++

			// 检查是否达到失败阈值
			if executor.HealthCheckFailures >= h.config.FailureThreshold {
				// 标记为不健康
				if executor.IsHealthy {
					executor.IsHealthy = false
					h.logger.Warn("executor marked as unhealthy",
						zap.String("executor_id", executor.ID),
						zap.String("instance_id", executor.InstanceID),
						zap.Int("failures", executor.HealthCheckFailures))
				}

				// 标记为离线
				executor.Status = models.ExecutorStatusOffline
				h.logger.Warn("executor marked as offline due to health check failures",
					zap.String("executor_id", executor.ID),
					zap.String("instance_id", executor.InstanceID),
					zap.Int("failures", executor.HealthCheckFailures))

				// 清理熔断器，避免错误计数累积
				if h.taskRunner != nil {
					h.taskRunner.RemoveBreaker(executor.ID)
				}
			}
		}
	}

	// 移除基于心跳的离线判断，完全依赖健康检查结果
	// 这样确保离线的执行器能够通过健康检查恢复

	// 保存更新
	if err := h.storage.DB().Save(executor).Error; err != nil {
		h.logger.Error("failed to update executor health status",
			zap.String("executor_id", executor.ID),
			zap.Error(err))
	}
}

func (h *HealthChecker) ping(ctx context.Context, executor *models.Executor) bool {
	if executor.HealthCheckURL == "" {
		// 如果没有健康检查URL，使用基础URL
		executor.HealthCheckURL = fmt.Sprintf("%s/health", executor.BaseURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, executor.HealthCheckURL, nil)
	if err != nil {
		h.logger.Error("failed to create health check request",
			zap.String("executor_id", executor.ID),
			zap.Error(err))
		return false
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Debug("health check failed",
			zap.String("executor_id", executor.ID),
			zap.String("url", executor.HealthCheckURL),
			zap.Error(err))
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logger.Debug("health check returned non-200 status",
			zap.String("executor_id", executor.ID),
			zap.Int("status_code", resp.StatusCode))
		return false
	}

	return true
}
