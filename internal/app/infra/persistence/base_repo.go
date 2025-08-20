package persistence

import (
	"context"

	"github.com/jobs/scheduler/internal/app/infra/interfaces"
	"github.com/jobs/scheduler/internal/app/types"
	"gorm.io/gorm"
)

// DefaultRepo 提供事务管理的基础Repository实现
// 遵循架构指导文档的设计模式
type DefaultRepo struct {
	db interfaces.DB
}

// NewDefaultRepo 创建默认Repository
func NewDefaultRepo(db interfaces.DB) DefaultRepo {
	return DefaultRepo{db: db}
}

// Execute 执行事务
func (r *DefaultRepo) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将事务存储在context中
		txCtx := context.WithValue(ctx, types.ContextTxKey{}, tx)
		return fn(txCtx)
	})
}

// dbFromContext 从context中获取数据库连接（可能是事务）
func (r *DefaultRepo) dbFromContext(ctx context.Context) interfaces.DB {
	// 尝试从context中获取事务
	if tx, ok := ctx.Value(types.ContextTxKey{}).(*gorm.DB); ok {
		return tx
	}
	// 没有事务则返回普通数据库连接
	return r.db
}

// Db 获取当前的数据库连接（自动识别是否在事务中）
func (r *DefaultRepo) Db(ctx context.Context) interfaces.DB {
	return r.dbFromContext(ctx).WithContext(ctx)
}

// GetDB 获取原始数据库连接（用于特殊操作）
func (r *DefaultRepo) GetDB() interfaces.DB {
	return r.db
}

// IsInTransaction 检查是否在事务中
func (r *DefaultRepo) IsInTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(types.ContextTxKey{}).(*gorm.DB)
	return ok
}
