package orm

import (
	"fmt"
	"time"

	"github.com/google/wire"
	"github.com/jobs/scheduler/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Provider = wire.NewSet(New)

type Config struct {
	Host                  string
	Port                  int
	Database              string
	User                  string
	Password              string
	MaxConnections        int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
}

type Storage struct {
	db *gorm.DB
}

func New(cfg Config) (*Storage, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用外键约束创建，保留关联关系
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(cfg.ConnectionMaxLifetime)

	// 自动迁移 - 调整顺序确保被引用的表先创建
	if err := db.AutoMigrate(
		&models.Task{},              // 先创建tasks表
		&models.Executor{},          // 再创建executors表
		&models.TaskExecutor{},      // 然后创建task_executors表(引用了tasks和executors)
		&models.TaskExecution{},     // 再创建task_executions表(引用了tasks和executors)
		&models.LoadBalanceState{},  // 负载均衡状态表
		&models.SchedulerInstance{}, // 调度器实例表
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) DB() *gorm.DB {
	return s.db
}

func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Storage) Ping() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
