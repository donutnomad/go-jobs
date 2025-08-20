package api

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ErrorResponse 统一错误响应格式
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	mu              sync.RWMutex
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	state           string // "closed", "open", "half-open"
	threshold       int
	timeout         time.Duration
	resetTimeout    time.Duration
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        "closed",
		threshold:    threshold,
		timeout:      timeout,
		resetTimeout: 30 * time.Second,
	}
}

// Call 执行函数调用
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// 检查熔断器状态
	switch cb.state {
	case "open":
		// 检查是否可以进入半开状态
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = "half-open"
			cb.failureCount = 0
			cb.successCount = 0
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// 执行函数
	err := fn()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		// 检查是否需要打开熔断器
		if cb.failureCount >= cb.threshold {
			cb.state = "open"
		}
		return err
	}

	// 成功执行
	if cb.state == "half-open" {
		cb.successCount++
		// 半开状态下，连续成功后关闭熔断器
		if cb.successCount >= 3 {
			cb.state = "closed"
			cb.failureCount = 0
		}
	}

	return nil
}

// ErrorHandlingMiddleware 统一错误处理中间件
func ErrorHandlingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method))

				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Code:    "INTERNAL_ERROR",
					Message: "An internal error occurred",
				})
				c.Abort()
			}
		}()

		c.Next()

		// 处理错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			logger.Error("request error",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))

			// 根据错误类型返回适当的响应
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				c.JSON(http.StatusNotFound, ErrorResponse{
					Code:    "NOT_FOUND",
					Message: "Resource not found",
				})
			case errors.Is(err, gorm.ErrDuplicatedKey):
				c.JSON(http.StatusConflict, ErrorResponse{
					Code:    "DUPLICATE",
					Message: "Resource already exists",
				})
			default:
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Code:    "INTERNAL_ERROR",
					Message: "An error occurred while processing your request",
					Details: err.Error(),
				})
			}
		}
	}
}

// NetworkPartitionDetector 网络分区检测中间件
func NetworkPartitionDetector(storage interface{ Ping() error }, logger *zap.Logger) gin.HandlerFunc {
	var consecutiveFailures int
	const maxFailures = 3

	return func(c *gin.Context) {
		// 定期检查数据库连接
		if err := storage.Ping(); err != nil {
			consecutiveFailures++
			logger.Warn("database connection check failed",
				zap.Error(err),
				zap.Int("consecutive_failures", consecutiveFailures))

			if consecutiveFailures >= maxFailures {
				logger.Error("possible network partition detected")
				c.JSON(http.StatusServiceUnavailable, ErrorResponse{
					Code:    "SERVICE_UNAVAILABLE",
					Message: "Service is temporarily unavailable",
				})
				c.Abort()
				return
			}
		} else {
			consecutiveFailures = 0
		}

		c.Next()
	}
}

// RetryMiddleware 重试中间件
func RetryMiddleware(maxRetries int, backoff time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var lastErr error

		for i := 0; i <= maxRetries; i++ {
			if i > 0 {
				time.Sleep(backoff * time.Duration(i))
			}

			// 创建一个副本来重试
			copyContext := *c
			copyContext.Writer = &responseWriter{
				ResponseWriter: c.Writer,
				written:        false,
			}

			copyContext.Next()

			if !copyContext.IsAborted() && copyContext.Writer.Status() < 500 {
				// 成功或客户端错误，不需要重试
				return
			}

			lastErr = copyContext.Errors.Last()
		}

		if lastErr != nil {
			c.Error(lastErr)
		}
	}
}

// responseWriter 包装响应写入器
type responseWriter struct {
	gin.ResponseWriter
	written bool
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if !w.written {
		w.written = true
		return w.ResponseWriter.Write(data)
	}
	return len(data), nil
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.written {
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}
