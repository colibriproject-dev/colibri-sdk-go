package sqlDB

import (
	"context"
	"database/sql"
	"errors"
)

// Statement is a struct for sql statement
type Statement struct {
	ctx   context.Context
	query string
	args  []any
}

// NewStatement creates a new pointer to Statement struct.
//
// ctx: the context.Context for the statement
// query: the query string for the statement
// params: variadic any for additional parameters
// Returns a pointer to Statement struct
func NewStatement(ctx context.Context, query string, params ...any) *Statement {
	return &Statement{ctx, query, params}
}

// Execute applies the statement in the database.
//
// No parameters.
// Returns an error.
func (s *Statement) Execute() error {
	return s.ExecuteInInstance(sqlDBInstance)
}

// ExecuteInInstance executes the statement in the provided database instance.
//
// instance: the sql database instance to execute the statement in.
// Returns an error.
func (s *Statement) ExecuteInInstance(instance *sql.DB) error {
	if err := s.validate(instance); err != nil {
		return err
	}

	stmt, err := s.createStatement(instance)
	if err != nil {
		return err
	}
	defer closer(stmt)

	if _, err = stmt.Exec(s.args...); err != nil {
		return err
	}

	return nil
}

// validate checks if the Statement instance is initialized and if the query is empty.
//
// No parameters.
// Returns an error.
func (s *Statement) validate(instance *sql.DB) error {
	if instance == nil {
		return errors.New(dbNotInitializedError)
	}

	if s.query == "" {
		return errors.New(queryIsEmptyError)
	}

	return nil
}

// createStatement creates a SQL statement for execution.
//
// No parameters.
// Returns a pointer to sql.Stmt and an error.
func (s *Statement) createStatement(instance *sql.DB) (*sql.Stmt, error) {
	if tx := s.ctx.Value(SqlTxContext); tx != nil {
		return tx.(*sql.Tx).PrepareContext(s.ctx, s.query)
	}

	return instance.PrepareContext(s.ctx, s.query)
}
