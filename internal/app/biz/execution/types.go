package execution

import (
	"fmt"
	"time"

	"github.com/jobs/scheduler/internal/app/types"
)

// ExecutionStatus 执行状态
type ExecutionStatus int

const (
	ExecutionStatusPending ExecutionStatus = iota + 1
	ExecutionStatusRunning
	ExecutionStatusSuccess
	ExecutionStatusFailed
	ExecutionStatusTimeout
	ExecutionStatusSkipped
	ExecutionStatusCancelled
)

func (s ExecutionStatus) String() string {
	switch s {
	case ExecutionStatusPending:
		return "pending"
	case ExecutionStatusRunning:
		return "running"
	case ExecutionStatusSuccess:
		return "success"
	case ExecutionStatusFailed:
		return "failed"
	case ExecutionStatusTimeout:
		return "timeout"
	case ExecutionStatusSkipped:
		return "skipped"
	case ExecutionStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// IsTerminal 判断是否为终止状态
func (s ExecutionStatus) IsTerminal() bool {
	switch s {
	case ExecutionStatusSuccess, ExecutionStatusFailed,
		ExecutionStatusTimeout, ExecutionStatusSkipped, ExecutionStatusCancelled:
		return true
	default:
		return false
	}
}

// IsRunning 判断是否正在运行
func (s ExecutionStatus) IsRunning() bool {
	return s == ExecutionStatusRunning
}

// IsSuccessful 判断是否执行成功
func (s ExecutionStatus) IsSuccessful() bool {
	return s == ExecutionStatusSuccess
}

// IsFailed 判断是否执行失败
func (s ExecutionStatus) IsFailed() bool {
	switch s {
	case ExecutionStatusFailed, ExecutionStatusTimeout, ExecutionStatusCancelled:
		return true
	default:
		return false
	}
}

// CanBeRetried 判断是否可以重试
func (s ExecutionStatus) CanBeRetried() bool {
	switch s {
	case ExecutionStatusFailed, ExecutionStatusTimeout:
		return true
	default:
		return false
	}
}

// CanBeStopped 判断是否可以被停止
func (s ExecutionStatus) CanBeStopped() bool {
	switch s {
	case ExecutionStatusPending, ExecutionStatusRunning:
		return true
	default:
		return false
	}
}

// RetryPolicy 重试策略值对象
type RetryPolicy struct {
	MaxRetries  int           `json:"max_retries"`
	BackoffBase time.Duration `json:"backoff_base"`
	MaxBackoff  time.Duration `json:"max_backoff"`
}

// NewRetryPolicy 创建重试策略
func NewRetryPolicy(maxRetries int) RetryPolicy {
	return RetryPolicy{
		MaxRetries:  maxRetries,
		BackoffBase: 1 * time.Second,
		MaxBackoff:  30 * time.Second,
	}
}

// CalculateBackoff 计算退避时间
func (p RetryPolicy) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// 指数退避：1s, 2s, 4s, 8s...
	backoff := time.Duration(1<<uint(attempt-1)) * p.BackoffBase
	if backoff > p.MaxBackoff {
		backoff = p.MaxBackoff
	}
	return backoff
}

// ShouldRetry 判断是否应该重试
func (p RetryPolicy) ShouldRetry(currentAttempt int) bool {
	return currentAttempt < p.MaxRetries
}

// ExecutionContext 执行上下文值对象
type ExecutionContext struct {
	TaskID        types.ID      `json:"task_id"`
	ExecutorID    *types.ID     `json:"executor_id"`
	Parameters    types.JSONMap `json:"parameters"`
	ScheduledTime time.Time     `json:"scheduled_time"`
	CallbackURL   string        `json:"callback_url"`
}

// NewExecutionContext 创建执行上下文
func NewExecutionContext(taskID types.ID, parameters types.JSONMap, scheduledTime time.Time) ExecutionContext {
	return ExecutionContext{
		TaskID:        taskID,
		Parameters:    parameters,
		ScheduledTime: scheduledTime,
	}
}

// WithExecutor 设置执行器
func (c ExecutionContext) WithExecutor(executorID types.ID) ExecutionContext {
	c.ExecutorID = &executorID
	return c
}

// WithCallbackURL 设置回调URL
func (c ExecutionContext) WithCallbackURL(callbackURL string) ExecutionContext {
	c.CallbackURL = callbackURL
	return c
}

// HasExecutor 判断是否有执行器
func (c ExecutionContext) HasExecutor() bool {
	return c.ExecutorID != nil && !c.ExecutorID.IsZero()
}

// GetExecutorID 获取执行器ID
func (c ExecutionContext) GetExecutorID() types.ID {
	if c.ExecutorID == nil {
		return ""
	}
	return *c.ExecutorID
}

// IsScheduled 判断是否已调度
func (c ExecutionContext) IsScheduled() bool {
	return !c.ScheduledTime.IsZero()
}

// ExecutionResult 执行结果值对象
type ExecutionResult struct {
	Status    ExecutionStatus `json:"status"`
	Result    types.JSONMap   `json:"result"`
	Logs      string          `json:"logs"`
	Error     string          `json:"error"`
	StartTime *time.Time      `json:"start_time"`
	EndTime   *time.Time      `json:"end_time"`
	Duration  *time.Duration  `json:"duration"`
}

// NewExecutionResult 创建执行结果
func NewExecutionResult(status ExecutionStatus) ExecutionResult {
	return ExecutionResult{
		Status: status,
		Result: make(types.JSONMap),
	}
}

// WithResult 设置结果数据
func (r ExecutionResult) WithResult(result types.JSONMap) ExecutionResult {
	r.Result = result
	return r
}

// WithLogs 设置日志
func (r ExecutionResult) WithLogs(logs string) ExecutionResult {
	r.Logs = logs
	return r
}

// WithError 设置错误
func (r ExecutionResult) WithError(err string) ExecutionResult {
	r.Error = err
	return r
}

// WithTiming 设置时间信息
func (r ExecutionResult) WithTiming(startTime, endTime *time.Time) ExecutionResult {
	r.StartTime = startTime
	r.EndTime = endTime

	if startTime != nil && endTime != nil {
		duration := endTime.Sub(*startTime)
		r.Duration = &duration
	}

	return r
}

// IsSuccessful 判断是否成功
func (r ExecutionResult) IsSuccessful() bool {
	return r.Status.IsSuccessful()
}

// HasError 判断是否有错误
func (r ExecutionResult) HasError() bool {
	return r.Error != ""
}

// GetDurationSeconds 获取执行时长（秒）
func (r ExecutionResult) GetDurationSeconds() float64 {
	if r.Duration == nil {
		return 0
	}
	return r.Duration.Seconds()
}

// CallbackInfo 回调信息值对象
type CallbackInfo struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Timeout time.Duration     `json:"timeout"`
}

// NewCallbackInfo 创建回调信息
func NewCallbackInfo(url string) CallbackInfo {
	return CallbackInfo{
		URL:     url,
		Headers: make(map[string]string),
		Timeout: 30 * time.Second,
	}
}

// WithHeader 添加头部
func (c CallbackInfo) WithHeader(key, value string) CallbackInfo {
	if c.Headers == nil {
		c.Headers = make(map[string]string)
	}
	c.Headers[key] = value
	return c
}

// WithTimeout 设置超时时间
func (c CallbackInfo) WithTimeout(timeout time.Duration) CallbackInfo {
	c.Timeout = timeout
	return c
}

// IsValid 验证回调信息是否有效
func (c CallbackInfo) IsValid() bool {
	return c.URL != "" && c.Timeout > 0
}

// TriggerExecutionRequest 触发执行请求
type TriggerExecutionRequest struct {
	TaskID     types.ID      `json:"task_id" binding:"required"`
	Parameters types.JSONMap `json:"parameters"`
}

// Validate 验证请求
func (r TriggerExecutionRequest) Validate() error {
	if r.TaskID.IsZero() {
		return fmt.Errorf("task ID is required")
	}
	return nil
}

// ExecutionCallbackRequest 执行回调请求
type ExecutionCallbackRequest struct {
	Status  ExecutionStatus `json:"status" binding:"required"`
	Result  types.JSONMap   `json:"result"`
	Logs    string          `json:"logs"`
	Error   string          `json:"error"`
	EndTime *time.Time      `json:"end_time"`
}

// Validate 验证回调请求
func (r ExecutionCallbackRequest) Validate() error {
	if r.Status == 0 {
		return fmt.Errorf("status is required")
	}
	return nil
}

// ToExecutionResult 转换为执行结果
func (r ExecutionCallbackRequest) ToExecutionResult() ExecutionResult {
	result := NewExecutionResult(r.Status)
	if r.Result != nil {
		result = result.WithResult(r.Result)
	}
	if r.Logs != "" {
		result = result.WithLogs(r.Logs)
	}
	if r.Error != "" {
		result = result.WithError(r.Error)
	}
	if r.EndTime != nil {
		result = result.WithTiming(nil, r.EndTime)
	}
	return result
}

// ExecutionFilter 执行过滤器
type ExecutionFilter struct {
	types.Filter
	TaskID     *types.ID  `json:"task_id"`
	ExecutorID *types.ID  `json:"executor_id"`
	Status     []string   `json:"status"`
	StartTime  *time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
}

// NewExecutionFilter 创建执行过滤器
func NewExecutionFilter() ExecutionFilter {
	return ExecutionFilter{
		Filter: types.NewFilter(),
	}
}

// StopExecutionRequest 停止执行请求
type StopExecutionRequest struct {
	Reason string `json:"reason"`
}
