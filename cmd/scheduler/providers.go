package main

import (
	"context"
	"fmt"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/jobs/scheduler/pkg/config"
	"go.uber.org/zap"
)

// ProvideRedisClient builds a redis client from typed config.
// Returns nil when redis is disabled.
// It also tests the connection and logs configuration details.
func ProvideRedisClient(cfg config.Config, logger *zap.Logger) *redis.Client {
	if !cfg.Redis.Enabled {
		logger.Info("Redis is disabled in configuration")
		return nil
	}
	addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	opts := &redis.Options{
		Addr: addr,
		DB:   cfg.Redis.DB,
	}
	
	// 只有当密码非空时才设置 Password，避免对无密码的 Redis 服务器发送 AUTH 命令
	hasPassword := cfg.Redis.Password != ""
	if hasPassword {
		opts.Password = cfg.Redis.Password
		logger.Info("Redis client configured with password",
			zap.String("host", cfg.Redis.Host),
			zap.Int("port", cfg.Redis.Port),
			zap.Int("db", cfg.Redis.DB),
			zap.Bool("has_password", true))
	} else {
		logger.Info("Redis client configured without password",
			zap.String("host", cfg.Redis.Host),
			zap.Int("port", cfg.Redis.Port),
			zap.Int("db", cfg.Redis.DB),
			zap.Bool("has_password", false))
	}
	
	client := redis.NewClient(opts)
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis",
			zap.String("address", addr),
			zap.Bool("has_password", hasPassword),
			zap.Error(err))
		// 不返回 nil，让调用者决定如何处理
		// 如果 Redis 不可用，EventBus 会回退到直接调用
	} else {
		logger.Info("Successfully connected to Redis",
			zap.String("address", addr),
			zap.Int("db", cfg.Redis.DB))
	}
	
	return client
}
