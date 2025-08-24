package scheduler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"runtime"

	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/utils/loExt"
	"github.com/jobs/scheduler/pkg/config"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type HealthChecker struct {
	logger       *zap.Logger
	config       config.HealthCheckConfig
	httpClient   *http.Client
	stopCh       chan struct{}
	wg           sync.WaitGroup
	taskRunner   ITaskRunner // 添加TaskRunner引用
	executorRepo executor.Repo

	// recovery tracking: consecutive success counts per executor
	successCounts map[uint64]int
	scMu          sync.Mutex
}

func NewHealthChecker(logger *zap.Logger, config config.HealthCheckConfig, taskRunner ITaskRunner, executorRepo executor.Repo) *HealthChecker {
    hc := &HealthChecker{
        logger: logger,
        config: config,
        stopCh:        make(chan struct{}),
        taskRunner:    taskRunner,
        executorRepo:  executorRepo,
        successCounts: make(map[uint64]int),
    }
    // Use a short per-request timeout for health checks (cap at 5s, default 3s)
    t := hc.requestTimeout()
    hc.httpClient = &http.Client{Timeout: t}
    return hc
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
	executors, err := h.executorRepo.FindByStatus(context.Background(),
		loExt.DefSlice(
			executor.ExecutorStatusOnline, executor.ExecutorStatusOffline,
		))
	if err != nil {
		h.logger.Error("failed to get executors for health check", zap.Error(err))
		return
	}

	if len(executors) == 0 {
		return
	}

	// 并发检查所有执行器（带并发上限）
	var g errgroup.Group
	g.SetLimit(h.maxConcurrentChecks())
	for _, exec := range executors {
		exec := exec
		g.Go(func() error {
			h.checkExecutor(exec)
			return nil
		})
	}
	_ = g.Wait()
}

// maxConcurrentChecks returns a reasonable concurrency cap for health checks.
func (h *HealthChecker) maxConcurrentChecks() int {
	// Up to 4x CPUs, but not more than 32 at once; at least 1.
	n := runtime.NumCPU() * 4
	if n > 32 {
		n = 32
	}
	if n < 1 {
		n = 1
	}
	return n
}

func (h *HealthChecker) checkExecutor(exe *executor.Executor) {
    // Always use a short timeout for health checks
    ctx, cancel := context.WithTimeout(context.Background(), h.requestTimeout())
    defer cancel()

	isHealthy := h.ping(ctx, exe)

	// reset entity patch aggregation for this round
	exe.ClearPatch().UpdateLastHealthCheck(time.Now())

	if isHealthy {
		_, recoveredOnline, didRecover := exe.TryRecoverAfterSuccess(h.incSuccess(exe.ID), h.config.RecoveryThreshold)
		if didRecover {
			// 恢复后，清空计数
			h.resetSuccess(exe.ID)
		}
		if recoveredOnline && h.taskRunner != nil {
			h.taskRunner.ResetBreaker(exe.ID)
		}
	} else {
		// 失败：清空连续成功计数
		h.resetSuccess(exe.ID)
		_, _, becameOffline := exe.OnHealthCheckFailure(h.config.FailureThreshold)
		// 清理熔断器，避免错误计数累积
		if becameOffline && h.taskRunner != nil {
			h.taskRunner.RemoveBreaker(exe.ID)
		}
	}

	h.update(exe.ID, exe.ExportPatch())
}

// requestTimeout returns a short timeout for health checks.
// Caps at 5s, defaults to 3s, and enforces a minimum of 1s.
func (h *HealthChecker) requestTimeout() time.Duration {
    const (
        def = 3 * time.Second
        capT = 5 * time.Second
        minT = 1 * time.Second
    )
    t := h.config.Timeout
    if t <= 0 {
        t = def
    }
    if t > capT {
        t = capT
    }
    if t < minT {
        t = minT
    }
    return t
}

// incSuccess increments and returns the consecutive success count for an executor.
func (h *HealthChecker) incSuccess(executorID uint64) int {
	h.scMu.Lock()
	defer h.scMu.Unlock()
	h.successCounts[executorID] = h.successCounts[executorID] + 1
	return h.successCounts[executorID]
}

// resetSuccess clears the consecutive success count for an executor.
func (h *HealthChecker) resetSuccess(executorID uint64) {
	h.scMu.Lock()
	defer h.scMu.Unlock()
	delete(h.successCounts, executorID)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		h.logger.Debug("health check returned non-2xx status", zap.Uint64("executor_id", executor.ID), zap.Int("status_code", resp.StatusCode))
		return false
	}

	return true
}

func (h *HealthChecker) update(id uint64, patch *executor.ExecutorPatch) {
	if patch == nil {
		return
	}
	// Queue full: fallback to direct update
	if err := h.executorRepo.Update(context.Background(), id, patch); err != nil {
		h.logger.Error("failed direct update when queue full", zap.Uint64("executor_id", id), zap.Error(err))
	}
}
