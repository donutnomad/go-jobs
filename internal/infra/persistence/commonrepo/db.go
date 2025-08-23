package commonrepo

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DB interface {
	Model(value any) (tx *gorm.DB)
	Create(value any) (tx *gorm.DB)
	Save(value any) (tx *gorm.DB)
	Where(query any, args ...any) (tx *gorm.DB)
	Table(name string, args ...any) (tx *gorm.DB)
	Delete(value any, conds ...any) (tx *gorm.DB)
	Transaction(fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error
	AutoMigrate(table ...any) error
	Unscoped() *gorm.DB
	FirstOrCreate(dest any, conds ...any) (tx *gorm.DB)
	First(dest any, conds ...any) (tx *gorm.DB)
	Find(dest any, conds ...any) (tx *gorm.DB)
	Scopes(funcs ...func(*gorm.DB) *gorm.DB) (tx *gorm.DB)
	Clauses(conds ...clause.Expression) (tx *gorm.DB)
	Raw(sql string, values ...any) (tx *gorm.DB)
	Exec(sql string, values ...any) (tx *gorm.DB)
	WithContext(ctx context.Context) *gorm.DB
	DB() (*sql.DB, error)

	Pluck(column string, dest any) *gorm.DB
	Count(count *int64) *gorm.DB
	Updates(values any) *gorm.DB
	Update(column string, value any) *gorm.DB
	UpdateColumn(column string, value any) *gorm.DB
	UpdateColumns(values any) *gorm.DB
	Take(dest any, conds ...any) *gorm.DB
	Last(dest any, conds ...any) *gorm.DB

	Distinct(args ...any) *gorm.DB
	Offset(offset int) *gorm.DB
	Limit(limit int) *gorm.DB
	Order(value any) *gorm.DB
}
