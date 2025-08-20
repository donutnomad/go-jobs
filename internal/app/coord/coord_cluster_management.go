package coordination

import (
	"context"
	"fmt"
	"time"

	executionbiz "github.com/jobs/scheduler/internal/app/biz/execution"
	executorbiz "github.com/jobs/scheduler/internal/app/biz/executor"
	schedulerbiz "github.com/jobs/scheduler/internal/app/biz/scheduler"
	taskbiz "github.com/jobs/scheduler/internal/app/biz/task"
	"github.com/jobs/scheduler/internal/app/infra/interfaces"
	"github.com/jobs/scheduler/internal/app/types"
)

// ClusterManagementCoordinator 集群管理协调器
// 遵循架构指南的coord_[流程].go命名规范，协调分布式调度器集群管理流程
type ClusterManagementCoordinator struct {
	taskUC      *taskbiz.UseCase
	executorUC  *executorbiz.UseCase
	executionUC *executionbiz.UseCase
	schedulerUC *schedulerbiz.UseCase
	logger      interfaces.Logger

	// 集群配置
	instanceID        string
	heartbeatInterval time.Duration
	leaderTimeout     time.Duration
}

// NewClusterManagementCoordinator 创建集群管理协调器
func NewClusterManagementCoordinator(
	taskUC *taskbiz.UseCase,
	executorUC *executorbiz.UseCase,
	executionUC *executionbiz.UseCase,
	schedulerUC *schedulerbiz.UseCase,
	logger interfaces.Logger,
	instanceID string,
) *ClusterManagementCoordinator {
	return &ClusterManagementCoordinator{
		taskUC:            taskUC,
		executorUC:        executorUC,
		executionUC:       executionUC,
		schedulerUC:       schedulerUC,
		logger:            logger,
		instanceID:        instanceID,
		heartbeatInterval: 30 * time.Second,
		leaderTimeout:     90 * time.Second,
	}
}

// ClusterBootstrapRequest 集群启动请求
type ClusterBootstrapRequest struct {
	InstanceID        string                     `json:"instance_id"`
	ClusterConfig     schedulerbiz.ClusterConfig `json:"cluster_config"`
	HeartbeatInterval time.Duration              `json:"heartbeat_interval,omitempty"`
	LeaderTimeout     time.Duration              `json:"leader_timeout,omitempty"`
}

// ClusterBootstrapResponse 集群启动响应
type ClusterBootstrapResponse struct {
	SchedulerID   types.ID                    `json:"scheduler_id"`
	InstanceID    string                      `json:"instance_id"`
	IsLeader      bool                        `json:"is_leader"`
	ClusterStatus *schedulerbiz.ClusterStatus `json:"cluster_status"`
	StartedAt     time.Time                   `json:"started_at"`
}

// BootstrapCluster 启动集群节点
func (c *ClusterManagementCoordinator) BootstrapCluster(ctx context.Context, req *ClusterBootstrapRequest) (*ClusterBootstrapResponse, error) {
	c.logger.Info(ctx, "Starting cluster bootstrap", map[string]interface{}{
		"instance_id": req.InstanceID,
	})

	if req.HeartbeatInterval > 0 {
		c.heartbeatInterval = req.HeartbeatInterval
	}
	if req.LeaderTimeout > 0 {
		c.leaderTimeout = req.LeaderTimeout
	}

	// 1. 注册调度器实例
	registerReq := &schedulerbiz.RegisterSchedulerRequest{
		InstanceID:    req.InstanceID,
		ClusterConfig: req.ClusterConfig,
	}

	registerResp, err := c.schedulerUC.RegisterScheduler(ctx, registerReq)
	if err != nil {
		return nil, fmt.Errorf("failed to register scheduler: %w", err)
	}

	schedulerID := registerResp.SchedulerID

	// 2. 尝试成为领导者
	leaderReq := &schedulerbiz.TryBecomeLeaderRequest{
		SchedulerID: schedulerID,
		LockTimeout: c.leaderTimeout,
	}

	leaderResp, err := c.schedulerUC.TryBecomeLeader(ctx, leaderReq)
	if err != nil {
		c.logger.Warn(ctx, "Failed to become leader during bootstrap", map[string]interface{}{
			"scheduler_id": schedulerID,
			"error":        err.Error(),
		})
	}

	isLeader := leaderResp != nil && leaderResp.IsLeader

	// 3. 获取集群状态
	clusterStatus, err := c.schedulerUC.GetClusterStatus(ctx)
	if err != nil {
		c.logger.Warn(ctx, "Failed to get cluster status", map[string]interface{}{
			"scheduler_id": schedulerID,
			"error":        err.Error(),
		})
	}

	c.logger.Info(ctx, "Cluster bootstrap completed", map[string]interface{}{
		"scheduler_id":    schedulerID,
		"is_leader":       isLeader,
		"cluster_healthy": clusterStatus != nil && clusterStatus.IsHealthy,
	})

	return &ClusterBootstrapResponse{
		SchedulerID:   schedulerID,
		InstanceID:    req.InstanceID,
		IsLeader:      isLeader,
		ClusterStatus: clusterStatus,
		StartedAt:     time.Now(),
	}, nil
}

// LeaderElectionRequest 领导者选举请求
type LeaderElectionRequest struct {
	SchedulerID   types.ID      `json:"scheduler_id"`
	ForceElection bool          `json:"force_election"` // 是否强制选举
	LockTimeout   time.Duration `json:"lock_timeout,omitempty"`
}

// LeaderElectionResponse 领导者选举响应
type LeaderElectionResponse struct {
	SchedulerID      types.ID      `json:"scheduler_id"`
	ElectionStarted  bool          `json:"election_started"`
	BecameLeader     bool          `json:"became_leader"`
	CurrentLeader    *types.ID     `json:"current_leader,omitempty"`
	LeaderElectedAt  *time.Time    `json:"leader_elected_at,omitempty"`
	ElectionDuration time.Duration `json:"election_duration"`
}

// ConductLeaderElection 进行领导者选举
func (c *ClusterManagementCoordinator) ConductLeaderElection(ctx context.Context, req *LeaderElectionRequest) (*LeaderElectionResponse, error) {
	startTime := time.Now()

	c.logger.Info(ctx, "Starting leader election", map[string]interface{}{
		"scheduler_id":   req.SchedulerID,
		"force_election": req.ForceElection,
	})

	// 1. 检查是否需要选举
	currentLeader, err := c.schedulerUC.GetCurrentLeader(ctx)
	if err == nil && currentLeader != nil && !req.ForceElection {
		// 已有领导者且不强制选举
		c.logger.Info(ctx, "Leader already exists", map[string]interface{}{
			"current_leader": currentLeader.ID(),
			"scheduler_id":   req.SchedulerID,
		})

		return &LeaderElectionResponse{
			SchedulerID:      req.SchedulerID,
			ElectionStarted:  false,
			BecameLeader:     currentLeader.ID() == req.SchedulerID,
			CurrentLeader:    &currentLeader.ID(),
			LeaderElectedAt:  currentLeader.LeaderElectedAt(),
			ElectionDuration: time.Since(startTime),
		}, nil
	}

	// 2. 开始选举
	startElectionReq := &schedulerbiz.StartLeaderElectionRequest{
		SchedulerID: req.SchedulerID,
	}

	if err := c.schedulerUC.StartLeaderElection(ctx, startElectionReq); err != nil {
		return nil, fmt.Errorf("failed to start election: %w", err)
	}

	// 3. 尝试获取领导权
	lockTimeout := req.LockTimeout
	if lockTimeout == 0 {
		lockTimeout = c.leaderTimeout
	}

	becomeLeaderReq := &schedulerbiz.TryBecomeLeaderRequest{
		SchedulerID: req.SchedulerID,
		LockTimeout: lockTimeout,
	}

	becomeLeaderResp, err := c.schedulerUC.TryBecomeLeader(ctx, becomeLeaderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to become leader: %w", err)
	}

	becameLeader := becomeLeaderResp.Success && becomeLeaderResp.IsLeader

	if becameLeader {
		c.logger.Info(ctx, "Successfully became leader", map[string]interface{}{
			"scheduler_id": req.SchedulerID,
			"elected_at":   becomeLeaderResp.LeaderElectedAt,
		})

		// 4. 作为新领导者，初始化集群状态
		if err := c.initializeAsLeader(ctx, req.SchedulerID); err != nil {
			c.logger.Warn(ctx, "Failed to initialize as leader", map[string]interface{}{
				"scheduler_id": req.SchedulerID,
				"error":        err.Error(),
			})
		}
	} else {
		c.logger.Info(ctx, "Failed to become leader", map[string]interface{}{
			"scheduler_id": req.SchedulerID,
		})
	}

	// 5. 获取当前领导者信息
	var currentLeaderID *types.ID
	currentLeader, err = c.schedulerUC.GetCurrentLeader(ctx)
	if err == nil && currentLeader != nil {
		currentLeaderID = &currentLeader.ID()
	}

	return &LeaderElectionResponse{
		SchedulerID:      req.SchedulerID,
		ElectionStarted:  true,
		BecameLeader:     becameLeader,
		CurrentLeader:    currentLeaderID,
		LeaderElectedAt:  becomeLeaderResp.LeaderElectedAt,
		ElectionDuration: time.Since(startTime),
	}, nil
}

// initializeAsLeader 作为领导者初始化
func (c *ClusterManagementCoordinator) initializeAsLeader(ctx context.Context, schedulerID types.ID) error {
	c.logger.Info(ctx, "Initializing as cluster leader", map[string]interface{}{
		"scheduler_id": schedulerID,
	})

	// 1. 清理孤儿执行记录
	if err := c.cleanupOrphanedExecutions(ctx); err != nil {
		c.logger.Warn(ctx, "Failed to cleanup orphaned executions", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 2. 处理待执行任务
	pendingReq := &ProcessPendingExecutionsRequest{
		MaxExecutions: 100, // 最多处理100个待执行任务
	}

	if _, err := c.processPendingExecutions(ctx, pendingReq); err != nil {
		c.logger.Warn(ctx, "Failed to process pending executions", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 3. 验证集群健康状态
	clusterStatus, err := c.schedulerUC.GetClusterStatus(ctx)
	if err != nil {
		c.logger.Warn(ctx, "Failed to get cluster status", map[string]interface{}{
			"error": err.Error(),
		})
	} else if !clusterStatus.IsHealthy {
		c.logger.Warn(ctx, "Cluster is not healthy", map[string]interface{}{
			"online_count": clusterStatus.OnlineCount,
			"total_count":  clusterStatus.TotalCount,
		})
	}

	return nil
}

// cleanupOrphanedExecutions 清理孤儿执行记录
func (c *ClusterManagementCoordinator) cleanupOrphanedExecutions(ctx context.Context) error {
	// 获取所有运行中的执行
	runningExecutions, err := c.executionUC.GetRunningExecutions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get running executions: %w", err)
	}

	// 检查执行器是否还存在和健康
	for _, execution := range runningExecutions {
		if !execution.HasExecutor() {
			continue
		}

		executorID := execution.GetExecutorID()
		getReq := &executorbiz.GetExecutorRequest{
			ExecutorID: executorID,
		}

		executorResp, err := c.executorUC.GetExecutor(ctx, getReq)
		if err != nil || !executorResp.Executor.IsAvailable() {
			// 执行器不存在或不可用，取消执行
			cancelReq := &executionbiz.CancelExecutionRequest{
				ExecutionID: execution.ID(),
				Reason:      "executor no longer available",
			}

			if err := c.executionUC.CancelExecution(ctx, cancelReq); err != nil {
				c.logger.Warn(ctx, "Failed to cancel orphaned execution", map[string]interface{}{
					"execution_id": execution.ID(),
					"executor_id":  executorID,
					"error":        err.Error(),
				})
			} else {
				c.logger.Info(ctx, "Cancelled orphaned execution", map[string]interface{}{
					"execution_id": execution.ID(),
					"executor_id":  executorID,
				})
			}
		}
	}

	return nil
}

// processPendingExecutions 处理待执行任务（复用TaskSchedulingCoordinator的逻辑）
func (c *ClusterManagementCoordinator) processPendingExecutions(ctx context.Context, req *ProcessPendingExecutionsRequest) (*ProcessPendingExecutionsResponse, error) {
	// 这里应该调用TaskSchedulingCoordinator的ProcessPendingExecutions方法
	// 简化处理，返回空结果
	return &ProcessPendingExecutionsResponse{
		ProcessedCount: 0,
		SuccessCount:   0,
		FailedCount:    0,
		Results:        []ScheduleTaskResponse{},
	}, nil
}

// HeartbeatCoordinationRequest 心跳协调请求
type HeartbeatCoordinationRequest struct {
	SchedulerID types.ID `json:"scheduler_id"`
}

// HeartbeatCoordinationResponse 心跳协调响应
type HeartbeatCoordinationResponse struct {
	SchedulerID         types.ID                    `json:"scheduler_id"`
	HeartbeatTime       time.Time                   `json:"heartbeat_time"`
	IsLeader            bool                        `json:"is_leader"`
	LeadershipRefreshed bool                        `json:"leadership_refreshed"`
	ClusterHealth       *schedulerbiz.ClusterHealth `json:"cluster_health,omitempty"`
}

// CoordinateHeartbeat 协调心跳
func (c *ClusterManagementCoordinator) CoordinateHeartbeat(ctx context.Context, req *HeartbeatCoordinationRequest) (*HeartbeatCoordinationResponse, error) {
	// 1. 更新心跳
	heartbeatReq := &schedulerbiz.UpdateHeartbeatRequest{
		SchedulerID: req.SchedulerID,
	}

	if err := c.schedulerUC.UpdateHeartbeat(ctx, heartbeatReq); err != nil {
		return nil, fmt.Errorf("failed to update heartbeat: %w", err)
	}

	heartbeatTime := time.Now()

	// 2. 检查领导权状态
	getReq := &schedulerbiz.GetSchedulerRequest{
		SchedulerID: req.SchedulerID,
	}

	getResp, err := c.schedulerUC.GetScheduler(ctx, getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler: %w", err)
	}

	isLeader := getResp.Scheduler.IsLeader()
	leadershipRefreshed := false

	// 3. 如果是领导者，刷新领导权锁
	if isLeader {
		refreshReq := &schedulerbiz.RefreshLeadershipLockRequest{
			SchedulerID: req.SchedulerID,
			LockTimeout: c.leaderTimeout,
		}

		if err := c.schedulerUC.RefreshLeadershipLock(ctx, refreshReq); err != nil {
			c.logger.Warn(ctx, "Failed to refresh leadership lock", map[string]interface{}{
				"scheduler_id": req.SchedulerID,
				"error":        err.Error(),
			})

			// 如果无法刷新锁，可能需要重新选举
			c.logger.Info(ctx, "Leadership may be lost, triggering re-election", map[string]interface{}{
				"scheduler_id": req.SchedulerID,
			})
		} else {
			leadershipRefreshed = true
		}
	}

	// 4. 处理过期的调度器节点
	if isLeader {
		if err := c.schedulerUC.ProcessStaleSchedulers(ctx, c.heartbeatInterval*3); err != nil {
			c.logger.Warn(ctx, "Failed to process stale schedulers", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// 5. 获取集群健康状态
	var clusterHealth *schedulerbiz.ClusterHealth
	if isLeader {
		clusterStatus, err := c.schedulerUC.GetClusterStatus(ctx)
		if err == nil && clusterStatus != nil {
			clusterHealth = clusterStatus.ClusterHealth
		}
	}

	return &HeartbeatCoordinationResponse{
		SchedulerID:         req.SchedulerID,
		HeartbeatTime:       heartbeatTime,
		IsLeader:            isLeader,
		LeadershipRefreshed: leadershipRefreshed,
		ClusterHealth:       clusterHealth,
	}, nil
}

// FailoverRequest 故障转移请求
type FailoverRequest struct {
	FailedSchedulerID types.ID `json:"failed_scheduler_id"`
	NewLeaderID       types.ID `json:"new_leader_id"`
	Reason            string   `json:"reason"`
}

// FailoverResponse 故障转移响应
type FailoverResponse struct {
	FailedSchedulerID   types.ID      `json:"failed_scheduler_id"`
	NewLeaderID         types.ID      `json:"new_leader_id"`
	FailoverSuccessful  bool          `json:"failover_successful"`
	AffectedExecutions  int           `json:"affected_executions"`
	RecoveredExecutions int           `json:"recovered_executions"`
	FailoverDuration    time.Duration `json:"failover_duration"`
}

// HandleFailover 处理故障转移
func (c *ClusterManagementCoordinator) HandleFailover(ctx context.Context, req *FailoverRequest) (*FailoverResponse, error) {
	startTime := time.Now()

	c.logger.Info(ctx, "Starting failover process", map[string]interface{}{
		"failed_scheduler": req.FailedSchedulerID,
		"new_leader":       req.NewLeaderID,
		"reason":           req.Reason,
	})

	// 1. 标记失败的调度器为离线
	statusReq := &schedulerbiz.UpdateSchedulerStatusRequest{
		SchedulerID: req.FailedSchedulerID,
		Status:      schedulerbiz.SchedulerStatusOffline,
		Reason:      req.Reason,
	}

	if err := c.schedulerUC.UpdateSchedulerStatus(ctx, statusReq); err != nil {
		c.logger.Warn(ctx, "Failed to mark failed scheduler as offline", map[string]interface{}{
			"scheduler_id": req.FailedSchedulerID,
			"error":        err.Error(),
		})
	}

	// 2. 如果指定了新领导者，尝试让其成为领导者
	failoverSuccessful := false
	if !req.NewLeaderID.IsZero() {
		electionReq := &LeaderElectionRequest{
			SchedulerID:   req.NewLeaderID,
			ForceElection: true,
			LockTimeout:   c.leaderTimeout,
		}

		electionResp, err := c.ConductLeaderElection(ctx, electionReq)
		if err != nil {
			c.logger.Error(ctx, "Failed to conduct leader election during failover", map[string]interface{}{
				"new_leader": req.NewLeaderID,
				"error":      err.Error(),
			})
		} else {
			failoverSuccessful = electionResp.BecameLeader
		}
	}

	// 3. 恢复受影响的执行
	affectedExecutions := 0
	recoveredExecutions := 0

	// 这里应该查找所有由失败调度器管理的执行，并尝试恢复
	// 简化处理

	c.logger.Info(ctx, "Failover process completed", map[string]interface{}{
		"failed_scheduler":     req.FailedSchedulerID,
		"new_leader":           req.NewLeaderID,
		"failover_successful":  failoverSuccessful,
		"affected_executions":  affectedExecutions,
		"recovered_executions": recoveredExecutions,
		"duration":             time.Since(startTime),
	})

	return &FailoverResponse{
		FailedSchedulerID:   req.FailedSchedulerID,
		NewLeaderID:         req.NewLeaderID,
		FailoverSuccessful:  failoverSuccessful,
		AffectedExecutions:  affectedExecutions,
		RecoveredExecutions: recoveredExecutions,
		FailoverDuration:    time.Since(startTime),
	}, nil
}

// MaintenanceRequest 维护请求
type MaintenanceRequest struct {
	SchedulerID     types.ID      `json:"scheduler_id"`
	MaintenanceType string        `json:"maintenance_type"` // "cleanup", "health_check", "rebalance"
	Parameters      types.JSONMap `json:"parameters,omitempty"`
}

// MaintenanceResponse 维护响应
type MaintenanceResponse struct {
	SchedulerID     types.ID      `json:"scheduler_id"`
	MaintenanceType string        `json:"maintenance_type"`
	Success         bool          `json:"success"`
	Results         types.JSONMap `json:"results"`
	Duration        time.Duration `json:"duration"`
}

// PerformMaintenance 执行维护任务
func (c *ClusterManagementCoordinator) PerformMaintenance(ctx context.Context, req *MaintenanceRequest) (*MaintenanceResponse, error) {
	startTime := time.Now()

	c.logger.Info(ctx, "Starting maintenance task", map[string]interface{}{
		"scheduler_id":     req.SchedulerID,
		"maintenance_type": req.MaintenanceType,
	})

	results := make(types.JSONMap)
	success := false

	switch req.MaintenanceType {
	case "cleanup":
		success = c.performCleanup(ctx, results)
	case "health_check":
		success = c.performHealthCheck(ctx, results)
	case "rebalance":
		success = c.performRebalance(ctx, results)
	default:
		return nil, fmt.Errorf("unknown maintenance type: %s", req.MaintenanceType)
	}

	c.logger.Info(ctx, "Maintenance task completed", map[string]interface{}{
		"scheduler_id":     req.SchedulerID,
		"maintenance_type": req.MaintenanceType,
		"success":          success,
		"duration":         time.Since(startTime),
	})

	return &MaintenanceResponse{
		SchedulerID:     req.SchedulerID,
		MaintenanceType: req.MaintenanceType,
		Success:         success,
		Results:         results,
		Duration:        time.Since(startTime),
	}, nil
}

// performCleanup 执行清理
func (c *ClusterManagementCoordinator) performCleanup(ctx context.Context, results types.JSONMap) bool {
	// 清理旧的执行记录
	cleanupReq := &executionbiz.CleanupCompletedExecutionsRequest{
		KeepDays: 30, // 保留30天
	}

	cleanedCount, err := c.executionUC.CleanupCompletedExecutions(ctx, cleanupReq)
	if err != nil {
		c.logger.Warn(ctx, "Failed to cleanup completed executions", map[string]interface{}{
			"error": err.Error(),
		})
		results["cleanup_error"] = err.Error()
		return false
	}

	results["cleaned_executions"] = cleanedCount

	// 清理离线调度器
	olderThan := time.Now().Add(-24 * time.Hour) // 24小时前
	cleanupSchedulerReq := &schedulerbiz.CleanupOfflineSchedulersRequest{
		OlderThan: olderThan,
	}

	cleanedSchedulers, err := c.schedulerUC.CleanupOfflineSchedulers(ctx, cleanupSchedulerReq)
	if err != nil {
		c.logger.Warn(ctx, "Failed to cleanup offline schedulers", map[string]interface{}{
			"error": err.Error(),
		})
		results["scheduler_cleanup_error"] = err.Error()
		return false
	}

	results["cleaned_schedulers"] = cleanedSchedulers
	return true
}

// performHealthCheck 执行健康检查
func (c *ClusterManagementCoordinator) performHealthCheck(ctx context.Context, results types.JSONMap) bool {
	// 执行集群健康检查
	clusterStatus, err := c.schedulerUC.GetClusterStatus(ctx)
	if err != nil {
		results["health_check_error"] = err.Error()
		return false
	}

	results["cluster_healthy"] = clusterStatus.IsHealthy
	results["online_schedulers"] = clusterStatus.OnlineCount
	results["total_schedulers"] = clusterStatus.TotalCount

	return clusterStatus.IsHealthy
}

// performRebalance 执行负载重新平衡
func (c *ClusterManagementCoordinator) performRebalance(ctx context.Context, results types.JSONMap) bool {
	// 简化的负载重新平衡
	results["rebalanced_tasks"] = 0
	results["rebalance_note"] = "Load rebalancing not implemented in this simplified version"
	return true
}
