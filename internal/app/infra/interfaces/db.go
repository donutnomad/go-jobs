package interfaces

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DB 数据库接口，遵循架构指导文档的定义
type DB interface {
	Model(value any) *gorm.DB
	Create(value any) *gorm.DB
	Save(value any) *gorm.DB
	Where(query any, args ...any) *gorm.DB
	Table(name string, args ...any) *gorm.DB
	Delete(value any, conds ...any) *gorm.DB
	Transaction(fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error
	AutoMigrate(table ...any) error
	Unscoped() *gorm.DB
	FirstOrCreate(dest any, conds ...any) *gorm.DB
	First(dest any, conds ...any) *gorm.DB
	Find(dest any, conds ...any) *gorm.DB
	Count(count *int64) *gorm.DB
	Limit(limit int) *gorm.DB
	Offset(offset int) *gorm.DB
	Order(value interface{}) *gorm.DB
	Select(query interface{}, args ...interface{}) *gorm.DB
	Joins(query string, args ...interface{}) *gorm.DB
	Preload(query string, args ...interface{}) *gorm.DB
	Group(name string) *gorm.DB
	Having(query interface{}, args ...interface{}) *gorm.DB
	Scopes(funcs ...func(*gorm.DB) *gorm.DB) *gorm.DB
	Clauses(conds ...clause.Expression) *gorm.DB
	Raw(sql string, values ...any) *gorm.DB
	Exec(sql string, values ...any) *gorm.DB
	WithContext(ctx context.Context) *gorm.DB
	DB() (*sql.DB, error)
	Begin(opts ...*sql.TxOptions) *gorm.DB
	Commit() *gorm.DB
	Rollback() *gorm.DB
}
