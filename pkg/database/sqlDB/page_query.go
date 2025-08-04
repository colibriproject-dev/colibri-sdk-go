package sqlDB

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/types"
)

const (
	pageTotalPostgresQuery string = "SELECT COUNT(tb.*) FROM (%s) tb"
	pageDataPostgresQuery  string = "%s ORDER BY %s LIMIT %d OFFSET %d"
)

// PageQuery is a struct for sql page query
type PageQuery[T any] struct {
	ctx   context.Context
	page  *types.PageRequest
	query string
	args  []any
}

// NewPageQuery creates a new pointer to PageQuery struct.
//
// ctx: the context.Context for the query
// page: the types.PageRequest for the query
// query: the query string to execute
// params: variadic any for additional parameters
// Returns a pointer to PageQuery struct
func NewPageQuery[T any](ctx context.Context, page *types.PageRequest, query string, params ...any) *PageQuery[T] {
	return &PageQuery[T]{ctx, page, query, params}
}

// Execute returns a pointer of a page type with slice of T data.
//
// No parameters.
// Returns a pointer to PageQuery struct and an error.
func (q *PageQuery[T]) Execute() (*types.Page[T], error) {
	return q.ExecuteInInstance(sqlDBInstance)
}

// ExecuteInInstance executes the page query in the given database instance.
//
// Parameters:
// - instance: the database instance to execute the query in.
// Returns a Page of type T and an error.
func (q *PageQuery[T]) ExecuteInInstance(instance *sql.DB) (*types.Page[T], error) {
	if err := q.validate(instance); err != nil {
		return nil, err
	}

	var result types.Page[T]
	var err error
	result.TotalItems, err = q.pageTotal(instance)
	if err != nil {
		return nil, err
	}

	result.Items, err = q.pageData(instance)
	return &result, err
}

// pageTotal calculates the total number of records in the query result.
//
// Parameters:
// - instance: the database instance to execute the query in.
// Returns a uint64 representing the total number of records and an error.
func (q *PageQuery[T]) pageTotal(instance *sql.DB) (uint64, error) {
	query := fmt.Sprintf(pageTotalPostgresQuery, q.query)

	var result uint64
	err := q.queryRowContext(instance, query).Scan(&result)
	return result, err
}

// pageData retrieves data for the page query from the given database instance.
//
// Parameters:
// - instance: the database instance to retrieve data from.
// Returns a slice of type T and an error.
func (q *PageQuery[T]) pageData(instance *sql.DB) ([]T, error) {
	query := fmt.Sprintf(pageDataPostgresQuery, q.query, q.page.GetOrder(), q.page.Size, ((q.page.Page - 1) * q.page.Size))

	rows, err := q.queryContext(instance, query)
	if err != nil {
		return nil, err
	}
	defer closer(rows)

	return getDataList[T](rows)
}

// validate checks if the PageQuery instance is initialized, if the page is empty, and if the query is empty.
//
// instance: the database instance to validate against
// Returns an error.
func (q *PageQuery[T]) validate(instance *sql.DB) error {
	if instance == nil {
		return errors.New(dbNotInitializedError)
	}

	if q.page == nil {
		return errors.New(pageIsEmptyError)
	}

	if q.query == "" {
		return errors.New(queryIsEmptyError)
	}

	return nil
}

// queryContext executes a query on the provided SQL instance.
//
// Parameters:
// - instance: The *sql.DB instance to execute the query.
// - query: The SQL query string to execute.
// Returns the resulting rows and an error.
func (q *PageQuery[T]) queryContext(instance *sql.DB, query string) (*sql.Rows, error) {
	if tx := q.ctx.Value(SqlTxContext); tx != nil {
		return tx.(*sql.Tx).QueryContext(q.ctx, query, q.args...)
	}

	return instance.QueryContext(q.ctx, query, q.args...)
}

// queryRowContext executes a query on the provided SQL instance and returns a single row.
//
// Parameters:
// - instance: The *sql.DB instance to execute the query.
// - query: The SQL query string to execute.
// Returns the resulting row.
func (q *PageQuery[T]) queryRowContext(instance *sql.DB, query string) *sql.Row {
	if tx := q.ctx.Value(SqlTxContext); tx != nil {
		return tx.(*sql.Tx).QueryRowContext(q.ctx, query, q.args...)
	}

	return instance.QueryRowContext(q.ctx, query, q.args...)
}
