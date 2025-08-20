package persistence

import (
	"context"
	"fmt"

	"github.com/jobs/scheduler/internal/app/infra/interfaces"
	"github.com/jobs/scheduler/internal/app/types"
	"gorm.io/gorm"
)

// DefaultTransactionManager 默认事务管理器实现
type DefaultTransactionManager struct {
	db *gorm.DB
}

// NewDefaultTransactionManager 创建默认事务管理器
func NewDefaultTransactionManager(db *gorm.DB) *DefaultTransactionManager {
	return &DefaultTransactionManager{
		db: db,
	}
}

// Execute 在事务中执行函数
func (tm *DefaultTransactionManager) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将事务实例注入到上下文中
		txCtx := context.WithValue(ctx, types.ContextTxKey{}, tx)
		return fn(txCtx)
	})
}

// DefaultDB 默认数据库接口实现
type DefaultDB struct {
	db *gorm.DB
}

// NewDefaultDB 创建默认数据库实现
func NewDefaultDB(db *gorm.DB) *DefaultDB {
	return &DefaultDB{
		db: db,
	}
}

// Model 指定模型
func (d *DefaultDB) Model(value any) *gorm.DB {
	return d.db.Model(value)
}

// Create 创建记录
func (d *DefaultDB) Create(value any) *gorm.DB {
	return d.db.Create(value)
}

// Save 保存记录
func (d *DefaultDB) Save(value any) *gorm.DB {
	return d.db.Save(value)
}

// Delete 删除记录
func (d *DefaultDB) Delete(value any, conds ...any) *gorm.DB {
	return d.db.Delete(value, conds...)
}

// Where 添加条件
func (d *DefaultDB) Where(query interface{}, args ...interface{}) *gorm.DB {
	return d.db.Where(query, args...)
}

// Find 查找记录
func (d *DefaultDB) Find(dest interface{}, conds ...interface{}) *gorm.DB {
	return d.db.Find(dest, conds...)
}

// First 查找第一条记录
func (d *DefaultDB) First(dest interface{}, conds ...interface{}) *gorm.DB {
	return d.db.First(dest, conds...)
}

// Last 查找最后一条记录
func (d *DefaultDB) Last(dest interface{}, conds ...interface{}) *gorm.DB {
	return d.db.Last(dest, conds...)
}

// Take 查找一条记录
func (d *DefaultDB) Take(dest interface{}, conds ...interface{}) *gorm.DB {
	return d.db.Take(dest, conds...)
}

// Count 计数
func (d *DefaultDB) Count(count *int64) *gorm.DB {
	return d.db.Count(count)
}

// Update 更新
func (d *DefaultDB) Update(column string, value interface{}) *gorm.DB {
	return d.db.Update(column, value)
}

// Updates 批量更新
func (d *DefaultDB) Updates(values interface{}) *gorm.DB {
	return d.db.Updates(values)
}

// Select 选择字段
func (d *DefaultDB) Select(query interface{}, args ...interface{}) *gorm.DB {
	return d.db.Select(query, args...)
}

// Omit 忽略字段
func (d *DefaultDB) Omit(columns ...string) *gorm.DB {
	return d.db.Omit(columns...)
}

// Joins 连接
func (d *DefaultDB) Joins(query string, args ...interface{}) *gorm.DB {
	return d.db.Joins(query, args...)
}

// Group 分组
func (d *DefaultDB) Group(name string) *gorm.DB {
	return d.db.Group(name)
}

// Having Having条件
func (d *DefaultDB) Having(query interface{}, args ...interface{}) *gorm.DB {
	return d.db.Having(query, args...)
}

// Order 排序
func (d *DefaultDB) Order(value interface{}) *gorm.DB {
	return d.db.Order(value)
}

// Limit 限制
func (d *DefaultDB) Limit(limit int) *gorm.DB {
	return d.db.Limit(limit)
}

// Offset 偏移
func (d *DefaultDB) Offset(offset int) *gorm.DB {
	return d.db.Offset(offset)
}

// Distinct 去重
func (d *DefaultDB) Distinct(args ...interface{}) *gorm.DB {
	return d.db.Distinct(args...)
}

// Table 指定表名
func (d *DefaultDB) Table(name string, args ...interface{}) *gorm.DB {
	return d.db.Table(name, args...)
}

// Raw 原生SQL
func (d *DefaultDB) Raw(sql string, values ...interface{}) *gorm.DB {
	return d.db.Raw(sql, values...)
}

// Exec 执行SQL
func (d *DefaultDB) Exec(sql string, values ...interface{}) *gorm.DB {
	return d.db.Exec(sql, values...)
}

// Transaction 事务处理
func (d *DefaultDB) Transaction(fn func(tx *gorm.DB) error, opts ...*gorm.Option) error {
	return d.db.Transaction(fn, opts...)
}

// WithContext 设置上下文
func (d *DefaultDB) WithContext(ctx context.Context) *gorm.DB {
	return d.db.WithContext(ctx)
}

// Session 创建会话
func (d *DefaultDB) Session(config *gorm.Session) *gorm.DB {
	return d.db.Session(config)
}

// Debug 开启调试
func (d *DefaultDB) Debug() *gorm.DB {
	return d.db.Debug()
}

// Error 获取错误
func (d *DefaultDB) Error() error {
	return d.db.Error
}

// RowsAffected 获取影响行数
func (d *DefaultDB) RowsAffected() int64 {
	return d.db.RowsAffected
}

// DefaultLogger 默认日志器实现
type DefaultLogger struct {
	// 这里可以使用zap或其他日志库
}

// NewDefaultLogger 创建默认日志器
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

// Debug 调试日志
func (l *DefaultLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	// 简化实现，实际应该使用专业的日志库
	fmt.Printf("[DEBUG] %s %+v\n", msg, fields)
}

// Info 信息日志
func (l *DefaultLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	fmt.Printf("[INFO] %s %+v\n", msg, fields)
}

// Warn 警告日志
func (l *DefaultLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	fmt.Printf("[WARN] %s %+v\n", msg, fields)
}

// Error 错误日志
func (l *DefaultLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	fmt.Printf("[ERROR] %s %+v\n", msg, fields)
}

// DefaultEventPublisher 默认事件发布器实现
type DefaultEventPublisher struct {
	logger interfaces.Logger
}

// NewDefaultEventPublisher 创建默认事件发布器
func NewDefaultEventPublisher(logger interfaces.Logger) *DefaultEventPublisher {
	return &DefaultEventPublisher{
		logger: logger,
	}
}

// Publish 发布事件
func (p *DefaultEventPublisher) Publish(ctx context.Context, event interface{}) error {
	// 简化实现，实际应该发布到消息队列或事件总线
	p.logger.Info(ctx, "Event published", map[string]interface{}{
		"event_type": fmt.Sprintf("%T", event),
		"event":      event,
	})
	return nil
}
