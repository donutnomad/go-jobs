package commonrepo

import (
	"context"

	"gorm.io/gorm"
)

type Transaction interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}

type dbContextKey struct{}

type DefaultRepo struct {
	db DB
}

func NewDefaultRepo(db DB) DefaultRepo {
	return DefaultRepo{db: db}
}

func (r *DefaultRepo) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.Db(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, dbContextKey{}, tx))
	})
}

func (r *DefaultRepo) dbFromContext(ctx context.Context) DB {
	db, ok := ctx.Value(dbContextKey{}).(DB)
	if !ok {
		return r.db
	}
	return db
}

func (r *DefaultRepo) Db(ctx context.Context) DB {
	return r.dbFromContext(ctx).WithContext(ctx)
}
