package error

import "errors"

// 领域层错误定义

var (
	// Task相关错误
	ErrTaskNotFound      = errors.New("任务未找到")
	ErrTaskAlreadyExists = errors.New("任务已存在")
	ErrTaskInvalidName   = errors.New("任务名称无效")
	ErrTaskInvalidCron   = errors.New("Cron表达式无效")
	ErrTaskAlreadyPaused = errors.New("任务已暂停")
	ErrTaskAlreadyActive = errors.New("任务已激活")
	ErrTaskDeleted       = errors.New("任务已删除")

	// Executor相关错误
	ErrExecutorNotFound      = errors.New("执行器未找到")
	ErrExecutorAlreadyExists = errors.New("执行器已存在")
	ErrExecutorOffline       = errors.New("执行器离线")
	ErrExecutorUnhealthy     = errors.New("执行器不健康")

	// Execution相关错误
	ErrExecutionNotFound         = errors.New("执行记录未找到")
	ErrExecutionInvalidStatus    = errors.New("执行状态无效")
	ErrExecutionAlreadyCompleted = errors.New("执行已完成")
	ErrExecutionNotRunning       = errors.New("执行未在运行")

	// Assignment相关错误
	ErrAssignmentNotFound      = errors.New("执行器分配未找到")
	ErrAssignmentAlreadyExists = errors.New("执行器分配已存在")

	// 通用错误
	ErrInvalidInput     = errors.New("输入参数无效")
	ErrInternalError    = errors.New("内部错误")
	ErrPermissionDenied = errors.New("权限被拒绝")
)

// DomainError 领域错误接口
type DomainError interface {
	error
	Code() string
	Message() string
}

// BusinessError 业务错误
type BusinessError struct {
	code    string
	message string
	cause   error
}

func NewBusinessError(code, message string, cause error) *BusinessError {
	return &BusinessError{
		code:    code,
		message: message,
		cause:   cause,
	}
}

func (e *BusinessError) Error() string {
	if e.cause != nil {
		return e.message + ": " + e.cause.Error()
	}
	return e.message
}

func (e *BusinessError) Code() string {
	return e.code
}

func (e *BusinessError) Message() string {
	return e.message
}

func (e *BusinessError) Unwrap() error {
	return e.cause
}
