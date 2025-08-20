package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/app/infra/interfaces"
	"github.com/jobs/scheduler/internal/app/types"
)

// UseCase Executor业务用例
// 遵循架构指南的usecase_[功能].go命名规范，组织执行器管理功能
type UseCase struct {
	executorRepo   Repository
	queryService   QueryService
	eventPublisher interfaces.EventPublisher
	txManager      interfaces.TransactionManager
}

// NewUseCase 创建Executor UseCase
func NewUseCase(
	executorRepo Repository,
	queryService QueryService,
	eventPublisher interfaces.EventPublisher,
	txManager interfaces.TransactionManager,
) *UseCase {
	return &UseCase{
		executorRepo:   executorRepo,
		queryService:   queryService,
		eventPublisher: eventPublisher,
		txManager:      txManager,
	}
}

// 执行器注册相关

// RegisterExecutorRequest 注册执行器请求
type RegisterExecutorRequest struct {
	Name           string `json:"name" validate:"required,min=1,max=100"`
	InstanceID     string `json:"instance_id" validate:"required,min=1,max=100"`
	BaseURL        string `json:"base_url" validate:"required,url"`
	HealthCheckURL string `json:"health_check_url,omitempty"`
}

// Validate 验证请求
func (r *RegisterExecutorRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("executor name is required")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("executor name too long")
	}
	if r.InstanceID == "" {
		return fmt.Errorf("instance ID is required")
	}
	if len(r.InstanceID) > 100 {
		return fmt.Errorf("instance ID too long")
	}
	if r.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	return nil
}

// RegisterExecutorResponse 注册执行器响应
type RegisterExecutorResponse struct {
	ExecutorID types.ID       `json:"executor_id"`
	Name       string         `json:"name"`
	InstanceID string         `json:"instance_id"`
	Status     ExecutorStatus `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
}

// RegisterExecutor 注册执行器
func (uc *UseCase) RegisterExecutor(ctx context.Context, req *RegisterExecutorRequest) (*RegisterExecutorResponse, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 检查实例ID是否已存在
	exists, err := uc.executorRepo.ExistsByInstanceID(ctx, req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check executor existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("executor with instance ID '%s' already exists", req.InstanceID)
	}

	// 在事务中执行
	var executor *Executor
	err = uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 创建执行器实体
		executor, err = NewExecutor(req.Name, req.InstanceID, req.BaseURL)
		if err != nil {
			return fmt.Errorf("failed to create executor: %w", err)
		}

		// 设置健康检查URL
		if req.HealthCheckURL != "" {
			if err := executor.UpdateConfig(req.BaseURL, req.HealthCheckURL); err != nil {
				return fmt.Errorf("failed to set health check URL: %w", err)
			}
		}

		// 保存到仓储
		if err := uc.executorRepo.Save(ctx, executor); err != nil {
			return fmt.Errorf("failed to save executor: %w", err)
		}

		// 发布领域事件
		events := executor.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		// 清除事件
		executor.ClearDomainEvents()

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &RegisterExecutorResponse{
		ExecutorID: executor.ID(),
		Name:       executor.Name(),
		InstanceID: executor.InstanceID(),
		Status:     executor.Status(),
		CreatedAt:  executor.CreatedAt(),
	}, nil
}

// 执行器查询相关

// GetExecutorRequest 获取执行器请求
type GetExecutorRequest struct {
	ExecutorID types.ID `json:"executor_id" validate:"required"`
}

// GetExecutorResponse 获取执行器响应
type GetExecutorResponse struct {
	Executor          *Executor    `json:"executor"`
	TaskCount         int          `json:"task_count"`
	RunningTasks      int          `json:"running_tasks"`
	CompletedTasks    int64        `json:"completed_tasks"`
	FailedTasks       int64        `json:"failed_tasks"`
	LastExecutionTime *time.Time   `json:"last_execution_time,omitempty"`
	HealthStats       *HealthStats `json:"health_stats"`
}

// GetExecutor 获取执行器详情
func (uc *UseCase) GetExecutor(ctx context.Context, req *GetExecutorRequest) (*GetExecutorResponse, error) {
	if req.ExecutorID.IsZero() {
		return nil, fmt.Errorf("executor ID is required")
	}

	// 获取执行器概览
	overview, err := uc.queryService.GetExecutorOverview(ctx, req.ExecutorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor overview: %w", err)
	}

	return &GetExecutorResponse{
		Executor:          overview.Executor,
		TaskCount:         overview.TaskCount,
		RunningTasks:      overview.RunningTasks,
		CompletedTasks:    overview.CompletedTasks,
		FailedTasks:       overview.FailedTasks,
		LastExecutionTime: overview.LastExecutionTime,
		HealthStats:       overview.HealthStats,
	}, nil
}

// ListExecutorsRequest 列表执行器请求
type ListExecutorsRequest struct {
	Filters    ExecutorFilters  `json:"filters"`
	Pagination types.Pagination `json:"pagination"`
}

// ListExecutorsResponse 列表执行器响应
type ListExecutorsResponse struct {
	Executors  []*Executor      `json:"executors"`
	Pagination types.Pagination `json:"pagination"`
	Total      int64            `json:"total"`
}

// ListExecutors 获取执行器列表
func (uc *UseCase) ListExecutors(ctx context.Context, req *ListExecutorsRequest) (*ListExecutorsResponse, error) {
	// 获取执行器列表
	executors, err := uc.executorRepo.FindByFilters(ctx, req.Filters, req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get executors: %w", err)
	}

	// 获取总数
	total, err := uc.executorRepo.Count(ctx, req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count executors: %w", err)
	}

	return &ListExecutorsResponse{
		Executors:  executors,
		Pagination: req.Pagination,
		Total:      total,
	}, nil
}

// GetAvailableExecutors 获取可用执行器列表
func (uc *UseCase) GetAvailableExecutors(ctx context.Context) ([]*Executor, error) {
	return uc.executorRepo.FindAvailableExecutors(ctx)
}

// 执行器状态管理

// UpdateExecutorStatusRequest 更新执行器状态请求
type UpdateExecutorStatusRequest struct {
	ExecutorID types.ID       `json:"executor_id" validate:"required"`
	Status     ExecutorStatus `json:"status" validate:"required"`
	Reason     string         `json:"reason,omitempty"`
}

// UpdateExecutorStatus 更新执行器状态
func (uc *UseCase) UpdateExecutorStatus(ctx context.Context, req *UpdateExecutorStatusRequest) error {
	if req.ExecutorID.IsZero() {
		return fmt.Errorf("executor ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 获取执行器
		executor, err := uc.executorRepo.FindByID(ctx, req.ExecutorID)
		if err != nil {
			return fmt.Errorf("failed to find executor: %w", err)
		}

		// 更新状态
		if err := executor.UpdateStatus(req.Status, req.Reason); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}

		// 保存更新
		if err := uc.executorRepo.Update(ctx, executor); err != nil {
			return fmt.Errorf("failed to save executor: %w", err)
		}

		// 发布领域事件
		events := executor.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		executor.ClearDomainEvents()
		return nil
	})
}

// SetMaintenanceMode 设置维护模式
func (uc *UseCase) SetMaintenanceMode(ctx context.Context, executorID types.ID, reason string) error {
	if executorID.IsZero() {
		return fmt.Errorf("executor ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		executor, err := uc.executorRepo.FindByID(ctx, executorID)
		if err != nil {
			return fmt.Errorf("failed to find executor: %w", err)
		}

		if err := executor.EnterMaintenance(reason); err != nil {
			return fmt.Errorf("failed to enter maintenance: %w", err)
		}

		if err := uc.executorRepo.Update(ctx, executor); err != nil {
			return fmt.Errorf("failed to save executor: %w", err)
		}

		// 发布领域事件
		events := executor.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		executor.ClearDomainEvents()
		return nil
	})
}

// BringOnline 使执行器上线
func (uc *UseCase) BringOnline(ctx context.Context, executorID types.ID) error {
	if executorID.IsZero() {
		return fmt.Errorf("executor ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		executor, err := uc.executorRepo.FindByID(ctx, executorID)
		if err != nil {
			return fmt.Errorf("failed to find executor: %w", err)
		}

		if err := executor.GoOnline(); err != nil {
			return fmt.Errorf("failed to go online: %w", err)
		}

		if err := uc.executorRepo.Update(ctx, executor); err != nil {
			return fmt.Errorf("failed to save executor: %w", err)
		}

		// 发布领域事件
		events := executor.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		executor.ClearDomainEvents()
		return nil
	})
}

// TakeOffline 使执行器离线
func (uc *UseCase) TakeOffline(ctx context.Context, executorID types.ID, reason string) error {
	if executorID.IsZero() {
		return fmt.Errorf("executor ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		executor, err := uc.executorRepo.FindByID(ctx, executorID)
		if err != nil {
			return fmt.Errorf("failed to find executor: %w", err)
		}

		if err := executor.GoOffline(reason); err != nil {
			return fmt.Errorf("failed to go offline: %w", err)
		}

		if err := uc.executorRepo.Update(ctx, executor); err != nil {
			return fmt.Errorf("failed to save executor: %w", err)
		}

		// 发布领域事件
		events := executor.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		executor.ClearDomainEvents()
		return nil
	})
}

// 执行器配置管理

// UpdateExecutorConfigRequest 更新执行器配置请求
type UpdateExecutorConfigRequest struct {
	ExecutorID     types.ID `json:"executor_id" validate:"required"`
	Name           string   `json:"name,omitempty"`
	BaseURL        string   `json:"base_url,omitempty"`
	HealthCheckURL string   `json:"health_check_url,omitempty"`
}

// UpdateExecutorConfig 更新执行器配置
func (uc *UseCase) UpdateExecutorConfig(ctx context.Context, req *UpdateExecutorConfigRequest) error {
	if req.ExecutorID.IsZero() {
		return fmt.Errorf("executor ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		executor, err := uc.executorRepo.FindByID(ctx, req.ExecutorID)
		if err != nil {
			return fmt.Errorf("failed to find executor: %w", err)
		}

		// 更新名称
		if req.Name != "" {
			if err := executor.UpdateName(req.Name); err != nil {
				return fmt.Errorf("failed to update name: %w", err)
			}
		}

		// 更新配置
		if req.BaseURL != "" || req.HealthCheckURL != "" {
			baseURL := req.BaseURL
			if baseURL == "" {
				baseURL = executor.Config().BaseURL
			}
			healthCheckURL := req.HealthCheckURL
			if healthCheckURL == "" {
				healthCheckURL = executor.Config().HealthCheckURL
			}

			if err := executor.UpdateConfig(baseURL, healthCheckURL); err != nil {
				return fmt.Errorf("failed to update config: %w", err)
			}
		}

		// 保存更新
		if err := uc.executorRepo.Update(ctx, executor); err != nil {
			return fmt.Errorf("failed to save executor: %w", err)
		}

		// 发布领域事件
		events := executor.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		executor.ClearDomainEvents()
		return nil
	})
}

// 健康检查相关

// PerformHealthCheckRequest 执行健康检查请求
type PerformHealthCheckRequest struct {
	ExecutorID types.ID `json:"executor_id" validate:"required"`
}

// PerformHealthCheckResponse 执行健康检查响应
type PerformHealthCheckResponse struct {
	ExecutorID   types.ID  `json:"executor_id"`
	IsHealthy    bool      `json:"is_healthy"`
	ResponseTime int64     `json:"response_time"` // 毫秒
	ErrorMsg     string    `json:"error_msg,omitempty"`
	CheckTime    time.Time `json:"check_time"`
}

// PerformHealthCheck 执行健康检查
func (uc *UseCase) PerformHealthCheck(ctx context.Context, req *PerformHealthCheckRequest) (*PerformHealthCheckResponse, error) {
	if req.ExecutorID.IsZero() {
		return nil, fmt.Errorf("executor ID is required")
	}

	var isHealthy bool
	var responseTime int64
	var errorMsg string
	checkTime := time.Now()

	err := uc.txManager.Execute(ctx, func(ctx context.Context) error {
		executor, err := uc.executorRepo.FindByID(ctx, req.ExecutorID)
		if err != nil {
			return fmt.Errorf("failed to find executor: %w", err)
		}

		// 这里应该实际调用执行器的健康检查接口
		// 简化处理，假设健康检查成功
		isHealthy = true
		responseTime = 100 // 100ms

		// 更新健康状态
		if isHealthy {
			executor.MarkHealthy()
		} else {
			executor.MarkUnhealthy()
		}

		// 保存更新
		if err := uc.executorRepo.Update(ctx, executor); err != nil {
			return fmt.Errorf("failed to save executor: %w", err)
		}

		// 发布领域事件
		events := executor.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		executor.ClearDomainEvents()
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &PerformHealthCheckResponse{
		ExecutorID:   req.ExecutorID,
		IsHealthy:    isHealthy,
		ResponseTime: responseTime,
		ErrorMsg:     errorMsg,
		CheckTime:    checkTime,
	}, nil
}

// BatchHealthCheck 批量健康检查
func (uc *UseCase) BatchHealthCheck(ctx context.Context, interval time.Duration) error {
	// 获取需要健康检查的执行器
	executors, err := uc.executorRepo.FindExecutorsNeedingHealthCheck(ctx, interval)
	if err != nil {
		return fmt.Errorf("failed to find executors needing health check: %w", err)
	}

	// 逐个进行健康检查
	for _, executor := range executors {
		req := &PerformHealthCheckRequest{ExecutorID: executor.ID()}
		if _, err := uc.PerformHealthCheck(ctx, req); err != nil {
			// 记录错误但继续检查其他执行器
			// 这里应该使用日志记录
			continue
		}
	}

	return nil
}

// 执行器注销

// UnregisterExecutorRequest 注销执行器请求
type UnregisterExecutorRequest struct {
	ExecutorID types.ID `json:"executor_id" validate:"required"`
	Reason     string   `json:"reason,omitempty"`
}

// UnregisterExecutor 注销执行器
func (uc *UseCase) UnregisterExecutor(ctx context.Context, req *UnregisterExecutorRequest) error {
	if req.ExecutorID.IsZero() {
		return fmt.Errorf("executor ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		executor, err := uc.executorRepo.FindByID(ctx, req.ExecutorID)
		if err != nil {
			return fmt.Errorf("failed to find executor: %w", err)
		}

		// 先使执行器离线
		if err := executor.GoOffline(req.Reason); err != nil {
			return fmt.Errorf("failed to take executor offline: %w", err)
		}

		// 添加注销事件
		executor.AddDomainEvent(ExecutorUnregisteredEvent{
			ExecutorID:     executor.ID(),
			Name:           executor.Name(),
			InstanceID:     executor.InstanceID(),
			UnregisteredAt: time.Now(),
			Reason:         req.Reason,
		})

		// 删除执行器
		if err := uc.executorRepo.Delete(ctx, req.ExecutorID); err != nil {
			return fmt.Errorf("failed to delete executor: %w", err)
		}

		// 发布领域事件
		events := executor.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		return nil
	})
}

// 统计和分析相关

// GetExecutorStatisticsRequest 获取执行器统计请求
type GetExecutorStatisticsRequest struct {
	TimeRange TimeRange `json:"time_range"`
}

// GetExecutorStatistics 获取执行器统计
func (uc *UseCase) GetExecutorStatistics(ctx context.Context, req *GetExecutorStatisticsRequest) (*ExecutorStatistics, error) {
	return uc.queryService.GetExecutorStatistics(ctx, req.TimeRange)
}

// SearchExecutorsRequest 搜索执行器请求
type SearchExecutorsRequest struct {
	Query      string           `json:"query" validate:"required"`
	Pagination types.Pagination `json:"pagination"`
}

// SearchExecutorsResponse 搜索执行器响应
type SearchExecutorsResponse struct {
	Executors  []*Executor      `json:"executors"`
	Pagination types.Pagination `json:"pagination"`
	Total      int64            `json:"total"`
}

// SearchExecutors 搜索执行器
func (uc *UseCase) SearchExecutors(ctx context.Context, req *SearchExecutorsRequest) (*SearchExecutorsResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	executors, err := uc.queryService.SearchExecutors(ctx, req.Query, req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to search executors: %w", err)
	}

	// 简化处理，实际应该获取搜索结果总数
	total := int64(len(executors))

	return &SearchExecutorsResponse{
		Executors:  executors,
		Pagination: req.Pagination,
		Total:      total,
	}, nil
}
