package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/middleware"
	"github.com/jobs/scheduler/internal/executor"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/internal/scheduler"
	"go.uber.org/zap"
)

type Server struct {
	router *gin.Engine
}

func NewServer(
	storage *orm.Storage,
	scheduler *scheduler.Scheduler,
	executorManager *executor.Manager,
	taskRunner *scheduler.TaskRunner,
	logger *zap.Logger,
) *Server {
	s := &Server{}

	s.router = gin.New()
	s.router.Use(gin.Recovery())
	s.router.Use(middleware.ErrorHandlingMiddleware(logger))
	s.router.Use(middleware.Cors())

	NewTaskAPIWrap(NewTaskAPI(storage, scheduler)).BindAll(s.router)
	NewExecutorAPIWrap(NewExecutorAPI(storage, executorManager, logger)).BindAll(s.router)
	NewExecutionAPIWrap(NewExecutionAPI(storage, taskRunner, logger)).BindAll(s.router)
	NewCommonAPIWrap(NewCommonAPI(storage)).BindAll(s.router)

	return s
}

func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
