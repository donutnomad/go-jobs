package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Locker MySQL分布式锁
type Locker struct {
	db       *sql.DB
	lockName string
	timeout  time.Duration
	logger   *zap.Logger
	locked   bool
}

// NewLocker 创建分布式锁
func NewLocker(db *sql.DB, lockName string, timeout time.Duration, logger *zap.Logger) *Locker {
	return &Locker{
		db:       db,
		lockName: lockName,
		timeout:  timeout,
		logger:   logger,
		locked:   false,
	}
}

// TryLock 尝试获取锁
func (l *Locker) TryLock(ctx context.Context) (bool, error) {
	if l.locked {
		return true, nil
	}

	// MySQL GET_LOCK 函数
	// 返回值: 1-成功获取锁, 0-超时, NULL-错误
	query := "SELECT GET_LOCK(?, ?)"
	timeoutSeconds := int(l.timeout.Seconds())

	var result sql.NullInt64
	err := l.db.QueryRowContext(ctx, query, l.lockName, timeoutSeconds).Scan(&result)
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !result.Valid {
		return false, fmt.Errorf("lock query returned NULL")
	}

	if result.Int64 == 1 {
		l.locked = true
		l.logger.Info("acquired distributed lock",
			zap.String("lock_name", l.lockName))
		return true, nil
	}

	return false, nil
}

// Unlock 释放锁
func (l *Locker) Unlock(ctx context.Context) error {
	if !l.locked {
		return nil
	}

	// MySQL RELEASE_LOCK 函数
	// 返回值: 1-成功释放锁, 0-锁不存在或不是持有者, NULL-错误
	query := "SELECT RELEASE_LOCK(?)"

	var result sql.NullInt64
	err := l.db.QueryRowContext(ctx, query, l.lockName).Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("release lock query returned NULL")
	}

	if result.Int64 == 1 {
		l.locked = false
		l.logger.Info("released distributed lock",
			zap.String("lock_name", l.lockName))
		return nil
	}

	return fmt.Errorf("failed to release lock: not owner or lock does not exist")
}

// IsLocked 检查是否持有锁
func (l *Locker) IsLocked() bool {
	return l.locked
}

// Renew 续约锁（MySQL GET_LOCK会自动续约，这里主要用于心跳检测）
func (l *Locker) Renew(ctx context.Context) error {
	if !l.locked {
		return fmt.Errorf("not holding lock")
	}

	// 检查连接是否有效
	if err := l.db.PingContext(ctx); err != nil {
		l.locked = false
		return fmt.Errorf("database connection lost: %w", err)
	}

	// MySQL的GET_LOCK是会话级别的，只要连接不断开，锁就会一直持有
	// 这里我们可以通过IS_USED_LOCK检查锁的状态
	query := "SELECT IS_USED_LOCK(?)"

	var result sql.NullString
	err := l.db.QueryRowContext(ctx, query, l.lockName).Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to check lock status: %w", err)
	}

	if !result.Valid || result.String == "" {
		l.locked = false
		return fmt.Errorf("lock is not held")
	}

	return nil
}

// WithLock 在持有锁的情况下执行函数
func (l *Locker) WithLock(ctx context.Context, fn func() error) error {
	locked, err := l.TryLock(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !locked {
		return fmt.Errorf("could not acquire lock within timeout")
	}

	defer func() {
		if err := l.Unlock(ctx); err != nil {
			l.logger.Error("failed to release lock",
				zap.String("lock_name", l.lockName),
				zap.Error(err))
		}
	}()

	return fn()
}
