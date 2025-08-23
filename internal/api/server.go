package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/api/handler"
	"github.com/jobs/scheduler/internal/api/middleware"
	"github.com/jobs/scheduler/internal/dto/mapper"
	"github.com/jobs/scheduler/internal/infrastructure/repository"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/internal/service"
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

	// 创建Repository层
	taskRepo := repository.NewTaskRepository(db)
	executorRepo := repository.NewExecutorRepository(db)
	executionRepo := repository.NewExecutionRepository(db)

	// 创建Service层
	taskService := service.NewTaskService(taskRepo, executorRepo, executionRepo, emitter)
	executorService := service.NewExecutorService(executorRepo, taskRepo, logger)
	executionService := service.NewExecutionService(executionRepo, executorRepo, emitter, logger)

	// 创建Mapper层
	taskMapper := mapper.NewTaskMapper()
	executorMapper := mapper.NewExecutorMapper()
	executionMapper := mapper.NewExecutionMapper(taskMapper, executorMapper)

	// 创建Handler层
	taskHandler := handler.NewTaskHandler(taskService, executorService, taskMapper, executionMapper, executorMapper)
	executorHandler := handler.NewExecutorHandler(executorService, executorMapper)
	executionHandler := handler.NewExecutionHandler(executionService, executionMapper)
	commonHandler := handler.NewCommonHandler(db)

	// 创建适配器以保持与现有生成代码的兼容性
	taskAdapter := NewTaskAPIAdapter(taskHandler)
	executorAdapter := NewExecutorAPIAdapter(executorHandler)
	executionAdapter := NewExecutionAPIAdapter(executionHandler)
	commonAdapter := NewCommonAPIAdapter(commonHandler)

	// 绑定路由（使用适配器保持兼容性）
	NewTaskAPIWrap(taskAdapter).BindAll(g)
	NewExecutorAPIWrap(executorAdapter).BindAll(g)
	NewExecutionAPIWrap(executionAdapter).BindAll(g)
	NewCommonAPIWrap(commonAdapter).BindAll(g)

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
