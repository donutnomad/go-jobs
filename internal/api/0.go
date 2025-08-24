package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

var Provider = wire.NewSet(
	NewExecutionAPI,
	NewExecutorAPI,
	NewTaskAPI,
	NewCommonAPI,
	NewServer,
)

//go:generate go tool github.com/donutnomad/gotoolkit/swagGen -path . -out 0api_generated.go

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
