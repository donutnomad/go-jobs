package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/middleware"
	"go.uber.org/zap"
)

type Server struct {
	router *gin.Engine
}

func NewServer(
	logger *zap.Logger,

	taskAPI ITaskAPI,
	executionAPI IExecutionAPI,
	executorAPI IExecutorAPI,
	commonAPI ICommonAPI,
) *Server {
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(middleware.ErrorHandlingMiddleware(logger))
	g.Use(middleware.Cors())

	// 绑定路由（使用适配器保持兼容性）
	NewTaskAPIWrap(taskAPI).BindAll(g)
	NewExecutorAPIWrap(executorAPI).BindAll(g)
	NewExecutionAPIWrap(executionAPI).BindAll(g)
	NewCommonAPIWrap(commonAPI).BindAll(g)

	return &Server{
		router: g,
	}
}

func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
