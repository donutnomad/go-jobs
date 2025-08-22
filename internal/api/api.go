package api

import (
	"time"

	"github.com/gin-gonic/gin"
	models "github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
)

type ICommonAPI interface {
	// HealthCheck 健康检查
	// 检查服务是否健康
	// @GET(api/v1/health)
	HealthCheck(ctx *gin.Context) (gin.H, error)

	// SchedulerStats 获取执行统计
	// 获取指定任务的执行统计
	// @GET(api/v1/scheduler/stats)
	SchedulerStats(ctx *gin.Context) (SchedulerStatsResp, error)
}

var _ ICommonAPI = (*CommonAPI)(nil)

type CommonAPI struct {
	storage *orm.Storage
}

func NewCommonAPI(storage *orm.Storage) *CommonAPI {
	return &CommonAPI{
		storage: storage,
	}
}

type SchedulerStatsResp struct {
	Instances []models.SchedulerInstance `json:"instances"`
	Time      time.Time                  `json:"time"`
}

func (c *CommonAPI) HealthCheck(ctx *gin.Context) (gin.H, error) {
	if err := c.storage.Ping(); err != nil {
		return gin.H{}, err
	}

	return gin.H{
		"status": "healthy",
		"time":   time.Now(),
	}, nil
}

func (c *CommonAPI) SchedulerStats(ctx *gin.Context) (SchedulerStatsResp, error) {
	var instances []models.SchedulerInstance
	if err := c.storage.DB().Find(&instances).Error; err != nil {
		return SchedulerStatsResp{}, err
	}

	return SchedulerStatsResp{
		Instances: instances,
		Time:      time.Now(),
	}, nil
}
