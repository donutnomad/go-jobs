package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/models"
	"gorm.io/gorm"
)

type ICommonAPI interface {
	// HealthCheck 健康检查
	// 检查服务是否健康
	// @GET(api/v1/health)
	HealthCheck(ctx *gin.Context) (gin.H, error)

	// SchedulerStats 获取执行统计
	// 获取指定任务的执行统计
	// @GET(api/v1/scheduler/status)
	SchedulerStats(ctx *gin.Context) (SchedulerStatsResp, error)
}

type CommonAPI struct {
	db *gorm.DB
}

func NewCommonAPI(db *gorm.DB) ICommonAPI {
	return &CommonAPI{
		db: db,
	}
}

func (c *CommonAPI) HealthCheck(ctx *gin.Context) (gin.H, error) {
	return gin.H{
		"status": "healthy",
		"time":   time.Now(),
	}, nil
}

func (c *CommonAPI) SchedulerStats(ctx *gin.Context) (SchedulerStatsResp, error) {
	var instances []models.SchedulerInstance
	if err := c.db.WithContext(ctx).Find(&instances).Error; err != nil {
		return SchedulerStatsResp{}, err
	}
	return SchedulerStatsResp{
		Instances: instances,
		Time:      time.Now(),
	}, nil
}
