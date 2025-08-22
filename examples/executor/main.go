package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	ExecutionID string         `json:"execution_id"`
	TaskID      string         `json:"task_id"`
	TaskName    string         `json:"task_name"`
	Parameters  map[string]any `json:"parameters"`
	CallbackURL string         `json:"callback_url"`
}

// CallbackRequest 回调请求
type CallbackRequest struct {
	ExecutionID string         `json:"execution_id"`
	Status      string         `json:"status"`
	Result      map[string]any `json:"result"`
	Logs        string         `json:"logs"`
}

// TaskDefinition 任务定义
type TaskDefinition struct {
	Name                string         `json:"name"`
	ExecutionMode       string         `json:"execution_mode"`
	CronExpression      string         `json:"cron_expression"`
	LoadBalanceStrategy string         `json:"load_balance_strategy"`
	MaxRetry            int            `json:"max_retry"`
	TimeoutSeconds      int            `json:"timeout_seconds"`
	Parameters          map[string]any `json:"parameters"`
	Status              string         `json:"status"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	ExecutorID     string           `json:"executor_id"`
	ExecutorName   string           `json:"executor_name"`
	ExecutorURL    string           `json:"executor_url"`
	HealthCheckURL string           `json:"health_check_url"`
	Tasks          []TaskDefinition `json:"tasks"`
	Metadata       map[string]any   `json:"metadata"`
}

// StopRequest 停止任务请求
type StopRequest struct {
	ExecutionID string `json:"execution_id"`
}

// RunningTask 正在运行的任务
type RunningTask struct {
	ExecutionID string
	TaskName    string
	Cancel      context.CancelFunc
	StartTime   time.Time
}

// TaskManager 任务管理器
type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*RunningTask
}

// NewTaskManager 创建任务管理器
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*RunningTask),
	}
}

// Add 添加任务
func (tm *TaskManager) Add(task *RunningTask) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tasks[task.ExecutionID] = task
}

// Remove 移除任务
func (tm *TaskManager) Remove(executionID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.tasks, executionID)
}

// Get 获取任务
func (tm *TaskManager) Get(executionID string) (*RunningTask, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, ok := tm.tasks[executionID]
	return task, ok
}

// Stop 停止任务
func (tm *TaskManager) Stop(executionID string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[executionID]; ok {
		task.Cancel()
		delete(tm.tasks, executionID)
		return true
	}
	return false
}

var taskManager *TaskManager

const (
	ExecutorID   = "executor-1754725846387-v7gat"
	ExecutorName = "001"
	Port         = ":9090"
	SchedulerURL = "http://localhost:8080"
)

func main() {
	// 初始化任务管理器
	taskManager = NewTaskManager()

	// 注册路由
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/execute", executeHandler)
	http.HandleFunc("/stop", stopHandler)

	// 启动时自动注册到调度器
	if err := registerToScheduler(); err != nil {
		log.Printf("Failed to register to scheduler: %v", err)
		log.Println("Continuing without registration...")
	}

	// 启动执行器服务
	log.Printf("Sample executor listening on %s", Port)
	log.Printf("Executor ID: %s", ExecutorID)
	log.Fatal(http.ListenAndServe(Port, nil))
}

// registerToScheduler 向调度器注册执行器和任务
func registerToScheduler() error {
	// 获取本机IP（简化版，实际生产环境可能需要更复杂的逻辑）
	executorURL := fmt.Sprintf("http://localhost%s", Port)

	// 定义要注册的任务
	tasks := []TaskDefinition{
		{
			Name:                "数据同步任务",
			ExecutionMode:       "parallel",
			CronExpression:      "0 */5 * * * *", // 每5分钟执行一次
			LoadBalanceStrategy: "round_robin",
			MaxRetry:            3,
			TimeoutSeconds:      300,
			Parameters: map[string]any{
				"source_db": "mysql_primary",
				"target_db": "mysql_replica",
			},
			Status: "paused", // 初始为暂停状态
		},
		{
			Name:                "系统清理任务",
			ExecutionMode:       "sequential",
			CronExpression:      "0 0 2 * * *", // 每天凌晨2点执行
			LoadBalanceStrategy: "least_loaded",
			MaxRetry:            2,
			TimeoutSeconds:      600,
			Parameters: map[string]any{
				"cleanup_days": 7,
				"path":         "/tmp/logs",
			},
			Status: "paused", // 初始为暂停状态
		},
		{
			Name:                "健康检查任务",
			ExecutionMode:       "parallel",
			CronExpression:      "0 */1 * * * *", // 每分钟执行一次
			LoadBalanceStrategy: "random",
			MaxRetry:            1,
			TimeoutSeconds:      60,
			Parameters:          map[string]any{},
			Status:              "active", // 活跃状态，立即可执行
		},
	}

	// 构建注册请求
	registerReq := RegisterRequest{
		ExecutorID:     ExecutorID,
		ExecutorName:   ExecutorName,
		ExecutorURL:    executorURL,
		HealthCheckURL: executorURL + "/health",
		Tasks:          tasks,
		Metadata: map[string]any{
			"version":     "1.0.0",
			"language":    "go",
			"description": "示例执行器，演示自动注册功能",
			"features":    []string{"data_sync", "system_cleanup", "health_check"},
		},
	}

	// 序列化请求数据
	jsonData, err := json.MarshalIndent(registerReq, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal register request: %w", err)
	}

	log.Printf("Registering to scheduler with data:\n%s", string(jsonData))

	// 发送注册请求
	registerURL := fmt.Sprintf("%s/api/v1/executors/register", SchedulerURL)
	resp, err := http.Post(registerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send register request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return fmt.Errorf("registration failed with status %d: %v", resp.StatusCode, errorResp)
		}
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	// 解析响应
	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("Successfully registered to scheduler!")
	log.Printf("Response: %+v", response)
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "healthy",
		"time":        time.Now().Format(time.RFC3339),
		"executor_id": ExecutorID,
	})
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received task: %s (ID: %s)", req.TaskName, req.ExecutionID)
	log.Printf("Task parameters: %+v", req.Parameters)

	// 创建可取消的 context
	ctx, cancel := context.WithCancel(context.Background())

	// 添加到任务管理器
	task := &RunningTask{
		ExecutionID: req.ExecutionID,
		TaskName:    req.TaskName,
		Cancel:      cancel,
		StartTime:   time.Now(),
	}
	taskManager.Add(task)

	// 异步执行任务
	go func() {
		defer taskManager.Remove(req.ExecutionID)

		// 模拟任务执行时间
		duration := time.Duration(rand.Intn(10)+1) * time.Second
		log.Printf("Executing task %s for %v...", req.TaskName, duration)

		// 使用带超时的 context
		timer := time.NewTimer(duration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			// 任务被取消
			log.Printf("Task %s (ID: %s) was cancelled", req.TaskName, req.ExecutionID)
			callback := CallbackRequest{
				ExecutionID: req.ExecutionID,
				Status:      "cancelled",
				Result:      map[string]any{"reason": "Task was stopped by user"},
				Logs:        fmt.Sprintf("Task %s was cancelled after %v", req.TaskName, time.Since(task.StartTime)),
			}
			if err := sendCallback(req.CallbackURL, callback); err != nil {
				log.Printf("Failed to send cancellation callback: %v", err)
			}
			return
		case <-timer.C:
			// 继续执行任务
		}

		// 根据任务名称执行不同的逻辑
		status := "success"
		result := make(map[string]any)
		logs := ""

		switch req.TaskName {
		case "数据同步任务":
			// 模拟数据同步
			recordCount := rand.Intn(1000) + 100
			result["records_synced"] = recordCount
			result["source_db"] = req.Parameters["source_db"]
			result["target_db"] = req.Parameters["target_db"]
			logs = fmt.Sprintf("同步了 %d 条记录从 %v 到 %v", recordCount, req.Parameters["source_db"], req.Parameters["target_db"])

		case "系统清理任务":
			// 模拟系统清理
			filesDeleted := rand.Intn(50) + 10
			result["files_deleted"] = filesDeleted
			result["cleanup_days"] = req.Parameters["cleanup_days"]
			logs = fmt.Sprintf("清理了 %d 个文件，保留 %v 天内的文件", filesDeleted, req.Parameters["cleanup_days"])

		case "健康检查任务":
			// 模拟健康检查
			services := []string{"database", "redis", "api_gateway"}
			healthyServices := rand.Intn(len(services)) + 1
			result["total_services"] = len(services)
			result["healthy_services"] = healthyServices
			result["services"] = services[:healthyServices]
			logs = fmt.Sprintf("检查了 %d 个服务，%d 个健康", len(services), healthyServices)

		default:
			// 通用任务执行
			logs = fmt.Sprintf("Task %s executed successfully in %v", req.TaskName, duration)
		}

		// 随机模拟10%的失败率
		if rand.Float32() < 0.1 {
			status = "failed"
			logs = fmt.Sprintf("Task %s failed after %v: simulated failure", req.TaskName, duration)
			result["error"] = "simulated execution failure"
		}

		result["duration"] = duration.Seconds()
		result["execution_time"] = time.Now().Format(time.RFC3339)

		// 再次检查是否被取消
		select {
		case <-ctx.Done():
			log.Printf("Task %s (ID: %s) was cancelled before callback", req.TaskName, req.ExecutionID)
			callback := CallbackRequest{
				ExecutionID: req.ExecutionID,
				Status:      "cancelled",
				Result:      map[string]any{"reason": "Task was stopped by user"},
				Logs:        fmt.Sprintf("Task %s was cancelled after %v", req.TaskName, time.Since(task.StartTime)),
			}
			if err := sendCallback(req.CallbackURL, callback); err != nil {
				log.Printf("Failed to send cancellation callback: %v", err)
			}
			return
		default:
			// 回调调度器
			callback := CallbackRequest{
				ExecutionID: req.ExecutionID,
				Status:      status,
				Result:      result,
				Logs:        logs,
			}

			if err := sendCallback(req.CallbackURL, callback); err != nil {
				log.Printf("Failed to send callback: %v", err)
			} else {
				log.Printf("Callback sent for execution %s (status: %s)", req.ExecutionID, status)
			}
		}
	}()

	// 立即返回接受响应
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"message":      "Task accepted",
		"execution_id": req.ExecutionID,
		"task_name":    req.TaskName,
	})
}

// stopHandler 处理停止任务请求
func stopHandler(w http.ResponseWriter, r *http.Request) {
	var req StopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received stop request for execution: %s", req.ExecutionID)

	if taskManager.Stop(req.ExecutionID) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message":      "Task stopped successfully",
			"execution_id": req.ExecutionID,
		})
		log.Printf("Task %s stopped successfully", req.ExecutionID)
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error":        "Task not found or already completed",
			"execution_id": req.ExecutionID,
		})
		log.Printf("Task %s not found or already completed", req.ExecutionID)
	}
}

func sendCallback(url string, callback CallbackRequest) error {
	jsonData, err := json.Marshal(callback)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}

	return nil
}
