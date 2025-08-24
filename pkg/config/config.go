package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
    Scheduler   SchedulerConfig   `mapstructure:"scheduler"`
    HealthCheck HealthCheckConfig `mapstructure:"health_check"`
    Database    DatabaseConfig    `mapstructure:"database"`
    Server      ServerConfig      `mapstructure:"server"`
    Log         LogConfig         `mapstructure:"log"`
    Redis       RedisConfig       `mapstructure:"redis"`
}

type SchedulerConfig struct {
	InstanceID        string        `mapstructure:"instance_id"`
	LockKey           string        `mapstructure:"lock_key"`
	LockTimeout       time.Duration `mapstructure:"lock_timeout"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	MaxWorkers        int           `mapstructure:"max_workers"`
}

type HealthCheckConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	Interval          time.Duration `mapstructure:"interval"`
	Timeout           time.Duration `mapstructure:"timeout"`
	FailureThreshold  int           `mapstructure:"failure_threshold"`
	RecoveryThreshold int           `mapstructure:"recovery_threshold"`
}

type DatabaseConfig struct {
	Host                  string        `mapstructure:"host"`
	Port                  int           `mapstructure:"port"`
	Database              string        `mapstructure:"database"`
	User                  string        `mapstructure:"user"`
	Password              string        `mapstructure:"password"`
	MaxConnections        int           `mapstructure:"max_connections"`
	MaxIdleConnections    int           `mapstructure:"max_idle_connections"`
	ConnectionMaxLifetime time.Duration `mapstructure:"connection_max_lifetime"`
}

type ServerConfig struct {
	IP             string        `mapstructure:"ip"`
	Port           int           `mapstructure:"port"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes"`
}

type LogConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
    Output string `mapstructure:"output"`
    File   string `mapstructure:"file"`
}

type RedisConfig struct {
    Enabled  bool   `mapstructure:"enabled"`
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	viper.SetDefault("scheduler.instance_id", "scheduler-001")
	viper.SetDefault("scheduler.lock_key", "scheduler_leader_lock")
	viper.SetDefault("scheduler.lock_timeout", "30s")
	viper.SetDefault("scheduler.heartbeat_interval", "10s")
	viper.SetDefault("scheduler.max_workers", 10)

	viper.SetDefault("health_check.enabled", true)
	viper.SetDefault("health_check.interval", "30s")
	viper.SetDefault("health_check.timeout", "5s")
	viper.SetDefault("health_check.failure_threshold", 3)
	viper.SetDefault("health_check.recovery_threshold", 2)

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.max_connections", 20)
	viper.SetDefault("database.max_idle_connections", 10)
	viper.SetDefault("database.connection_max_lifetime", "1h")

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.max_header_bytes", 1048576)

	viper.SetDefault("log.level", "info")
    viper.SetDefault("log.format", "json")
    viper.SetDefault("log.output", "stdout")

    // redis defaults
    viper.SetDefault("redis.enabled", false)
    viper.SetDefault("redis.host", "localhost")
    viper.SetDefault("redis.port", 6379)
    viper.SetDefault("redis.password", "")
    viper.SetDefault("redis.db", 0)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

    return &cfg, nil
}
