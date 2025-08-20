package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/app/infra/interfaces"
	"github.com/jobs/scheduler/internal/app/types"
)

// UseCase Execution业务用例
// 遵循架构指南的usecase_[功能].go命名规范，组织任务执行管理功能
type UseCase struct {
	executionRepo  Repository
	queryService   QueryService
	eventPublisher interfaces.EventPublisher
	txManager      interfaces.TransactionManager
}

// NewUseCase 创建Execution UseCase
func NewUseCase(
	executionRepo Repository,
	queryService QueryService,
	eventPublisher interfaces.EventPublisher,
	txManager interfaces.TransactionManager,
) *UseCase {
	return &UseCase{
		executionRepo:  executionRepo,
		queryService:   queryService,
		eventPublisher: eventPublisher,
		txManager:      txManager,
	}
}

// 任务执行创建相关

// CreateExecutionRequest 创建任务执行请求
type CreateExecutionRequest struct {
	TaskID        types.ID      `json:"task_id" validate:"required"`
	Parameters    types.JSONMap `json:"parameters,omitempty"`
	ScheduledTime time.Time     `json:"scheduled_time"`
	CallbackURL   string        `json:"callback_url,omitempty"`
}

// Validate 验证请求
func (r *CreateExecutionRequest) Validate() error {
	if r.TaskID.IsZero() {
		return fmt.Errorf("task ID is required")
	}
	if r.ScheduledTime.IsZero() {
		r.ScheduledTime = time.Now()
	}
	return nil
}

// CreateExecutionResponse 创建任务执行响应
type CreateExecutionResponse struct {
	ExecutionID   types.ID        `json:"execution_id"`
	TaskID        types.ID        `json:"task_id"`
	Status        ExecutionStatus `json:"status"`
	ScheduledTime time.Time       `json:"scheduled_time"`
	CreatedAt     time.Time       `json:"created_at"`
}

// CreateExecution 创建任务执行
func (uc *UseCase) CreateExecution(ctx context.Context, req *CreateExecutionRequest) (*CreateExecutionResponse, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 在事务中执行
	var execution *TaskExecution
	err := uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 创建任务执行实体
		var err error
		execution, err = NewTaskExecution(req.TaskID, req.Parameters, req.ScheduledTime)
		if err != nil {
			return fmt.Errorf("failed to create execution: %w", err)
		}

		// 设置回调URL
		if req.CallbackURL != "" {
			if err := execution.UpdateCallbackURL(req.CallbackURL); err != nil {
				return fmt.Errorf("failed to set callback URL: %w", err)
			}
		}

		// 保存到仓储
		if err := uc.executionRepo.Save(ctx, execution); err != nil {
			return fmt.Errorf("failed to save execution: %w", err)
		}

		// 发布领域事件
		events := execution.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		// 清除事件
		execution.ClearDomainEvents()

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &CreateExecutionResponse{
		ExecutionID:   execution.ID(),
		TaskID:        execution.TaskID(),
		Status:        execution.Status(),
		ScheduledTime: execution.GetScheduledTime(),
		CreatedAt:     execution.CreatedAt(),
	}, nil
}

// 任务执行查询相关

// GetExecutionRequest 获取任务执行请求
type GetExecutionRequest struct {
	ExecutionID types.ID `json:"execution_id" validate:"required"`
}

// GetExecutionResponse 获取任务执行响应
type GetExecutionResponse struct {
	Execution    *TaskExecution   `json:"execution"`
	TaskInfo     *TaskInfo        `json:"task_info"`
	ExecutorInfo *ExecutorInfo    `json:"executor_info"`
	RetryHistory []*TaskExecution `json:"retry_history"`
}

// GetExecution 获取任务执行详情
func (uc *UseCase) GetExecution(ctx context.Context, req *GetExecutionRequest) (*GetExecutionResponse, error) {
	if req.ExecutionID.IsZero() {
		return nil, fmt.Errorf("execution ID is required")
	}

	// 获取执行概览
	overview, err := uc.queryService.GetExecutionOverview(ctx, req.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution overview: %w", err)
	}

	return &GetExecutionResponse{
		Execution:    overview.Execution,
		TaskInfo:     overview.TaskInfo,
		ExecutorInfo: overview.ExecutorInfo,
		RetryHistory: overview.RetryHistory,
	}, nil
}

// ListExecutionsRequest 列表任务执行请求
type ListExecutionsRequest struct {
	Filters    ExecutionFilters `json:"filters"`
	Pagination types.Pagination `json:"pagination"`
}

// ListExecutionsResponse 列表任务执行响应
type ListExecutionsResponse struct {
	Executions []*TaskExecution `json:"executions"`
	Pagination types.Pagination `json:"pagination"`
	Total      int64            `json:"total"`
}

// ListExecutions 获取任务执行列表
func (uc *UseCase) ListExecutions(ctx context.Context, req *ListExecutionsRequest) (*ListExecutionsResponse, error) {
	// 获取执行列表
	executions, err := uc.executionRepo.FindByFilters(ctx, req.Filters, req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get executions: %w", err)
	}

	// 获取总数
	total, err := uc.executionRepo.Count(ctx, req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count executions: %w", err)
	}

	return &ListExecutionsResponse{
		Executions: executions,
		Pagination: req.Pagination,
		Total:      total,
	}, nil
}

// GetRunningExecutions 获取运行中的执行
func (uc *UseCase) GetRunningExecutions(ctx context.Context) ([]*TaskExecution, error) {
	return uc.executionRepo.FindRunningExecutions(ctx)
}

// GetPendingExecutions 获取待执行的任务
func (uc *UseCase) GetPendingExecutions(ctx context.Context) ([]*TaskExecution, error) {
	return uc.executionRepo.FindPendingExecutions(ctx)
}

// 任务执行控制相关

// StartExecutionRequest 开始执行请求
type StartExecutionRequest struct {
	ExecutionID types.ID `json:"execution_id" validate:"required"`
	ExecutorID  types.ID `json:"executor_id" validate:"required"`
}

// StartExecution 开始执行
func (uc *UseCase) StartExecution(ctx context.Context, req *StartExecutionRequest) error {
	if req.ExecutionID.IsZero() {
		return fmt.Errorf("execution ID is required")
	}
	if req.ExecutorID.IsZero() {
		return fmt.Errorf("executor ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 获取任务执行
		execution, err := uc.executionRepo.FindByID(ctx, req.ExecutionID)
		if err != nil {
			return fmt.Errorf("failed to find execution: %w", err)
		}

		// 开始执行
		if err := execution.Start(req.ExecutorID); err != nil {
			return fmt.Errorf("failed to start execution: %w", err)
		}

		// 保存更新
		if err := uc.executionRepo.Update(ctx, execution); err != nil {
			return fmt.Errorf("failed to save execution: %w", err)
		}

		// 发布领域事件
		events := execution.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		execution.ClearDomainEvents()
		return nil
	})
}

// CompleteExecutionRequest 完成执行请求
type CompleteExecutionRequest struct {
	ExecutionID types.ID        `json:"execution_id" validate:"required"`
	Status      ExecutionStatus `json:"status" validate:"required"`
	Result      types.JSONMap   `json:"result,omitempty"`
	Logs        string          `json:"logs,omitempty"`
	ErrorMsg    string          `json:"error_msg,omitempty"`
}

// CompleteExecution 完成执行
func (uc *UseCase) CompleteExecution(ctx context.Context, req *CompleteExecutionRequest) error {
	if req.ExecutionID.IsZero() {
		return fmt.Errorf("execution ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		// 获取任务执行
		execution, err := uc.executionRepo.FindByID(ctx, req.ExecutionID)
		if err != nil {
			return fmt.Errorf("failed to find execution: %w", err)
		}

		// 完成执行
		if err := execution.Complete(req.Status, req.Result, req.Logs, req.ErrorMsg); err != nil {
			return fmt.Errorf("failed to complete execution: %w", err)
		}

		// 保存更新
		if err := uc.executionRepo.Update(ctx, execution); err != nil {
			return fmt.Errorf("failed to save execution: %w", err)
		}

		// 发布领域事件
		events := execution.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		execution.ClearDomainEvents()
		return nil
	})
}

// CancelExecutionRequest 取消执行请求
type CancelExecutionRequest struct {
	ExecutionID types.ID `json:"execution_id" validate:"required"`
	Reason      string   `json:"reason,omitempty"`
}

// CancelExecution 取消执行
func (uc *UseCase) CancelExecution(ctx context.Context, req *CancelExecutionRequest) error {
	if req.ExecutionID.IsZero() {
		return fmt.Errorf("execution ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		execution, err := uc.executionRepo.FindByID(ctx, req.ExecutionID)
		if err != nil {
			return fmt.Errorf("failed to find execution: %w", err)
		}

		if err := execution.Cancel(req.Reason); err != nil {
			return fmt.Errorf("failed to cancel execution: %w", err)
		}

		if err := uc.executionRepo.Update(ctx, execution); err != nil {
			return fmt.Errorf("failed to save execution: %w", err)
		}

		// 发布领域事件
		events := execution.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		execution.ClearDomainEvents()
		return nil
	})
}

// RetryExecutionRequest 重试执行请求
type RetryExecutionRequest struct {
	ExecutionID types.ID `json:"execution_id" validate:"required"`
}

// RetryExecution 重试执行
func (uc *UseCase) RetryExecution(ctx context.Context, req *RetryExecutionRequest) error {
	if req.ExecutionID.IsZero() {
		return fmt.Errorf("execution ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		execution, err := uc.executionRepo.FindByID(ctx, req.ExecutionID)
		if err != nil {
			return fmt.Errorf("failed to find execution: %w", err)
		}

		if err := execution.Retry(); err != nil {
			return fmt.Errorf("failed to retry execution: %w", err)
		}

		if err := uc.executionRepo.Update(ctx, execution); err != nil {
			return fmt.Errorf("failed to save execution: %w", err)
		}

		// 发布领域事件
		events := execution.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		execution.ClearDomainEvents()
		return nil
	})
}

// SkipExecutionRequest 跳过执行请求
type SkipExecutionRequest struct {
	ExecutionID types.ID `json:"execution_id" validate:"required"`
	Reason      string   `json:"reason,omitempty"`
}

// SkipExecution 跳过执行
func (uc *UseCase) SkipExecution(ctx context.Context, req *SkipExecutionRequest) error {
	if req.ExecutionID.IsZero() {
		return fmt.Errorf("execution ID is required")
	}

	return uc.txManager.Execute(ctx, func(ctx context.Context) error {
		execution, err := uc.executionRepo.FindByID(ctx, req.ExecutionID)
		if err != nil {
			return fmt.Errorf("failed to find execution: %w", err)
		}

		if err := execution.Skip(req.Reason); err != nil {
			return fmt.Errorf("failed to skip execution: %w", err)
		}

		if err := uc.executionRepo.Update(ctx, execution); err != nil {
			return fmt.Errorf("failed to save execution: %w", err)
		}

		// 发布领域事件
		events := execution.GetDomainEvents()
		for _, event := range events {
			if err := uc.eventPublisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}
		}

		execution.ClearDomainEvents()
		return nil
	})
}

// 批量操作

// BatchCancelExecutionsRequest 批量取消执行请求
type BatchCancelExecutionsRequest struct {
	ExecutionIDs []types.ID `json:"execution_ids" validate:"required"`
	Reason       string     `json:"reason,omitempty"`
}

// BatchCancelExecutions 批量取消执行
func (uc *UseCase) BatchCancelExecutions(ctx context.Context, req *BatchCancelExecutionsRequest) error {
	if len(req.ExecutionIDs) == 0 {
		return fmt.Errorf("execution IDs are required")
	}

	return uc.executionRepo.BatchCancel(ctx, req.ExecutionIDs, req.Reason)
}

// BatchRetryExecutionsRequest 批量重试执行请求
type BatchRetryExecutionsRequest struct {
	ExecutionIDs []types.ID `json:"execution_ids" validate:"required"`
}

// BatchRetryExecutions 批量重试执行
func (uc *UseCase) BatchRetryExecutions(ctx context.Context, req *BatchRetryExecutionsRequest) error {
	if len(req.ExecutionIDs) == 0 {
		return fmt.Errorf("execution IDs are required")
	}

	return uc.executionRepo.BatchRetry(ctx, req.ExecutionIDs)
}

// 超时处理

// HandleTimeoutExecutions 处理超时执行
func (uc *UseCase) HandleTimeoutExecutions(ctx context.Context, timeoutDuration time.Duration) error {
	// 获取超时执行
	executions, err := uc.executionRepo.FindTimeoutExecutions(ctx, timeoutDuration)
	if err != nil {
		return fmt.Errorf("failed to find timeout executions: %w", err)
	}

	// 批量处理超时
	for _, execution := range executions {
		err := uc.txManager.Execute(ctx, func(ctx context.Context) error {
			if err := execution.Timeout(); err != nil {
				return fmt.Errorf("failed to timeout execution %s: %w", execution.ID(), err)
			}

			if err := uc.executionRepo.Update(ctx, execution); err != nil {
				return fmt.Errorf("failed to save execution %s: %w", execution.ID(), err)
			}

			// 发布领域事件
			events := execution.GetDomainEvents()
			for _, event := range events {
				if err := uc.eventPublisher.Publish(ctx, event); err != nil {
					return fmt.Errorf("failed to publish event: %w", err)
				}
			}

			execution.ClearDomainEvents()
			return nil
		})

		if err != nil {
			// 记录错误但继续处理其他执行
			continue
		}
	}

	return nil
}

// 清理操作

// CleanupOldExecutionsRequest 清理旧执行请求
type CleanupOldExecutionsRequest struct {
	OlderThan time.Time `json:"older_than"`
}

// CleanupOldExecutions 清理旧执行
func (uc *UseCase) CleanupOldExecutions(ctx context.Context, req *CleanupOldExecutionsRequest) (int64, error) {
	if req.OlderThan.IsZero() {
		return 0, fmt.Errorf("older_than time is required")
	}

	return uc.executionRepo.CleanupOldExecutions(ctx, req.OlderThan)
}

// CleanupCompletedExecutionsRequest 清理已完成执行请求
type CleanupCompletedExecutionsRequest struct {
	KeepDays int `json:"keep_days" validate:"min=1"`
}

// CleanupCompletedExecutions 清理已完成执行
func (uc *UseCase) CleanupCompletedExecutions(ctx context.Context, req *CleanupCompletedExecutionsRequest) (int64, error) {
	if req.KeepDays < 1 {
		return 0, fmt.Errorf("keep_days must be at least 1")
	}

	return uc.executionRepo.CleanupCompletedExecutions(ctx, req.KeepDays)
}

// 统计和分析相关

// GetExecutionStatisticsRequest 获取执行统计请求
type GetExecutionStatisticsRequest struct {
	TimeRange TimeRange        `json:"time_range"`
	Filters   ExecutionFilters `json:"filters"`
}

// GetExecutionStatistics 获取执行统计
func (uc *UseCase) GetExecutionStatistics(ctx context.Context, req *GetExecutionStatisticsRequest) (*ExecutionStatistics, error) {
	return uc.queryService.GetExecutionStatistics(ctx, req.TimeRange, req.Filters)
}

// GetPerformanceMetricsRequest 获取性能指标请求
type GetPerformanceMetricsRequest struct {
	TaskID    types.ID  `json:"task_id" validate:"required"`
	TimeRange TimeRange `json:"time_range"`
}

// GetPerformanceMetrics 获取性能指标
func (uc *UseCase) GetPerformanceMetrics(ctx context.Context, req *GetPerformanceMetricsRequest) (*PerformanceMetrics, error) {
	if req.TaskID.IsZero() {
		return nil, fmt.Errorf("task ID is required")
	}

	return uc.queryService.GetPerformanceMetrics(ctx, req.TaskID, req.TimeRange)
}

// GetRetryAnalysisRequest 获取重试分析请求
type GetRetryAnalysisRequest struct {
	TaskID    types.ID  `json:"task_id" validate:"required"`
	TimeRange TimeRange `json:"time_range"`
}

// GetRetryAnalysis 获取重试分析
func (uc *UseCase) GetRetryAnalysis(ctx context.Context, req *GetRetryAnalysisRequest) (*RetryAnalysis, error) {
	if req.TaskID.IsZero() {
		return nil, fmt.Errorf("task ID is required")
	}

	return uc.queryService.GetRetryAnalysis(ctx, req.TaskID, req.TimeRange)
}

// GetRunningExecutionSummary 获取运行中执行摘要
func (uc *UseCase) GetRunningExecutionSummary(ctx context.Context) (*RunningExecutionSummary, error) {
	return uc.queryService.GetRunningExecutionSummary(ctx)
}

// GetFailureAnalysisRequest 获取失败分析请求
type GetFailureAnalysisRequest struct {
	TimeRange TimeRange `json:"time_range"`
}

// GetFailureAnalysis 获取失败分析
func (uc *UseCase) GetFailureAnalysis(ctx context.Context, req *GetFailureAnalysisRequest) (*FailureAnalysis, error) {
	return uc.queryService.GetFailureAnalysis(ctx, req.TimeRange)
}

// SearchExecutionsRequest 搜索执行请求
type SearchExecutionsRequest struct {
	Query      string           `json:"query" validate:"required"`
	Pagination types.Pagination `json:"pagination"`
}

// SearchExecutionsResponse 搜索执行响应
type SearchExecutionsResponse struct {
	Executions []*TaskExecution `json:"executions"`
	Pagination types.Pagination `json:"pagination"`
	Total      int64            `json:"total"`
}

// SearchExecutions 搜索执行
func (uc *UseCase) SearchExecutions(ctx context.Context, req *SearchExecutionsRequest) (*SearchExecutionsResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	executions, err := uc.queryService.SearchExecutions(ctx, req.Query, req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to search executions: %w", err)
	}

	// 简化处理，实际应该获取搜索结果总数
	total := int64(len(executions))

	return &SearchExecutionsResponse{
		Executions: executions,
		Pagination: req.Pagination,
		Total:      total,
	}, nil
}
