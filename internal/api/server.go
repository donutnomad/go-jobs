package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/middleware"
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
	taskRunner *scheduler.TaskRunner,
	logger *zap.Logger,
) *Server {
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(middleware.ErrorHandlingMiddleware(logger))
	g.Use(middleware.Cors())
	db := storage.DB()

	var emitter = NewEventBus(scheduler, taskRunner)

	NewTaskAPIWrap(NewTaskAPI(db, emitter)).BindAll(g)
	NewExecutorAPIWrap(NewExecutorAPI(db, logger)).BindAll(g)
	NewExecutionAPIWrap(NewExecutionAPI(db, logger, emitter)).BindAll(g)
	NewCommonAPIWrap(NewCommonAPI(db)).BindAll(g)

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
