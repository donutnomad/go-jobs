package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/app/service"
)

// Router API路由器
type Router struct {
	taskHandler      *TaskHandler
	executionHandler *ExecutionHandler
	// TODO: 添加其他handler
}

// NewRouter 创建API路由器
func NewRouter(taskService *service.TaskService, executionService *service.ExecutionService) *Router {
	return &Router{
		taskHandler:      NewTaskHandler(taskService),
		executionHandler: NewExecutionHandler(executionService, executionService.ExecutionQuery),
	}
}

// SetupRoutes 设置路由
func (r *Router) SetupRoutes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	// CORS配置
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	engine.Use(cors.New(config))

	// API v1路由组
	v1 := engine.Group("/api/v1")
	{
		// 任务路由
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", r.taskHandler.CreateTask)
			tasks.GET("", r.taskHandler.ListTasks)
			tasks.GET("/:id", r.taskHandler.GetTask)
			tasks.GET("/:id/detail", r.taskHandler.GetTaskDetail)
			tasks.PUT("/:id", r.taskHandler.UpdateTask)
			tasks.POST("/:id/pause", r.taskHandler.PauseTask)
			tasks.POST("/:id/resume", r.taskHandler.ResumeTask)
			tasks.DELETE("/:id", r.taskHandler.DeleteTask)
		}

		// 执行路由
		executions := v1.Group("/executions")
		{
			executions.GET("", r.executionHandler.ListExecutions)
			executions.GET("/:id", r.executionHandler.GetExecution)
			executions.GET("/statistics", r.executionHandler.GetExecutionStatistics)
			executions.GET("/running", r.executionHandler.GetRunningExecutions)
			executions.GET("/tasks/:task_id/history", r.executionHandler.GetTaskExecutionHistory)
			executions.GET("/failure-analysis", r.executionHandler.GetFailureAnalysis)
			executions.GET("/performance-analysis", r.executionHandler.GetPerformanceAnalysis)
		}

		// TODO: 添加其他路由
		// executors := v1.Group("/executors")
		// scheduler := v1.Group("/scheduler")
	}

	// 健康检查
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return engine
}
