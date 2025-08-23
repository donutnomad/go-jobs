package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/biz/scheduler_instance"
	"github.com/jobs/scheduler/internal/infra/persistence/schedulerinstancerepo"
	"github.com/samber/lo"
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
	db                    *gorm.DB
	schedulerInstanceRepo scheduler_instance.Repo
}

func NewCommonAPI(db *gorm.DB) ICommonAPI {
	return &CommonAPI{
		db:                    db,
		schedulerInstanceRepo: schedulerinstancerepo.NewMysqlRepositoryImpl(db),
	}
}

func (c *CommonAPI) HealthCheck(ctx *gin.Context) (gin.H, error) {
	return gin.H{
		"status": "healthy",
		"time":   time.Now(),
	}, nil
}

func (c *CommonAPI) SchedulerStats(ctx *gin.Context) (SchedulerStatsResp, error) {
	instances, err := c.schedulerInstanceRepo.List(ctx)
	if err != nil {
		return SchedulerStatsResp{}, err
	}
	return SchedulerStatsResp{
		Instances: lo.Map(instances, func(instance *scheduler_instance.SchedulerInstance, _ int) SchedulerInstanceResp {
			return SchedulerInstanceResp{
				ID:         instance.ID,
				InstanceID: instance.InstanceID,
				Host:       instance.Host,
				Port:       instance.Port,
				IsLeader:   instance.IsLeader,
				CreatedAt:  instance.CreatedAt,
				UpdatedAt:  instance.UpdatedAt,
			}
		}),
		Time: time.Now(),
	}, nil
}
