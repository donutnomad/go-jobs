package middleware

import (
	"errors"
	"net/http"

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
