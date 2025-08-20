package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobs/scheduler/internal/api"
	"github.com/jobs/scheduler/internal/executor"
	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/scheduler"
	"github.com/jobs/scheduler/internal/storage"
	"github.com/jobs/scheduler/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestSetup 测试环境设置
type TestSetup struct {
	Storage         *storage.Storage
	Scheduler       *scheduler.Scheduler
	ExecutorManager *executor.Manager
	APIServer       *api.Server
	Router          *gin.Engine
	Logger          *zap.Logger
}

// SetupTest 初始化测试环境
func SetupTest(t *testing.T) *TestSetup {
	// 创建测试配置
	cfg := config.Config{
		Scheduler: config.SchedulerConfig{
			InstanceID:        "test-scheduler-001",
			LockKey:           "test_scheduler_lock",
			LockTimeout:       30 * time.Second,
			HeartbeatInterval: 10 * time.Second,
			MaxWorkers:        5,
		},
		HealthCheck: config.HealthCheckConfig{
			Enabled:           false, // 测试时禁用健康检查
			Interval:          30 * time.Second,
			Timeout:           5 * time.Second,
			FailureThreshold:  3,
			RecoveryThreshold: 2,
		},
		Database: config.DatabaseConfig{
			Host:                  "127.0.0.1",
			Port:                  3306,
			Database:              "jobs",
			User:                  "root",
			Password:              "123456",
			MaxConnections:        10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: time.Hour,
		},
		Server: config.ServerConfig{
			Port:           8081,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1048576,
		},
	}

	// 创建日志器
	logger := zap.NewNop()

	// 创建存储
	storageConfig := storage.Config{
		Host:                  cfg.Database.Host,
		Port:                  cfg.Database.Port,
		Database:              cfg.Database.Database,
		User:                  cfg.Database.User,
		Password:              cfg.Database.Password,
		MaxConnections:        cfg.Database.MaxConnections,
		MaxIdleConnections:    cfg.Database.MaxIdleConnections,
		ConnectionMaxLifetime: cfg.Database.ConnectionMaxLifetime,
	}

	db, err := storage.New(storageConfig)
	require.NoError(t, err)

	// 清理测试数据
	cleanupTestData(db)

	// 创建调度器
	sched, err := scheduler.New(cfg, db, logger)
	require.NoError(t, err)

	// 创建执行器管理器
	executorManager := executor.NewManager(db, logger)

	// 创建API服务器
	apiServer := api.NewServer(db, sched, executorManager, sched.GetTaskRunner(), logger)

	return &TestSetup{
		Storage:         db,
		Scheduler:       sched,
		ExecutorManager: executorManager,
		APIServer:       apiServer,
		Router:          apiServer.Router(),
		Logger:          logger,
	}
}

// cleanupTestData 清理测试数据
func cleanupTestData(db *storage.Storage) {
	db.DB().Exec("DELETE FROM task_executions")
	db.DB().Exec("DELETE FROM task_executors")
	db.DB().Exec("DELETE FROM tasks")
	db.DB().Exec("DELETE FROM executors")
	db.DB().Exec("DELETE FROM load_balance_state")
	db.DB().Exec("DELETE FROM scheduler_instances")
}

// TestTaskCreation 测试任务创建
func TestTaskCreation(t *testing.T) {
	setup := SetupTest(t)
	defer setup.Storage.Close()

	// 创建任务请求
	taskReq := api.CreateTaskRequest{
		Name:                "test_task_" + uuid.New().String()[:8],
		CronExpression:      "*/30 * * * * *",
		ExecutionMode:       models.ExecutionModeParallel,
		LoadBalanceStrategy: models.LoadBalanceRoundRobin,
		MaxRetry:            3,
		TimeoutSeconds:      60,
		Parameters: map[string]interface{}{
			"key": "value",
		},
	}

	body, _ := json.Marshal(taskReq)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var task models.Task
	err := json.Unmarshal(w.Body.Bytes(), &task)
	require.NoError(t, err)

	assert.Equal(t, taskReq.Name, task.Name)
	assert.Equal(t, taskReq.CronExpression, task.CronExpression)
	assert.Equal(t, taskReq.ExecutionMode, task.ExecutionMode)
	assert.Equal(t, taskReq.LoadBalanceStrategy, task.LoadBalanceStrategy)
	assert.Equal(t, models.TaskStatusActive, task.Status)
}

// TestExecutorRegistration 测试执行器注册
func TestExecutorRegistration(t *testing.T) {
	setup := SetupTest(t)
	defer setup.Storage.Close()

	// 创建任务
	task := models.Task{
		ID:                  uuid.New().String(),
		Name:                "test_task_" + uuid.New().String()[:8],
		CronExpression:      "*/30 * * * * *",
		ExecutionMode:       models.ExecutionModeParallel,
		LoadBalanceStrategy: models.LoadBalanceRoundRobin,
		Status:              models.TaskStatusActive,
	}
	err := setup.Storage.DB().Create(&task).Error
	require.NoError(t, err)

	// 注册执行器
	registerReq := executor.RegisterRequest{
		Name:           "test-executor",
		InstanceID:     "executor-test-001",
		BaseURL:        "http://localhost:9091",
		HealthCheckURL: "http://localhost:9091/health",
	}

	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/v1/executors/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var executorResp models.Executor
	err = json.Unmarshal(w.Body.Bytes(), &executorResp)
	require.NoError(t, err)

	assert.Equal(t, registerReq.Name, executorResp.Name)
	assert.Equal(t, registerReq.InstanceID, executorResp.InstanceID)
	assert.Equal(t, models.ExecutorStatusOnline, executorResp.Status)
	assert.True(t, executorResp.IsHealthy)
}

// TestTaskTrigger 测试手动触发任务
func TestTaskTrigger(t *testing.T) {
	setup := SetupTest(t)
	defer setup.Storage.Close()

	// 启动调度器
	err := setup.Scheduler.Start()
	require.NoError(t, err)
	defer setup.Scheduler.Stop()

	// 创建任务
	task := models.Task{
		ID:                  uuid.New().String(),
		Name:                "test_task_" + uuid.New().String()[:8],
		CronExpression:      "*/30 * * * * *",
		ExecutionMode:       models.ExecutionModeParallel,
		LoadBalanceStrategy: models.LoadBalanceRoundRobin,
		Status:              models.TaskStatusActive,
		TimeoutSeconds:      60,
	}
	err = setup.Storage.DB().Create(&task).Error
	require.NoError(t, err)

	// 创建模拟执行器
	mockExecutor := createMockExecutor(t, "9092")
	defer mockExecutor.Close()

	// 注册执行器
	executorModel := models.Executor{
		ID:             uuid.New().String(),
		Name:           "test-executor",
		InstanceID:     "executor-test-002",
		BaseURL:        "http://localhost:9092",
		HealthCheckURL: "http://localhost:9092/health",
		Status:         models.ExecutorStatusOnline,
		IsHealthy:      true,
	}
	now := time.Now()
	executorModel.LastHeartbeat = &now
	err = setup.Storage.DB().Create(&executorModel).Error
	require.NoError(t, err)

	// 创建任务-执行器关联
	taskExecutor := models.TaskExecutor{
		ID:         uuid.New().String(),
		TaskID:     task.ID,
		ExecutorID: executorModel.ID,
		Priority:   10,
		Weight:     1,
	}
	err = setup.Storage.DB().Create(&taskExecutor).Error
	require.NoError(t, err)

	// 触发任务
	triggerReq := executor.TriggerTaskRequest{
		Parameters: map[string]interface{}{
			"test": "manual",
		},
	}

	body, _ := json.Marshal(triggerReq)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/tasks/%s/trigger", task.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var execution models.TaskExecution
	err = json.Unmarshal(w.Body.Bytes(), &execution)
	require.NoError(t, err)

	assert.Equal(t, task.ID, execution.TaskID)
	assert.Equal(t, models.ExecutionStatusPending, execution.Status)

	// 等待执行完成
	time.Sleep(2 * time.Second)

	// 检查执行状态
	var updatedExecution models.TaskExecution
	err = setup.Storage.DB().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)

	// 执行应该已经开始
	assert.NotNil(t, updatedExecution.StartTime)
}

// TestExecutionHistory 测试执行历史查询
func TestExecutionHistory(t *testing.T) {
	setup := SetupTest(t)
	defer setup.Storage.Close()

	// 创建任务
	task := models.Task{
		ID:             uuid.New().String(),
		Name:           "test_task_" + uuid.New().String()[:8],
		CronExpression: "*/30 * * * * *",
		Status:         models.TaskStatusActive,
	}
	err := setup.Storage.DB().Create(&task).Error
	require.NoError(t, err)

	// 创建多个执行记录
	for i := 0; i < 5; i++ {
		execution := models.TaskExecution{
			ID:            uuid.New().String(),
			TaskID:        task.ID,
			ScheduledTime: time.Now().Add(time.Duration(i) * time.Minute),
			Status:        models.ExecutionStatusSuccess,
			Logs:          fmt.Sprintf("Execution %d", i),
		}
		startTime := time.Now()
		endTime := startTime.Add(10 * time.Second)
		execution.StartTime = &startTime
		execution.EndTime = &endTime

		err = setup.Storage.DB().Create(&execution).Error
		require.NoError(t, err)
	}

	// 查询执行历史
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/executions?task_id=%s", task.ID), nil)
	w := httptest.NewRecorder()
	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var executions []models.TaskExecution
	err = json.Unmarshal(w.Body.Bytes(), &executions)
	require.NoError(t, err)

	assert.Len(t, executions, 5)
	for _, exec := range executions {
		assert.Equal(t, task.ID, exec.TaskID)
		assert.Equal(t, models.ExecutionStatusSuccess, exec.Status)
	}
}

// TestLoadBalancing 测试负载均衡
func TestLoadBalancing(t *testing.T) {
	setup := SetupTest(t)
	defer setup.Storage.Close()

	ctx := context.Background()

	// 创建任务
	task := models.Task{
		ID:                  uuid.New().String(),
		Name:                "test_task_lb",
		CronExpression:      "*/30 * * * * *",
		LoadBalanceStrategy: models.LoadBalanceRoundRobin,
		Status:              models.TaskStatusActive,
	}
	err := setup.Storage.DB().Create(&task).Error
	require.NoError(t, err)

	// 创建多个执行器
	var executors []models.Executor
	for i := 0; i < 3; i++ {
		executor := models.Executor{
			ID:         uuid.New().String(),
			Name:       fmt.Sprintf("executor-%d", i),
			InstanceID: fmt.Sprintf("executor-inst-%d", i),
			BaseURL:    fmt.Sprintf("http://localhost:909%d", i),
			Status:     models.ExecutorStatusOnline,
			IsHealthy:  true,
		}
		now := time.Now()
		executor.LastHeartbeat = &now
		err = setup.Storage.DB().Create(&executor).Error
		require.NoError(t, err)
		executors = append(executors, executor)

		// 创建任务-执行器关联
		taskExecutor := models.TaskExecutor{
			ID:         uuid.New().String(),
			TaskID:     task.ID,
			ExecutorID: executor.ID,
			Weight:     1,
		}
		err = setup.Storage.DB().Create(&taskExecutor).Error
		require.NoError(t, err)
	}

	// 获取健康的执行器
	healthyExecutors, err := setup.ExecutorManager.GetHealthyExecutors(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, healthyExecutors, 3)

	// 验证所有执行器都被正确加载
	executorMap := make(map[string]bool)
	for _, exec := range healthyExecutors {
		executorMap[exec.ID] = true
	}
	assert.Len(t, executorMap, 3, "Should have 3 unique executors")
}

// createMockExecutor 创建模拟执行器服务器
func createMockExecutor(t *testing.T, port string) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})

		case "/execute":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			// 模拟异步执行
			go func() {
				time.Sleep(1 * time.Second)

				// 回调
				if callbackURL, ok := req["callback_url"].(string); ok {
					callback := map[string]interface{}{
						"execution_id": req["execution_id"],
						"status":       "success",
						"result":       map[string]interface{}{"test": "result"},
						"logs":         "Task completed successfully",
					}

					body, _ := json.Marshal(callback)
					http.Post(callbackURL, "application/json", bytes.NewBuffer(body))
				}
			}()

			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]string{
				"message":      "Task accepted",
				"execution_id": req["execution_id"].(string),
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	server := httptest.NewUnstartedServer(handler)
	server.Listener.Close()
	server.Listener, _ = net.Listen("tcp", ":"+port)
	server.Start()

	return server
}
