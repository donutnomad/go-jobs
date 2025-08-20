package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/executor"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/scheduler"
	"github.com/jobs/scheduler/internal/storage"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Server REST API服务器
type Server struct {
	storage         *storage.Storage
	scheduler       *scheduler.Scheduler
	executorManager *executor.Manager
	taskRunner      *scheduler.TaskRunner
	logger          *zap.Logger
	router          *gin.Engine
}

// NewServer 创建API服务器
func NewServer(
	storage *storage.Storage,
	scheduler *scheduler.Scheduler,
	executorManager *executor.Manager,
	taskRunner *scheduler.TaskRunner,
	logger *zap.Logger,
) *Server {
	s := &Server{
		storage:         storage,
		scheduler:       scheduler,
		executorManager: executorManager,
		taskRunner:      taskRunner,
		logger:          logger,
	}

	s.setupRouter()
	return s
}

// setupRouter 设置路由
func (s *Server) setupRouter() {
	s.router = gin.New() // 使用 New 而不是 Default，避免重复的日志中间件

	// 添加错误处理和恢复中间件
	s.router.Use(gin.Recovery())
	s.router.Use(ErrorHandlingMiddleware(s.logger))

	// 配置 CORS 中间件 - 开发环境配置
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	corsConfig.ExposeHeaders = []string{"Content-Length", "Content-Type"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 12 * time.Hour

	s.router.Use(cors.New(corsConfig))

	// API路由组
	api := s.router.Group("/api/v1")
	{
		// 健康检查
		api.GET("/health", s.healthCheck)

		// 任务管理
		tasks := api.Group("/tasks")
		{
			tasks.GET("", s.listTasks)
			tasks.GET("/:id", s.getTask)
			tasks.POST("", s.createTask)
			tasks.PUT("/:id", s.updateTask)
			tasks.DELETE("/:id", s.deleteTask)
			tasks.POST("/:id/trigger", s.triggerTask)
			tasks.POST("/:id/pause", s.pauseTask)
			tasks.POST("/:id/resume", s.resumeTask)
			tasks.GET("/:id/executors", s.getTaskExecutors)
			tasks.POST("/:id/executors", s.assignExecutor)
			tasks.PUT("/:id/executors/:executor_id", s.updateExecutorAssignment)
			tasks.DELETE("/:id/executors/:executor_id", s.unassignExecutor)
			tasks.GET("/:id/stats", s.getTaskStats) // 新增：获取任务统计
		}

		// 执行器管理
		executors := api.Group("/executors")
		{
			executors.GET("", s.listExecutors)
			executors.GET("/:id", s.getExecutor)
			executors.POST("/register", s.registerExecutor)
			executors.PUT("/:id", s.updateExecutor)
			executors.PUT("/:id/status", s.updateExecutorStatus)
			executors.DELETE("/:id", s.deleteExecutor)
		}

		// 执行历史
		executions := api.Group("/executions")
		{
			executions.GET("", s.listExecutions)
			executions.GET("/stats", s.getExecutionStats)
			executions.GET("/:id", s.getExecution)
			executions.POST("/:id/callback", s.executionCallback)
			executions.POST("/:id/stop", s.stopExecution)
		}

		// 调度器状态
		api.GET("/scheduler/status", s.getSchedulerStatus)
	}
}

// Router 返回gin路由器
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Run 启动API服务器
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// healthCheck 健康检查
func (s *Server) healthCheck(c *gin.Context) {
	if err := s.storage.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now(),
	})
}

// listTasks 获取任务列表
func (s *Server) listTasks(c *gin.Context) {
	var tasks []models.Task
	query := s.storage.DB().Preload("TaskExecutors").Preload("TaskExecutors.Executor")

	// 支持状态过滤
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// getTask 获取任务详情
func (s *Server) getTask(c *gin.Context) {
	taskID := c.Param("id")

	var task models.Task
	if err := s.storage.DB().
		Preload("TaskExecutors").
		Preload("TaskExecutors.Executor").
		Where("id = ?", taskID).
		First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// createTask 创建任务
func (s *Server) createTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := models.Task{
		ID:                  generateID(),
		Name:                req.Name,
		CronExpression:      req.CronExpression,
		Parameters:          req.Parameters,
		ExecutionMode:       req.ExecutionMode,
		LoadBalanceStrategy: req.LoadBalanceStrategy,
		MaxRetry:            req.MaxRetry,
		TimeoutSeconds:      req.TimeoutSeconds,
		Status:              models.TaskStatusActive,
	}

	// 设置默认值
	if task.ExecutionMode == "" {
		task.ExecutionMode = models.ExecutionModeParallel
	}
	if task.LoadBalanceStrategy == "" {
		task.LoadBalanceStrategy = models.LoadBalanceRoundRobin
	}
	if task.MaxRetry == 0 {
		task.MaxRetry = 3
	}
	if task.TimeoutSeconds == 0 {
		task.TimeoutSeconds = 300
	}

	if err := s.storage.DB().Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// updateTask 更新任务
func (s *Server) updateTask(c *gin.Context) {
	taskID := c.Param("id")

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var task models.Task
	if err := s.storage.DB().Where("id = ?", taskID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// 更新字段
	if req.Name != "" {
		task.Name = req.Name
	}
	if req.CronExpression != "" {
		task.CronExpression = req.CronExpression
	}
	if req.Parameters != nil {
		task.Parameters = req.Parameters
	}
	if req.ExecutionMode != "" {
		task.ExecutionMode = req.ExecutionMode
	}
	if req.LoadBalanceStrategy != "" {
		task.LoadBalanceStrategy = req.LoadBalanceStrategy
	}
	if req.MaxRetry > 0 {
		task.MaxRetry = req.MaxRetry
	}
	if req.TimeoutSeconds > 0 {
		task.TimeoutSeconds = req.TimeoutSeconds
	}
	if req.Status != "" {
		task.Status = req.Status
	}

	if err := s.storage.DB().Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// deleteTask 删除任务
func (s *Server) deleteTask(c *gin.Context) {
	taskID := c.Param("id")

	// 软删除，将状态设置为deleted
	result := s.storage.DB().
		Model(&models.Task{}).
		Where("id = ?", taskID).
		Update("status", models.TaskStatusDeleted)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
}

// triggerTask 手动触发任务
func (s *Server) triggerTask(c *gin.Context) {
	taskID := c.Param("id")

	var req executor.TriggerTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	execution, err := s.scheduler.TriggerTask(c.Request.Context(), taskID, req.Parameters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// getTaskExecutors 获取任务的执行器列表
func (s *Server) getTaskExecutors(c *gin.Context) {
	taskID := c.Param("id")

	// 获取任务的执行器
	var taskExecutors []models.TaskExecutor
	if err := s.storage.DB().
		Preload("Executor").
		Where("task_id = ?", taskID).
		Find(&taskExecutors).
		Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get executors"})
		return
	}

	c.JSON(http.StatusOK, taskExecutors)
}

// assignExecutor 为任务分配执行器
func (s *Server) assignExecutor(c *gin.Context) {
	taskID := c.Param("id")

	var req AssignExecutorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证任务是否存在
	var task models.Task
	if err := s.storage.DB().Where("id = ?", taskID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// 验证执行器是否存在
	var executor models.Executor
	if err := s.storage.DB().Where("id = ?", req.ExecutorID).First(&executor).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "executor not found"})
		return
	}

	// 创建任务执行器关联
	taskExecutor := models.TaskExecutor{
		ID:         generateID(),
		TaskID:     taskID,
		ExecutorID: req.ExecutorID,
		Priority:   req.Priority,
		Weight:     req.Weight,
	}

	// 设置默认值
	if taskExecutor.Priority == 0 {
		taskExecutor.Priority = 1
	}
	if taskExecutor.Weight == 0 {
		taskExecutor.Weight = 1
	}

	if err := s.storage.DB().Create(&taskExecutor).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, taskExecutor)
}

// updateExecutorAssignment 更新任务执行器分配
func (s *Server) updateExecutorAssignment(c *gin.Context) {
	taskID := c.Param("id")
	executorID := c.Param("executor_id")

	var req UpdateExecutorAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找现有分配
	var taskExecutor models.TaskExecutor
	if err := s.storage.DB().Where("task_id = ? AND executor_id = ?", taskID, executorID).First(&taskExecutor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	// 更新分配
	taskExecutor.Priority = req.Priority
	taskExecutor.Weight = req.Weight

	if err := s.storage.DB().Save(&taskExecutor).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, taskExecutor)
}

// unassignExecutor 取消任务执行器分配
func (s *Server) unassignExecutor(c *gin.Context) {
	taskID := c.Param("id")
	executorID := c.Param("executor_id")

	result := s.storage.DB().
		Where("task_id = ? AND executor_id = ?", taskID, executorID).
		Delete(&models.TaskExecutor{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "executor unassigned"})
}

// listExecutors 获取执行器列表
func (s *Server) listExecutors(c *gin.Context) {
	includeTasks := c.Query("include_tasks") == "true"

	executors, err := s.executorManager.ListExecutors(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果需要包含任务信息，为每个执行器加载关联的任务
	if includeTasks {
		for _, executor := range executors {
			var taskExecutors []models.TaskExecutor
			err := s.storage.DB().
				Preload("Task").
				Where("executor_id = ?", executor.ID).
				Find(&taskExecutors).Error
			if err != nil {
				s.logger.Error("failed to load task executors",
					zap.String("executor_id", executor.ID),
					zap.Error(err))
				continue
			}
			executor.TaskExecutors = taskExecutors
		}
	}
	sort.Slice(executors, func(i, j int) bool {
		return executors[i].Status.ToInt() < executors[j].Status.ToInt()
	})

	c.JSON(http.StatusOK, executors)
}

// getExecutor 获取执行器详情
func (s *Server) getExecutor(c *gin.Context) {
	executorID := c.Param("id")

	executor, err := s.executorManager.GetExecutorByID(c.Request.Context(), executorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "executor not found"})
		return
	}

	c.JSON(http.StatusOK, executor)
}

// registerExecutor 注册执行器
func (s *Server) registerExecutor(c *gin.Context) {
	var req executor.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	executor, err := s.executorManager.RegisterExecutor(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executor)
}

// updateExecutor 更新执行器信息
func (s *Server) updateExecutor(c *gin.Context) {
	executorID := c.Param("id")

	var req struct {
		Name           string `json:"name"`
		BaseURL        string `json:"base_url"`
		HealthCheckURL string `json:"health_check_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找执行器
	var executor models.Executor
	if err := s.storage.DB().Where("id = ?", executorID).First(&executor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "executor not found"})
		return
	}

	// 更新字段
	if req.Name != "" {
		executor.Name = req.Name
	}
	if req.BaseURL != "" {
		executor.BaseURL = req.BaseURL
	}
	if req.HealthCheckURL != "" {
		executor.HealthCheckURL = req.HealthCheckURL
	}

	// 保存更新
	if err := s.storage.DB().Save(&executor).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executor)
}

// updateExecutorStatus 更新执行器状态
func (s *Server) updateExecutorStatus(c *gin.Context) {
	executorID := c.Param("id")

	var req executor.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.executorManager.UpdateExecutorStatus(c.Request.Context(), executorID, req.Status, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// deleteExecutor 删除执行器
func (s *Server) deleteExecutor(c *gin.Context) {
	executorID := c.Param("id")

	// 先删除关联的 task_executors 记录
	if err := s.storage.DB().Where("executor_id = ?", executorID).Delete(&models.TaskExecutor{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 删除执行器
	result := s.storage.DB().Where("id = ?", executorID).Delete(&models.Executor{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "executor not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "executor deleted"})
}

// listExecutions 获取执行历史列表
func (s *Server) listExecutions(c *gin.Context) {
	// 分页响应结构
	type PaginatedResponse struct {
		Data       []models.TaskExecution `json:"data"`
		Total      int64                  `json:"total"`
		Page       int                    `json:"page"`
		PageSize   int                    `json:"page_size"`
		TotalPages int                    `json:"total_pages"`
	}

	var executions []models.TaskExecution
	query := s.storage.DB().Model(&models.TaskExecution{})
	countQuery := s.storage.DB().Model(&models.TaskExecution{})

	// 支持任务ID过滤
	if taskID := c.Query("task_id"); taskID != "" {
		query = query.Where("task_id = ?", taskID)
		countQuery = countQuery.Where("task_id = ?", taskID)
	}

	// 支持状态过滤
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
		countQuery = countQuery.Where("status = ?", status)
	}

	// 支持时间范围过滤
	if start := c.Query("start_time"); start != "" {
		query = query.Where("scheduled_time >= ?", start)
		countQuery = countQuery.Where("scheduled_time >= ?", start)
	}
	if end := c.Query("end_time"); end != "" {
		query = query.Where("scheduled_time <= ?", end)
		countQuery = countQuery.Where("scheduled_time <= ?", end)
	}

	// 获取总数
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 分页参数
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 20 // 默认每页20条
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询数据
	query = query.Preload("Task").Preload("Executor")
	if err := query.Order("scheduled_time DESC").Limit(pageSize).Offset(offset).Find(&executions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	// 返回分页响应
	response := PaginatedResponse{
		Data:       executions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// getExecutionStats 获取执行统计
func (s *Server) getExecutionStats(c *gin.Context) {
	type ExecutionStats struct {
		Total   int64 `json:"total"`
		Success int64 `json:"success"`
		Failed  int64 `json:"failed"`
		Running int64 `json:"running"`
		Pending int64 `json:"pending"`
	}

	var stats ExecutionStats
	db := s.storage.DB()

	// 构建基础查询条件
	buildQuery := func() *gorm.DB {
		query := db.Model(&models.TaskExecution{})

		// 支持时间范围过滤
		if start := c.Query("start_time"); start != "" {
			query = query.Where("scheduled_time >= ?", start)
		}
		if end := c.Query("end_time"); end != "" {
			query = query.Where("scheduled_time <= ?", end)
		}

		// 支持任务ID过滤
		if taskID := c.Query("task_id"); taskID != "" {
			query = query.Where("task_id = ?", taskID)
		}

		return query
	}

	// 统计总数
	if err := buildQuery().Count(&stats.Total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 统计各状态数量
	buildQuery().Where("status = ?", models.ExecutionStatusSuccess).Count(&stats.Success)
	buildQuery().Where("status = ?", models.ExecutionStatusFailed).Count(&stats.Failed)
	buildQuery().Where("status = ?", models.ExecutionStatusRunning).Count(&stats.Running)
	buildQuery().Where("status = ?", models.ExecutionStatusPending).Count(&stats.Pending)

	c.JSON(http.StatusOK, stats)
}

// getExecution 获取执行详情
func (s *Server) getExecution(c *gin.Context) {
	executionID := c.Param("id")

	var execution models.TaskExecution
	if err := s.storage.DB().
		Preload("Task").
		Preload("Executor").
		Where("id = ?", executionID).
		First(&execution).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// executionCallback 执行回调
func (s *Server) executionCallback(c *gin.Context) {
	executionID := c.Param("id")

	var req executor.ExecutionCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.taskRunner.HandleCallback(c.Request.Context(), executionID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "callback processed"})
}

// stopExecution 停止正在执行的任务
func (s *Server) stopExecution(c *gin.Context) {
	executionID := c.Param("id")

	// 查找执行记录
	var execution models.TaskExecution
	if err := s.storage.DB().
		Preload("Executor").
		Where("id = ?", executionID).
		First(&execution).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		return
	}

	// 检查执行状态
	if execution.Status != models.ExecutionStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execution is not running"})
		return
	}

	// 调用执行器的停止接口
	if execution.Executor != nil {
		stopURL := fmt.Sprintf("%s/stop", execution.Executor.BaseURL)
		stopReq := map[string]string{
			"execution_id": executionID,
		}

		jsonData, err := json.Marshal(stopReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal stop request"})
			return
		}

		resp, err := http.Post(stopURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			s.logger.Error("failed to call executor stop endpoint",
				zap.String("execution_id", executionID),
				zap.String("executor_id", *execution.ExecutorID),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop execution on executor"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errorResp map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errorResp)
			s.logger.Error("executor stop endpoint returned error",
				zap.String("execution_id", executionID),
				zap.Int("status_code", resp.StatusCode),
				zap.Any("error", errorResp))
		}
	}

	// 更新执行状态为取消中
	execution.Status = models.ExecutionStatusCancelled
	if err := s.storage.DB().Save(&execution).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update execution status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "stop request sent to executor",
		"execution_id": executionID,
	})
}

// pauseTask 暂停任务调度
func (s *Server) pauseTask(c *gin.Context) {
	taskID := c.Param("id")

	// 查找任务
	var task models.Task
	if err := s.storage.DB().Where("id = ?", taskID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// 检查任务状态
	if task.Status == models.TaskStatusPaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task is already paused"})
		return
	}

	if task.Status == models.TaskStatusDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot pause deleted task"})
		return
	}

	// 更新任务状态为暂停
	if err := s.storage.DB().
		Model(&task).
		Where("id = ?", taskID).
		Update("status", models.TaskStatusPaused).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载调度器任务
	if err := s.scheduler.ReloadTasks(); err != nil {
		s.logger.Error("failed to reload tasks after pause", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task paused successfully",
		"task_id": taskID,
	})
}

// resumeTask 恢复任务调度
func (s *Server) resumeTask(c *gin.Context) {
	taskID := c.Param("id")

	// 查找任务
	var task models.Task
	if err := s.storage.DB().Where("id = ?", taskID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// 检查任务状态
	if task.Status == models.TaskStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task is already active"})
		return
	}

	if task.Status == models.TaskStatusDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot resume deleted task"})
		return
	}

	// 更新任务状态为活跃
	if err := s.storage.DB().
		Model(&task).
		Where("id = ?", taskID).
		Update("status", models.TaskStatusActive).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载调度器任务
	if err := s.scheduler.ReloadTasks(); err != nil {
		s.logger.Error("failed to reload tasks after resume", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task resumed successfully",
		"task_id": taskID,
	})
}

// getSchedulerStatus 获取调度器状态
func (s *Server) getSchedulerStatus(c *gin.Context) {
	var instances []models.SchedulerInstance
	if err := s.storage.DB().Find(&instances).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": instances,
		"time":      time.Now(),
	})
}

// getTaskStats 获取任务统计数据
func (s *Server) getTaskStats(c *gin.Context) {
	taskID := c.Param("id")

	// 获取24小时成功率
	var successRate24h float64
	var totalCount24h int64
	var successCount24h int64

	// 计算24小时前的时间
	since24h := time.Now().Add(-24 * time.Hour)

	// 获取24小时内的总执行次数
	if err := s.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND created_at >= ?", taskID, since24h).
		Count(&totalCount24h).Error; err != nil {
		s.logger.Error("failed to get 24h total count", zap.Error(err))
	}

	// 获取24小时内的成功执行次数
	if err := s.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusSuccess, since24h).
		Count(&successCount24h).Error; err != nil {
		s.logger.Error("failed to get 24h success count", zap.Error(err))
	}

	if totalCount24h > 0 {
		successRate24h = float64(successCount24h) / float64(totalCount24h) * 100
	}

	// 获取90天健康度统计
	healthStats90d := s.calculateHealthStats(taskID, 90)

	// 获取90天每日统计（用于状态图）
	var dailyStats []map[string]interface{}
	for i := 89; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayTotal, daySuccess int64

		// 总数
		s.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND created_at >= ? AND created_at < ?",
				taskID, startOfDay, endOfDay).
			Count(&dayTotal)

		// 成功数
		s.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
				taskID, models.ExecutionStatusSuccess, startOfDay, endOfDay).
			Count(&daySuccess)

		successRate := float64(100) // 默认100%（无执行时）
		if dayTotal > 0 {
			successRate = float64(daySuccess) / float64(dayTotal) * 100
		}

		dailyStats = append(dailyStats, map[string]interface{}{
			"date":        startOfDay.Format("2006-01-02"),
			"successRate": successRate,
			"total":       dayTotal,
		})
	}

	// 获取最近执行统计
	var recentExecutions []struct {
		Date        string  `json:"date"`
		Total       int     `json:"total"`
		Success     int     `json:"success"`
		Failed      int     `json:"failed"`
		SuccessRate float64 `json:"success_rate"`
	}

	// 按天统计最近7天的执行情况
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayTotal, daySuccess, dayFailed int64

		// 总数
		s.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND created_at >= ? AND created_at < ?",
				taskID, startOfDay, endOfDay).
			Count(&dayTotal)

		// 成功数
		s.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
				taskID, models.ExecutionStatusSuccess, startOfDay, endOfDay).
			Count(&daySuccess)

		// 失败数
		s.storage.DB().Model(&models.TaskExecution{}).
			Where("task_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
				taskID, []string{string(models.ExecutionStatusFailed), string(models.ExecutionStatusTimeout)},
				startOfDay, endOfDay).
			Count(&dayFailed)

		successRate := float64(0)
		if dayTotal > 0 {
			successRate = float64(daySuccess) / float64(dayTotal) * 100
		}

		recentExecutions = append(recentExecutions, struct {
			Date        string  `json:"date"`
			Total       int     `json:"total"`
			Success     int     `json:"success"`
			Failed      int     `json:"failed"`
			SuccessRate float64 `json:"success_rate"`
		}{
			Date:        startOfDay.Format("2006-01-02"),
			Total:       int(dayTotal),
			Success:     int(daySuccess),
			Failed:      int(dayFailed),
			SuccessRate: successRate,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success_rate_24h":  successRate24h,
		"total_24h":         totalCount24h,
		"success_24h":       successCount24h,
		"health_90d":        healthStats90d,
		"recent_executions": recentExecutions,
		"daily_stats_90d":   dailyStats,
	})
}

// calculateHealthStats 计算健康度统计
func (s *Server) calculateHealthStats(taskID string, days int) map[string]interface{} {
	since := time.Now().AddDate(0, 0, -days)

	var totalCount, successCount, failedCount, timeoutCount int64

	// 总执行次数
	s.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND created_at >= ?", taskID, since).
		Count(&totalCount)

	// 成功次数
	s.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusSuccess, since).
		Count(&successCount)

	// 失败次数
	s.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusFailed, since).
		Count(&failedCount)

	// 超时次数
	s.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND status = ? AND created_at >= ?",
			taskID, models.ExecutionStatusTimeout, since).
		Count(&timeoutCount)

	// 计算健康度分数 (0-100)
	healthScore := float64(100)
	if totalCount > 0 {
		// 成功率占70%权重
		successRate := float64(successCount) / float64(totalCount)
		healthScore = successRate * 70

		// 超时率占30%权重（超时越少分数越高）
		timeoutRate := float64(timeoutCount) / float64(totalCount)
		healthScore += (1 - timeoutRate) * 30
	}

	// 计算平均执行时间
	var avgDuration float64
	s.storage.DB().Model(&models.TaskExecution{}).
		Where("task_id = ? AND created_at >= ? AND start_time IS NOT NULL AND end_time IS NOT NULL",
			taskID, since).
		Select("AVG(TIMESTAMPDIFF(SECOND, start_time, end_time))").
		Scan(&avgDuration)

	return map[string]interface{}{
		"health_score":         healthScore,
		"total_count":          totalCount,
		"success_count":        successCount,
		"failed_count":         failedCount,
		"timeout_count":        timeoutCount,
		"avg_duration_seconds": avgDuration,
		"period_days":          days,
	}
}
