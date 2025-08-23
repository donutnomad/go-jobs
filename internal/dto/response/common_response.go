package response

import (
	"time"

	"github.com/gin-gonic/gin"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

// SchedulerStatsResponse 调度器统计响应
type SchedulerStatsResponse struct {
	Instances []SchedulerInstanceResponse `json:"instances"`
	Time      time.Time                   `json:"time"`
}

// SchedulerInstanceResponse 调度器实例响应
type SchedulerInstanceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	IsLeader  bool      `json:"is_leader"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MessageResponse 简单消息响应
type MessageResponse struct {
	Message string `json:"message"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// ToGinH 转换为Gin H格式（用于健康检查等简单响应）
func (h *HealthResponse) ToGinH() gin.H {
	return gin.H{
		"status": h.Status,
		"time":   h.Time,
	}
}
