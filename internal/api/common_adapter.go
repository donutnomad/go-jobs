package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/handler"
	"github.com/jobs/scheduler/internal/models"
)

// CommonAPIAdapter 通用API适配器，将新架构适配到现有的生成代码系统
type CommonAPIAdapter struct {
	handler handler.ICommonHandler
}

// NewCommonAPIAdapter 创建通用API适配器
func NewCommonAPIAdapter(handler handler.ICommonHandler) ICommonAPI {
	return &CommonAPIAdapter{handler: handler}
}

// 实现原有的ICommonAPI接口，保持完全兼容

func (a *CommonAPIAdapter) HealthCheck(ctx *gin.Context) (gin.H, error) {
	return a.handler.HealthCheck(ctx)
}

func (a *CommonAPIAdapter) SchedulerStats(ctx *gin.Context) (SchedulerStatsResp, error) {
	resp, err := a.handler.SchedulerStats(ctx)
	if err != nil {
		return SchedulerStatsResp{}, err
	}

	// 转换为原有格式
	instances := make([]models.SchedulerInstance, len(resp.Instances))
	for i, instance := range resp.Instances {
		instances[i] = models.SchedulerInstance{
			ID:         instance.ID,
			InstanceID: instance.Name,     // 使用Name作为InstanceID
			Host:       instance.Host,     // 保留Host字段
			Port:       instance.Port,     // 保留Port字段
			IsLeader:   instance.IsLeader, // 保留IsLeader字段
			CreatedAt:  instance.CreatedAt,
			UpdatedAt:  instance.UpdatedAt,
		}
	}

	return SchedulerStatsResp{
		Instances: instances,
		Time:      resp.Time,
	}, nil
}
