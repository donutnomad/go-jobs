package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/middleware"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/infra/persistence/executionrepo"
	"github.com/jobs/scheduler/internal/infra/persistence/executorrepo"
	"github.com/jobs/scheduler/internal/infra/persistence/taskrepo"
	"github.com/jobs/scheduler/internal/orm"
	"go.uber.org/zap"
)

type Server struct {
	router *gin.Engine
}

func NewServer(
	storage *orm.Storage,
	emitter IEmitter,
	logger *zap.Logger,
) *Server {
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(middleware.ErrorHandlingMiddleware(logger))
	g.Use(middleware.Cors())
	db := storage.DB()

	executorRepo := executorrepo.NewMysqlRepositoryImpl(db)
	executionRepo := executionrepo.NewMysqlRepositoryImpl(db)
	taskRepo := taskrepo.NewMysqlRepositoryImpl(db)

	taskUsecase := task.NewUsecase(taskRepo)

	// 绑定路由（使用适配器保持兼容性）
	NewTaskAPIWrap(NewTaskAPI(db, emitter, taskUsecase, taskRepo)).BindAll(g)
	NewExecutorAPIWrap(NewExecutorAPI(db, logger, executorRepo, taskRepo)).BindAll(g)
	NewExecutionAPIWrap(NewExecutionAPI(db, logger, emitter, executionRepo, taskRepo, executorRepo)).BindAll(g)
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
