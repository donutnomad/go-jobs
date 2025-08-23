package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jobs/scheduler/internal/api"
	"github.com/jobs/scheduler/internal/infra/persistence/executionrepo"
	"github.com/jobs/scheduler/internal/infra/persistence/executorrepo"
	"github.com/jobs/scheduler/internal/infra/persistence/loadbalancerepo"
	"github.com/jobs/scheduler/internal/infra/persistence/schedulerinstancerepo"
	"github.com/jobs/scheduler/internal/infra/persistence/taskrepo"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/internal/scheduler"
	"github.com/jobs/scheduler/pkg/config"
	"github.com/jobs/scheduler/pkg/logger"
	"github.com/yitter/idgenerator-go/idgen"
	"go.uber.org/zap"
)

func main() {
	// 解析命令行参数
	var configPath string
	flag.StringVar(&configPath, "config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// 创建 IdGeneratorOptions 对象，可在构造函数中输入 WorkerId：
	var options = idgen.NewIdGeneratorOptions(20)
	options.BaseTime = 1755937966000
	options.WorkerIdBitLength = 6
	// options.WorkerIdBitLength = 10  // 默认值6，限定 WorkerId 最大值为2^6-1，即默认最多支持64个节点。
	// options.SeqBitLength = 6; // 默认值6，限制每毫秒生成的ID个数。若生成速度超过5万个/秒，建议加大 SeqBitLength 到 10。
	// options.BaseTime = Your_Base_Time // 如果要兼容老系统的雪花算法，此处应设置为老系统的BaseTime。

	idgen.SetIdGenerator(options)
	fmt.Println(idgen.NextId())

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建日志器
	zapLogger, err := logger.New(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer zapLogger.Sync()

	zapLogger.Info("Starting job scheduler",
		zap.String("instance_id", cfg.Scheduler.InstanceID))

	// 创建存储
	storageConfig := orm.Config{
		Host:                  cfg.Database.Host,
		Port:                  cfg.Database.Port,
		Database:              cfg.Database.Database,
		User:                  cfg.Database.User,
		Password:              cfg.Database.Password,
		MaxConnections:        cfg.Database.MaxConnections,
		MaxIdleConnections:    cfg.Database.MaxIdleConnections,
		ConnectionMaxLifetime: cfg.Database.ConnectionMaxLifetime,
	}

	db, err := orm.New(storageConfig)
	if err != nil {
		zapLogger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	myIP := cfg.Server.IP
	if myIP == "" {
		zapLogger.Fatal("Failed to get cfg.server.ip address")
	}

	// 创建repositories
	taskRepo := taskrepo.NewMysqlRepositoryImpl(db.DB())
	executionRepo := executionrepo.NewMysqlRepositoryImpl(db.DB())
	executorRepo := executorrepo.NewMysqlRepositoryImpl(db.DB())
	schedulerInstanceRepo := schedulerinstancerepo.NewMysqlRepositoryImpl(db.DB())
	loadBalanceRepo := loadbalancerepo.NewMysqlRepositoryImpl(db.DB())

	// 创建调度器
	sched, err := scheduler.New(
		*cfg, 
		db, 
		zapLogger, 
		api.ExecutionCallbackURL(net.JoinHostPort(myIP, strconv.Itoa(cfg.Server.Port)), false),
		taskRepo,
		executionRepo,
		executorRepo,
		schedulerInstanceRepo,
		loadBalanceRepo,
	)
	if err != nil {
		zapLogger.Fatal("Failed to create scheduler", zap.Error(err))
	}

	// 启动调度器
	if err := sched.Start(); err != nil {
		zapLogger.Fatal("Failed to start scheduler", zap.Error(err))
	}

	var emitter = api.NewEventBus(sched, sched.GetTaskRunner())

	// 创建API服务器
	apiServer := api.NewServer(db, emitter, zapLogger)

	// 启动HTTP服务器
	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        apiServer.Router(),
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	go func() {
		zapLogger.Info("Starting API server",
			zap.Int("port", cfg.Server.Port))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zapLogger.Fatal("Failed to start API server", zap.Error(err))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	zapLogger.Info("Shutting down...")

	// 优雅关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		zapLogger.Error("Failed to shutdown API server", zap.Error(err))
	}

	// 停止调度器
	if err := sched.Stop(); err != nil {
		zapLogger.Error("Failed to stop scheduler", zap.Error(err))
	}

	zapLogger.Info("Shutdown complete")
}
