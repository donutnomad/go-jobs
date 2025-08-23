package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/dto/response"
	"github.com/jobs/scheduler/internal/models"
	"gorm.io/gorm"
)

// ICommonHandler 通用处理器接口（与原有ICommonAPI保持兼容）
type ICommonHandler interface {
	// HealthCheck 健康检查
	// 检查服务是否健康
	// @GET(api/v1/health)
	HealthCheck(ctx *gin.Context) (gin.H, error)

	// SchedulerStats 获取执行统计
	// 获取指定任务的执行统计
	// @GET(api/v1/scheduler/stats)
	SchedulerStats(ctx *gin.Context) (response.SchedulerStatsResponse, error)
}

type CommonHandler struct {
	db *gorm.DB
}

// NewCommonHandler 创建通用处理器
func NewCommonHandler(db *gorm.DB) ICommonHandler {
	return &CommonHandler{
		db: db,
	}
}

func (h *CommonHandler) HealthCheck(ctx *gin.Context) (gin.H, error) {
	return gin.H{
		"status": "healthy",
		"time":   time.Now(),
	}, nil
}

func (h *CommonHandler) SchedulerStats(ctx *gin.Context) (response.SchedulerStatsResponse, error) {
	var instances []models.SchedulerInstance
	if err := h.db.WithContext(ctx).Find(&instances).Error; err != nil {
		return response.SchedulerStatsResponse{}, err
	}

	// 转换为响应格式
	respInstances := make([]response.SchedulerInstanceResponse, len(instances))
	for i, instance := range instances {
		respInstances[i] = response.SchedulerInstanceResponse{
			ID:        instance.ID,
			Name:      instance.InstanceID, // 使用InstanceID作为Name
			Host:      instance.Host,       // 保留Host字段
			Port:      instance.Port,       // 保留Port字段
			IsLeader:  instance.IsLeader,   // 保留IsLeader字段
			Status:    "online",            // 默认状态
			CreatedAt: instance.CreatedAt,
			UpdatedAt: instance.UpdatedAt,
		}
	}

	return response.SchedulerStatsResponse{
		Instances: respInstances,
		Time:      time.Now(),
	}, nil
}
