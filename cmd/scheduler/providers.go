package main

import (
	"fmt"

	redis "github.com/go-redis/redis/v8"
	"github.com/jobs/scheduler/pkg/config"
)

// ProvideRedisClient builds a redis client from typed config.
// Returns nil when redis is disabled.
func ProvideRedisClient(cfg config.Config) *redis.Client {
	if !cfg.Redis.Enabled {
		return nil
	}
	addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}
