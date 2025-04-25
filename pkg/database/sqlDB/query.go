package sqlDB

import (
	"context"
	"database/sql"
	"errors"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/database/cacheDB"
)

// Query is a struct for sql query
type Query[T any] struct {
	ctx   context.Context
	cache *cacheDB.Cache[T]
	query string
	args  []any
}

// NewQuery create a new pointer to Query struct.
//
// ctx: the context.Context for the query
// query: the query string to execute
// params: variadic interface{} for additional parameters
// Returns a pointer to Query struct
func NewQuery[T any](ctx context.Context, query string, params ...any) *Query[T] {
	return &Query[T]{ctx, nil, query, params}
}

// NewCachedQuery create a new pointer to Query struct with cache.
//
// ctx: the context.Context for the query
// cache: the cacheDB.Cache to store the query result
// query: the query string to execute
// params: variadic interface{} for additional parameters
// Returns a pointer to Query struct
func NewCachedQuery[T any](ctx context.Context, cache *cacheDB.Cache[T], query string, params ...any) (q *Query[T]) {
	return &Query[T]{ctx, cache, query, params}
}

// Many returns a slice of T value.
//
// No parameters are required. Returns a slice of T value and an error.
func (q *Query[T]) Many() ([]T, error) {
	return q.ManyInInstance(sqlDBInstance)
}

// ManyInInstance retrieves multiple items of type T for the given SQL instance.
//
// instance: The *sql.DB instance to execute the query.
// Returns a slice of retrieved items of type T and an error.
func (q *Query[T]) ManyInInstance(instance *sql.DB) ([]T, error) {
	if err := q.validate(instance); err != nil {
		return nil, err
	}

	if q.cache == nil {
		return q.fetchMany(instance)
	}

	result, err := q.cache.Many(q.ctx)
	if result == nil || err != nil {
		return q.fetchMany(instance)
	}
	return result, nil
}

// fetchMany retrieves multiple items of type T for the given SQL instance.
//
// instance: The *sql.DB instance to execute the query.
// Returns a slice of retrieved items of type T and an error.
func (q *Query[T]) fetchMany(instance *sql.DB) ([]T, error) {
	rows, err := q.queryContext(instance)
	if err != nil {
		return nil, err
	}
	defer closer(rows)

	list, err := getDataList[T](rows)
	if err != nil {
		return nil, err
	}

	if q.cache != nil {
		q.cache.Set(q.ctx, list)
	}

	return list, nil
}

// One return a pointer of T value
//
// No parameters.
// Returns a pointer of T and an error.
func (q *Query[T]) One() (*T, error) {
	return q.OneInInstance(sqlDBInstance)
}

// OneInInstance retrieves a single item of type T for the given SQL instance.
//
// instance: The *sql.DB instance to execute the query.
// Returns a pointer of T and an error.
func (q *Query[T]) OneInInstance(instance *sql.DB) (*T, error) {
	if err := q.validate(instance); err != nil {
		return nil, err
	}

	if q.cache == nil {
		return q.fetchOne(instance)
	}

	result, err := q.cache.One(q.ctx)
	if result == nil || err != nil {
		return q.fetchOne(instance)
	}
	return result, nil
}

// fetchOne retrieves a single item of type T for the given SQL instance.
//
// instance: The *sql.DB instance to execute the query.
// Returns a pointer of T and an error.
func (q *Query[T]) fetchOne(instance *sql.DB) (*T, error) {
	model := new(T)
	if err := q.queryRowContext(instance).Scan(reflectCols(model)...); err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == sql.ErrNoRows {
		return nil, nil
	}

	if q.cache != nil {
		q.cache.Set(q.ctx, model)
	}

	return model, nil
}

// validate checks if the Query instance is initialized and if the query is empty.
//
// instance: The *sql.DB instance to execute the query.
// Returns an error.
func (q *Query[T]) validate(instance *sql.DB) error {
	if instance == nil {
		return errors.New(db_not_initialized_error)
	}

	if q.query == "" {
		return errors.New(query_is_empty_error)
	}

	return nil
}

// queryContext executes a query on the provided SQL instance.
//
// instance: The *sql.DB instance to execute the query.
// Returns the resulting rows and an error.
func (q *Query[T]) queryContext(instance *sql.DB) (*sql.Rows, error) {
	if tx := q.ctx.Value(SqlTxContext); tx != nil {
		return tx.(*sql.Tx).QueryContext(q.ctx, q.query, q.args...)
	}

	return instance.QueryContext(q.ctx, q.query, q.args...)
}

// queryRowContext executes a query on the provided SQL instance and returns a single row.
//
// instance: The *sql.DB instance to execute the query.
// Returns the resulting row.
func (q *Query[T]) queryRowContext(instance *sql.DB) *sql.Row {
	if tx := q.ctx.Value(SqlTxContext); tx != nil {
		return tx.(*sql.Tx).QueryRowContext(q.ctx, q.query, q.args...)
	}

	return instance.QueryRowContext(q.ctx, q.query, q.args...)
}
