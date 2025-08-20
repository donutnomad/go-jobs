package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobs/scheduler/internal/app/biz/executionview"
	"github.com/jobs/scheduler/internal/app/biz/shared"
	"github.com/jobs/scheduler/internal/app/service"
)

// ExecutionHandler 执行处理器
type ExecutionHandler struct {
	executionService *service.ExecutionService
	queryService     executionview.IQueryService
}

// NewExecutionHandler 创建执行处理器
func NewExecutionHandler(executionService *service.ExecutionService, queryService executionview.IQueryService) *ExecutionHandler {
	return &ExecutionHandler{
		executionService: executionService,
		queryService:     queryService,
	}
}

// ListExecutions 列出执行
func (h *ExecutionHandler) ListExecutions(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建过滤器
	var filter executionview.ExecutionFilter

	if taskID := c.Query("task_id"); taskID != "" {
		filter.TaskID = &taskID
	}

	if executorID := c.Query("executor_id"); executorID != "" {
		filter.ExecutorID = &executorID
	}

	if status := c.Query("status"); status != "" {
		filter.Status = []string{status}
	}

	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filter.DateFrom = &t
		}
	}

	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			filter.DateTo = &t
		}
	}

	// 查询执行列表
	result, err := h.queryService.ListExecutions(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 计算总页数
	totalPages := int((result.Total + int64(pageSize) - 1) / int64(pageSize))

	// 返回前端期望的分页格式
	c.JSON(http.StatusOK, gin.H{
		"data":        result.Items,
		"total":       result.Total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// GetExecution 获取执行详情
func (h *ExecutionHandler) GetExecution(c *gin.Context) {
	idStr := c.Param("id")
	id, err := shared.NewID(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的执行ID"})
		return
	}

	execution, err := h.queryService.GetExecutionDetail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    execution,
		"message": "success",
	})
}

// GetExecutionStatistics 获取执行统计
func (h *ExecutionHandler) GetExecutionStatistics(c *gin.Context) {
	// 解析日期范围参数
	startStr := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -7).Format("2006-01-02"))
	endStr := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
		return
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
		return
	}

	// 设置为一天的结束时间
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	stats, err := h.queryService.GetExecutionStatistics(c.Request.Context(), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    stats,
		"message": "success",
	})
}

// GetRunningExecutions 获取正在运行的执行
func (h *ExecutionHandler) GetRunningExecutions(c *gin.Context) {
	executions, err := h.queryService.GetRunningExecutions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    executions,
		"message": "success",
	})
}

// GetTaskExecutionHistory 获取任务执行历史
func (h *ExecutionHandler) GetTaskExecutionHistory(c *gin.Context) {
	taskIDStr := c.Param("task_id")
	taskID, err := shared.NewID(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	result, err := h.queryService.GetTaskExecutionHistory(c.Request.Context(), taskID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    result,
		"message": "success",
	})
}

// GetFailureAnalysis 获取失败分析
func (h *ExecutionHandler) GetFailureAnalysis(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 || days > 90 {
		days = 7
	}

	analysis, err := h.queryService.GetFailureAnalysis(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    analysis,
		"message": "success",
	})
}

// GetPerformanceAnalysis 获取性能分析
func (h *ExecutionHandler) GetPerformanceAnalysis(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 || days > 90 {
		days = 7
	}

	var taskID *shared.ID
	var executorID *shared.ID

	if taskIDStr := c.Query("task_id"); taskIDStr != "" {
		if id, err := shared.NewID(taskIDStr); err == nil {
			taskID = &id
		}
	}

	if executorIDStr := c.Query("executor_id"); executorIDStr != "" {
		if id, err := shared.NewID(executorIDStr); err == nil {
			executorID = &id
		}
	}

	analysis, err := h.queryService.GetPerformanceAnalysis(c.Request.Context(), taskID, executorID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    analysis,
		"message": "success",
	})
}
