package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jobs/scheduler/internal/domain/entity"
	domainError "github.com/jobs/scheduler/internal/domain/error"
	"github.com/jobs/scheduler/internal/domain/repository"
	"go.uber.org/zap"
)

// IExecutionService 执行记录服务接口
type IExecutionService interface {
	// 执行记录管理
	GetExecution(ctx context.Context, id string) (*entity.TaskExecution, error)
	ListExecutions(ctx context.Context, filter repository.ExecutionFilter) ([]*entity.TaskExecution, int64, error)

	// 执行回调
	ProcessCallback(ctx context.Context, executionID string, status entity.ExecutionStatus, result map[string]any, logs string) error

	// 执行控制
	StopExecution(ctx context.Context, executionID string) error

	// 统计
	GetExecutionStats(ctx context.Context, filter repository.ExecutionStatsFilter) (*repository.ExecutionStats, error)
}

type ExecutionService struct {
	executionRepo repository.ExecutionRepository
	executorRepo  repository.ExecutorRepository
	emitter       IEmitter
	logger        *zap.Logger
}

// NewExecutionService 创建执行记录服务
func NewExecutionService(
	executionRepo repository.ExecutionRepository,
	executorRepo repository.ExecutorRepository,
	emitter IEmitter,
	logger *zap.Logger,
) IExecutionService {
	return &ExecutionService{
		executionRepo: executionRepo,
		executorRepo:  executorRepo,
		emitter:       emitter,
		logger:        logger,
	}
}

func (s *ExecutionService) GetExecution(ctx context.Context, id string) (*entity.TaskExecution, error) {
	return s.executionRepo.GetByID(ctx, id)
}

func (s *ExecutionService) ListExecutions(ctx context.Context, filter repository.ExecutionFilter) ([]*entity.TaskExecution, int64, error) {
	return s.executionRepo.List(ctx, filter)
}

func (s *ExecutionService) ProcessCallback(ctx context.Context, executionID string, status entity.ExecutionStatus, result map[string]any, logs string) error {
	// 加载执行记录
	execution, err := s.executionRepo.GetByID(ctx, executionID)
	if err != nil {
		return err
	}

	// 更新执行状态
	if err := execution.Complete(status, result, logs); err != nil {
		return domainError.NewBusinessError("INVALID_CALLBACK", "无效的回调状态", err)
	}

	if err := s.executionRepo.Update(ctx, execution); err != nil {
		return err
	}

	s.logger.Info("执行回调处理完成",
		zap.String("execution_id", executionID),
		zap.String("status", string(status)))

	// 取消执行超时定时器
	if err := s.emitter.CancelExecutionTimer(executionID); err != nil {
		s.logger.Warn("取消执行定时器失败",
			zap.String("execution_id", executionID),
			zap.Error(err))
	}

	return nil
}

func (s *ExecutionService) StopExecution(ctx context.Context, executionID string) error {
	// 获取执行记录
	execution, err := s.executionRepo.GetByID(ctx, executionID)
	if err != nil {
		return err
	}

	if execution.Status != entity.ExecutionStatusRunning {
		return domainError.ErrExecutionNotRunning
	}

	// 取消执行
	if err := execution.Cancel(); err != nil {
		return err
	}

	// 调用执行器的停止接口
	if execution.ExecutorID != nil && execution.Executor != nil {
		if err := s.callExecutorStop(execution.Executor, executionID); err != nil {
			s.logger.Error("调用执行器停止接口失败",
				zap.String("execution_id", executionID),
				zap.String("executor_id", *execution.ExecutorID),
				zap.Error(err))
			// 不影响本地状态更新
		}
	}

	// 更新执行记录状态
	if err := s.executionRepo.Update(ctx, execution); err != nil {
		return err
	}

	return nil
}

func (s *ExecutionService) callExecutorStop(executor *entity.Executor, executionID string) error {
	stopReq := map[string]string{
		"execution_id": executionID,
	}

	jsonData, err := json.Marshal(stopReq)
	if err != nil {
		return fmt.Errorf("序列化停止请求失败: %w", err)
	}

	resp, err := http.Post(executor.GetStopURL(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("调用执行器停止接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return fmt.Errorf("执行器停止接口返回错误: 状态码=%d, 错误=%v", resp.StatusCode, errorResp)
	}

	return nil
}

func (s *ExecutionService) GetExecutionStats(ctx context.Context, filter repository.ExecutionStatsFilter) (*repository.ExecutionStats, error) {
	return s.executionRepo.GetStats(ctx, filter)
}
