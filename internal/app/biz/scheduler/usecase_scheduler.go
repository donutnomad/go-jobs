package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/app/infra/interfaces"
	"github.com/jobs/scheduler/internal/app/types"
)

// UseCase Scheduler业务用例
// 遵循架构指南的usecase_[功能].go命名规范，组织调度器集群管理功能
type UseCase struct {
	schedulerRepo  Repository
	queryService   QueryService
	eventPublisher interfaces.EventPublisher
	txManager      interfaces.TransactionManager
}

// NewUseCase 创建Scheduler UseCase
func NewUseCase(
	schedulerRepo Repository,
	queryService QueryService,
	eventPublisher interfaces.EventPublisher,
	txManager interfaces.TransactionManager,
) *UseCase {
	return &UseCase{
		schedulerRepo:  schedulerRepo,
		queryService:   queryService,
		eventPublisher: eventPublisher,
		txManager:      txManager,
	}
}

// 调度器注册相关

// RegisterSchedulerRequest 注册调度器请求
type RegisterSchedulerRequest struct {
	InstanceID    string        `json:"instance_id" validate:"required,min=1,max=100"`
	ClusterConfig ClusterConfig `json:"cluster_config"`
}

// Validate 验证请求
func (r *RegisterSchedulerRequest) Validate() error {
	if r.InstanceID == "" {
		return fmt.Errorf("instance ID is required")
	}
	if len(r.InstanceID) > 100 {
		return fmt.Errorf("instance ID too long")
	}
	if !r.ClusterConfig.IsValid() {
		return fmt.Errorf("invalid cluster config")
	}
	return nil
}

// RegisterSchedulerResponse 注册调度器响应
type RegisterSchedulerResponse struct {
	SchedulerID types.ID        `json:"scheduler_id"`
	InstanceID  string          `json:"instance_id"`
	Status      SchedulerStatus `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
}

// RegisterScheduler 注册调度器
func (uc *UseCase) RegisterScheduler(ctx context.Context, req *RegisterSchedulerRequest) (*RegisterSchedulerResponse, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 检查实例ID是否已存在
	exists, err := uc.schedulerRepo.ExistsByInstanceID(ctx, req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check scheduler existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("scheduler with instance ID '%s' already exists", req.InstanceID)
	}

	// 在事务中执行
	var scheduler *Scheduler
	err = uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 创建调度器实体
		scheduler, err = NewScheduler(req.InstanceID, req.ClusterConfig)
		if err != nil {
			return fmt.Errorf("failed to create scheduler: %w", err)
		}

		// 保存到仓储
		if err := uc.schedulerRepo.Save(ctx, scheduler); err != nil {
			return fmt.Errorf("failed to save scheduler: %w", err)
		}

		// 发布领域事件
		events := scheduler.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		// 清除事件
		scheduler.ClearDomainEvents()

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &RegisterSchedulerResponse{
		SchedulerID: scheduler.ID(),
		InstanceID:  scheduler.InstanceID(),
		Status:      scheduler.Status(),
		CreatedAt:   scheduler.CreatedAt(),
	}, nil
}

// 调度器查询相关

// GetSchedulerRequest 获取调度器请求
type GetSchedulerRequest struct {
	SchedulerID types.ID `json:"scheduler_id" validate:"required"`
}

// GetSchedulerResponse 获取调度器响应
type GetSchedulerResponse struct {
	Scheduler           *Scheduler     `json:"scheduler"`
	ScheduledTasksCount int64          `json:"scheduled_tasks_count"`
	RunningTasksCount   int64          `json:"running_tasks_count"`
	CompletedTasksCount int64          `json:"completed_tasks_count"`
	LeadershipDuration  *time.Duration `json:"leadership_duration,omitempty"`
	ClusterRole         string         `json:"cluster_role"`
}

// GetScheduler 获取调度器详情
func (uc *UseCase) GetScheduler(ctx context.Context, req *GetSchedulerRequest) (*GetSchedulerResponse, error) {
	if req.SchedulerID.IsZero() {
		return nil, fmt.Errorf("scheduler ID is required")
	}

	// 获取调度器概览
	overview, err := uc.queryService.GetSchedulerOverview(ctx, req.SchedulerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler overview: %w", err)
	}

	return &GetSchedulerResponse{
		Scheduler:           overview.Scheduler,
		ScheduledTasksCount: overview.ScheduledTasksCount,
		RunningTasksCount:   overview.RunningTasksCount,
		CompletedTasksCount: overview.CompletedTasksCount,
		LeadershipDuration:  overview.LeadershipDuration,
		ClusterRole:         overview.ClusterRole,
	}, nil
}

// ListSchedulersRequest 列表调度器请求
type ListSchedulersRequest struct {
	Filters    SchedulerFilters `json:"filters"`
	Pagination types.Pagination `json:"pagination"`
}

// ListSchedulersResponse 列表调度器响应
type ListSchedulersResponse struct {
	Schedulers []*Scheduler     `json:"schedulers"`
	Pagination types.Pagination `json:"pagination"`
	Total      int64            `json:"total"`
}

// ListSchedulers 获取调度器列表
func (uc *UseCase) ListSchedulers(ctx context.Context, req *ListSchedulersRequest) (*ListSchedulersResponse, error) {
	// 获取调度器列表
	schedulers, err := uc.schedulerRepo.FindByFilters(ctx, req.Filters, req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedulers: %w", err)
	}

	// 获取总数
	total, err := uc.schedulerRepo.Count(ctx, req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count schedulers: %w", err)
	}

	return &ListSchedulersResponse{
		Schedulers: schedulers,
		Pagination: req.Pagination,
		Total:      total,
	}, nil
}

// GetOnlineSchedulers 获取在线调度器列表
func (uc *UseCase) GetOnlineSchedulers(ctx context.Context) ([]*Scheduler, error) {
	return uc.schedulerRepo.FindOnlineSchedulers(ctx)
}

// GetCurrentLeader 获取当前领导者
func (uc *UseCase) GetCurrentLeader(ctx context.Context) (*Scheduler, error) {
	return uc.schedulerRepo.FindCurrentLeader(ctx)
}

// 调度器状态管理

// UpdateSchedulerStatusRequest 更新调度器状态请求
type UpdateSchedulerStatusRequest struct {
	SchedulerID types.ID        `json:"scheduler_id" validate:"required"`
	Status      SchedulerStatus `json:"status" validate:"required"`
	Reason      string          `json:"reason,omitempty"`
}

// UpdateSchedulerStatus 更新调度器状态
func (uc *UseCase) UpdateSchedulerStatus(ctx context.Context, req *UpdateSchedulerStatusRequest) error {
	if req.SchedulerID.IsZero() {
		return fmt.Errorf("scheduler ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 获取调度器
		scheduler, err := uc.schedulerRepo.FindByID(ctx, req.SchedulerID)
		if err != nil {
			return fmt.Errorf("failed to find scheduler: %w", err)
		}

		// 更新状态
		switch req.Status {
		case SchedulerStatusOnline:
			err = scheduler.GoOnline()
		case SchedulerStatusOffline:
			err = scheduler.GoOffline(req.Reason)
		case SchedulerStatusMaintenance:
			err = scheduler.EnterMaintenance(req.Reason)
		default:
			return fmt.Errorf("invalid scheduler status: %s", req.Status)
		}

		if err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}

		// 保存更新
		if err := uc.schedulerRepo.Update(ctx, scheduler); err != nil {
			return fmt.Errorf("failed to save scheduler: %w", err)
		}

		// 发布领域事件
		events := scheduler.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		scheduler.ClearDomainEvents()
		return nil
	})
}

// 心跳管理

// UpdateHeartbeatRequest 更新心跳请求
type UpdateHeartbeatRequest struct {
	SchedulerID types.ID `json:"scheduler_id" validate:"required"`
}

// UpdateHeartbeat 更新心跳
func (uc *UseCase) UpdateHeartbeat(ctx context.Context, req *UpdateHeartbeatRequest) error {
	if req.SchedulerID.IsZero() {
		return fmt.Errorf("scheduler ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		scheduler, err := uc.schedulerRepo.FindByID(ctx, req.SchedulerID)
		if err != nil {
			return fmt.Errorf("failed to find scheduler: %w", err)
		}

		if err := scheduler.UpdateHeartbeat(); err != nil {
			return fmt.Errorf("failed to update heartbeat: %w", err)
		}

		if err := uc.schedulerRepo.Update(ctx, scheduler); err != nil {
			return fmt.Errorf("failed to save scheduler: %w", err)
		}

		return nil
	})
}

// ProcessStaleSchedulers 处理过期调度器
func (uc *UseCase) ProcessStaleSchedulers(ctx context.Context, staleThreshold time.Duration) error {
	// 获取过期的调度器
	staleSchedulers, err := uc.schedulerRepo.FindStaleSchedulers(ctx, staleThreshold)
	if err != nil {
		return fmt.Errorf("failed to find stale schedulers: %w", err)
	}

	// 批量标记为离线
	schedulerIDs := make([]types.ID, len(staleSchedulers))
	for i, scheduler := range staleSchedulers {
		schedulerIDs[i] = scheduler.ID()
	}

	if len(schedulerIDs) > 0 {
		return uc.schedulerRepo.BatchUpdateStatus(ctx, schedulerIDs, SchedulerStatusOffline, "heartbeat timeout")
	}

	return nil
}

// 领导者选举相关

// StartLeaderElectionRequest 开始领导者选举请求
type StartLeaderElectionRequest struct {
	SchedulerID types.ID `json:"scheduler_id" validate:"required"`
}

// StartLeaderElection 开始领导者选举
func (uc *UseCase) StartLeaderElection(ctx context.Context, req *StartLeaderElectionRequest) error {
	if req.SchedulerID.IsZero() {
		return fmt.Errorf("scheduler ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		scheduler, err := uc.schedulerRepo.FindByID(ctx, req.SchedulerID)
		if err != nil {
			return fmt.Errorf("failed to find scheduler: %w", err)
		}

		if err := scheduler.StartElection(); err != nil {
			return fmt.Errorf("failed to start election: %w", err)
		}

		if err := uc.schedulerRepo.Update(ctx, scheduler); err != nil {
			return fmt.Errorf("failed to save scheduler: %w", err)
		}

		// 发布领域事件
		events := scheduler.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		scheduler.ClearDomainEvents()
		return nil
	})
}

// TryBecomeLeaderRequest 尝试成为领导者请求
type TryBecomeLeaderRequest struct {
	SchedulerID types.ID      `json:"scheduler_id" validate:"required"`
	LockTimeout time.Duration `json:"lock_timeout,omitempty"`
}

// TryBecomeLeaderResponse 尝试成为领导者响应
type TryBecomeLeaderResponse struct {
	Success         bool       `json:"success"`
	IsLeader        bool       `json:"is_leader"`
	LeaderElectedAt *time.Time `json:"leader_elected_at,omitempty"`
}

// TryBecomeLeader 尝试成为领导者
func (uc *UseCase) TryBecomeLeader(ctx context.Context, req *TryBecomeLeaderRequest) (*TryBecomeLeaderResponse, error) {
	if req.SchedulerID.IsZero() {
		return nil, fmt.Errorf("scheduler ID is required")
	}

	if req.LockTimeout == 0 {
		req.LockTimeout = 30 * time.Second // 默认30秒
	}

	var success, isLeader bool
	var leaderElectedAt *time.Time

	err := uc.txManager.Execute(ctx, func(ctx context.Context) error {
		scheduler, err := uc.schedulerRepo.FindByID(ctx, req.SchedulerID)
		if err != nil {
			return fmt.Errorf("failed to find scheduler: %w", err)
		}

		// 检查是否应该尝试成为领导者
		if !scheduler.ShouldTryBecomeLeader() {
			success = false
			isLeader = scheduler.IsLeader()
			leaderElectedAt = scheduler.LeaderElectedAt()
			return nil
		}

		// 尝试获取分布式锁
		lockKey := scheduler.LeadershipLock()
		acquired, err := uc.schedulerRepo.AcquireLeadershipLock(ctx, req.SchedulerID, lockKey, req.LockTimeout)
		if err != nil {
			return fmt.Errorf("failed to acquire leadership lock: %w", err)
		}

		if !acquired {
			success = false
			isLeader = false
			return nil
		}

		// 成为领导者
		if err := scheduler.BecomeLeader(); err != nil {
			// 释放锁
			uc.schedulerRepo.ReleaseLeadershipLock(ctx, lockKey)
			return fmt.Errorf("failed to become leader: %w", err)
		}

		// 保存更新
		if err := uc.schedulerRepo.Update(ctx, scheduler); err != nil {
			// 释放锁
			uc.schedulerRepo.ReleaseLeadershipLock(ctx, lockKey)
			return fmt.Errorf("failed to save scheduler: %w", err)
		}

		success = true
		isLeader = true
		leaderElectedAt = scheduler.LeaderElectedAt()

		// 发布领域事件
		events := scheduler.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		scheduler.ClearDomainEvents()
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &TryBecomeLeaderResponse{
		Success:         success,
		IsLeader:        isLeader,
		LeaderElectedAt: leaderElectedAt,
	}, nil
}

// ResignLeadershipRequest 辞去领导权请求
type ResignLeadershipRequest struct {
	SchedulerID types.ID `json:"scheduler_id" validate:"required"`
	Reason      string   `json:"reason,omitempty"`
}

// ResignLeadership 辞去领导权
func (uc *UseCase) ResignLeadership(ctx context.Context, req *ResignLeadershipRequest) error {
	if req.SchedulerID.IsZero() {
		return fmt.Errorf("scheduler ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		scheduler, err := uc.schedulerRepo.FindByID(ctx, req.SchedulerID)
		if err != nil {
			return fmt.Errorf("failed to find scheduler: %w", err)
		}

		if !scheduler.IsLeader() {
			return fmt.Errorf("scheduler is not a leader")
		}

		// 释放分布式锁
		lockKey := scheduler.LeadershipLock()
		if err := uc.schedulerRepo.ReleaseLeadershipLock(ctx, lockKey); err != nil {
			return fmt.Errorf("failed to release leadership lock: %w", err)
		}

		// 成为跟随者
		if err := scheduler.BecomeFollower(req.Reason); err != nil {
			return fmt.Errorf("failed to become follower: %w", err)
		}

		// 保存更新
		if err := uc.schedulerRepo.Update(ctx, scheduler); err != nil {
			return fmt.Errorf("failed to save scheduler: %w", err)
		}

		// 发布领域事件
		events := scheduler.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		scheduler.ClearDomainEvents()
		return nil
	})
}

// RefreshLeadershipLockRequest 刷新领导权锁请求
type RefreshLeadershipLockRequest struct {
	SchedulerID types.ID      `json:"scheduler_id" validate:"required"`
	LockTimeout time.Duration `json:"lock_timeout,omitempty"`
}

// RefreshLeadershipLock 刷新领导权锁
func (uc *UseCase) RefreshLeadershipLock(ctx context.Context, req *RefreshLeadershipLockRequest) error {
	if req.SchedulerID.IsZero() {
		return fmt.Errorf("scheduler ID is required")
	}

	if req.LockTimeout == 0 {
		req.LockTimeout = 30 * time.Second
	}

	scheduler, err := uc.schedulerRepo.FindByID(ctx, req.SchedulerID)
	if err != nil {
		return fmt.Errorf("failed to find scheduler: %w", err)
	}

	if !scheduler.IsLeader() {
		return fmt.Errorf("scheduler is not a leader")
	}

	lockKey := scheduler.LeadershipLock()
	return uc.schedulerRepo.RefreshLock(ctx, req.SchedulerID, lockKey, req.LockTimeout)
}

// 集群管理相关

// GetClusterStatus 获取集群状态
func (uc *UseCase) GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	return uc.queryService.GetClusterStatus(ctx)
}

// GetClusterTopology 获取集群拓扑
func (uc *UseCase) GetClusterTopology(ctx context.Context) (*ClusterTopology, error) {
	return uc.queryService.GetClusterTopology(ctx)
}

// 清理操作

// CleanupOfflineSchedulersRequest 清理离线调度器请求
type CleanupOfflineSchedulersRequest struct {
	OlderThan time.Time `json:"older_than"`
}

// CleanupOfflineSchedulers 清理离线调度器
func (uc *UseCase) CleanupOfflineSchedulers(ctx context.Context, req *CleanupOfflineSchedulersRequest) (int64, error) {
	if req.OlderThan.IsZero() {
		return 0, fmt.Errorf("older_than time is required")
	}

	return uc.schedulerRepo.CleanupOfflineSchedulers(ctx, req.OlderThan)
}

// 统计和分析相关

// GetClusterStatisticsRequest 获取集群统计请求
type GetClusterStatisticsRequest struct {
	TimeRange TimeRange `json:"time_range"`
}

// GetClusterStatistics 获取集群统计
func (uc *UseCase) GetClusterStatistics(ctx context.Context, req *GetClusterStatisticsRequest) (*ClusterStatistics, error) {
	return uc.queryService.GetClusterStatistics(ctx, req.TimeRange)
}

// GetSchedulingMetricsRequest 获取调度指标请求
type GetSchedulingMetricsRequest struct {
	TimeRange TimeRange `json:"time_range"`
}

// GetSchedulingMetrics 获取调度指标
func (uc *UseCase) GetSchedulingMetrics(ctx context.Context, req *GetSchedulingMetricsRequest) (*SchedulingMetrics, error) {
	return uc.queryService.GetSchedulingMetrics(ctx, req.TimeRange)
}

// GetLeadershipHistoryRequest 获取领导权历史请求
type GetLeadershipHistoryRequest struct {
	Days int `json:"days" validate:"min=1"`
}

// GetLeadershipHistory 获取领导权历史
func (uc *UseCase) GetLeadershipHistory(ctx context.Context, req *GetLeadershipHistoryRequest) ([]*LeadershipRecord, error) {
	if req.Days < 1 {
		return nil, fmt.Errorf("days must be at least 1")
	}

	return uc.queryService.GetLeadershipHistory(ctx, req.Days)
}

// GetFailoverHistoryRequest 获取故障转移历史请求
type GetFailoverHistoryRequest struct {
	Days int `json:"days" validate:"min=1"`
}

// GetFailoverHistory 获取故障转移历史
func (uc *UseCase) GetFailoverHistory(ctx context.Context, req *GetFailoverHistoryRequest) ([]*FailoverRecord, error) {
	if req.Days < 1 {
		return nil, fmt.Errorf("days must be at least 1")
	}

	return uc.queryService.GetFailoverHistory(ctx, req.Days)
}

// SearchSchedulersRequest 搜索调度器请求
type SearchSchedulersRequest struct {
	Query      string           `json:"query" validate:"required"`
	Pagination types.Pagination `json:"pagination"`
}

// SearchSchedulersResponse 搜索调度器响应
type SearchSchedulersResponse struct {
	Schedulers []*Scheduler     `json:"schedulers"`
	Pagination types.Pagination `json:"pagination"`
	Total      int64            `json:"total"`
}

// SearchSchedulers 搜索调度器
func (uc *UseCase) SearchSchedulers(ctx context.Context, req *SearchSchedulersRequest) (*SearchSchedulersResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	schedulers, err := uc.queryService.SearchSchedulers(ctx, req.Query, req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to search schedulers: %w", err)
	}

	// 简化处理，实际应该获取搜索结果总数
	total := int64(len(schedulers))

	return &SearchSchedulersResponse{
		Schedulers: schedulers,
		Pagination: req.Pagination,
		Total:      total,
	}, nil
}
