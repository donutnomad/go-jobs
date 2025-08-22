package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jobs/scheduler/internal/api"
	"github.com/jobs/scheduler/internal/executor"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/internal/scheduler"
	"github.com/jobs/scheduler/pkg/config"
	"github.com/jobs/scheduler/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 解析命令行参数
	var configPath string
	flag.StringVar(&configPath, "config", "configs/config.yaml", "path to config file")
	flag.Parse()

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

	// 创建调度器
	sched, err := scheduler.New(*cfg, db, zapLogger)
	if err != nil {
		zapLogger.Fatal("Failed to create scheduler", zap.Error(err))
	}

	// 启动调度器
	if err := sched.Start(); err != nil {
		zapLogger.Fatal("Failed to start scheduler", zap.Error(err))
	}

	// 创建执行器管理器
	executorManager := executor.NewManager(db, zapLogger)

	// 创建API服务器
	apiServer := api.NewServer(db, sched, executorManager, sched.GetTaskRunner(), zapLogger)

	// 启动HTTP服务器
	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        apiServer.Router(),
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// 在新的goroutine中启动服务器
	go func() {
		zapLogger.Info("Starting API server",
			zap.Int("port", cfg.Server.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Failed to start API server", zap.Error(err))
		}
	}()

	// 等待中断信号
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
