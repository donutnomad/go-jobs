package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/jobs/scheduler/internal/api/handler"
	"github.com/jobs/scheduler/internal/dto/mapper"
	"github.com/jobs/scheduler/internal/infrastructure/repository"
	"github.com/jobs/scheduler/internal/service"
)

//go:generate go tool github.com/donutnomad/gotoolkit/swagGen -path . -out 0api_generated.go

var Provider = wire.NewSet(
	// 新架构的Repository层
	repository.NewTaskRepository,
	repository.NewExecutorRepository,
	repository.NewExecutionRepository,

	// 新架构的Service层
	service.NewTaskService,
	service.NewExecutorService,
	service.NewExecutionService,

	// 新架构的Mapper层
	mapper.NewTaskMapper,
	mapper.NewExecutorMapper,
	mapper.NewExecutionMapper,

	// 新架构的Handler层
	handler.NewTaskHandler,
	handler.NewExecutorHandler,
	handler.NewExecutionHandler,
	handler.NewCommonHandler,

	// 适配器层（保持与生成代码兼容）
	NewTaskAPIAdapter,
	NewExecutorAPIAdapter,
	NewExecutionAPIAdapter,
	NewCommonAPIAdapter,

	// 生成的Wrap层
	NewTaskAPIWrap,
	NewExecutorAPIWrap,
	NewExecutionAPIWrap,
	NewCommonAPIWrap,
)

func onGinBind(c *gin.Context, val any, typ string) bool {
	switch typ {
	case "JSON":
		if err := c.ShouldBindJSON(val); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return false
		}
	case "FORM":
		if err := c.ShouldBind(val); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return false
		}
	case "QUERY":
		if err := c.ShouldBindQuery(val); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return false
		}
	default:
		if err := c.ShouldBind(val); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return false
		}
	}
	return true
}

func onGinResponse[T any](c *gin.Context, data any, err error) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func onGinBindErr(c *gin.Context, err error) {
	c.JSON(500, gin.H{"error": err.Error()})
}
