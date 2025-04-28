package sqlDB

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/transaction"
)

// SqlTxContextKey is the type of the context key for the transaction.
type SqlTxContextKey string

const (
	SqlTxContext SqlTxContextKey = "SqlTxContext"

	transactionIsolationWarnMsg string = "transaction isolation just use first parameter, others will be ignored"
	transactionRollbackErrorMsg string = "error when executing transaction rollback: %v: %w"
	transactionCommitErrorMsg   string = "could not commit transaction: %w"
	transactionStartErrorMsg    string = "could not start database transaction: %v"
)

// sqlTransaction implements a transaction.Transaction
type sqlTransaction struct {
	isolation sql.IsolationLevel
}

// NewTransaction creates a new sqlTransaction implementing the transaction.Transaction interface.
//
// It takes an optional variable number of sql.IsolationLevel parameters and returns a transaction.Transaction.
func NewTransaction(isolation ...sql.IsolationLevel) transaction.Transaction {
	isolationLevel := sql.LevelDefault
	if len(isolation) == 1 {
		isolationLevel = isolation[0]
	} else if len(isolation) > 1 {
		isolationLevel = isolation[0]
		logging.Warn(context.Background()).Msg(transactionIsolationWarnMsg)
	}

	return &sqlTransaction{isolation: isolationLevel}
}

// Execute executes a transactional SQL.
//
// ctx: The context for the transaction.
// fn: The function to be executed.
// Returns an error.
func (t *sqlTransaction) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return t.ExecuteInInstance(ctx, sqlDBInstance, fn)
}

// ExecuteInInstance executes a transaction in a specific database instance.
//
// ctx: The context for the transaction.
// instance: The specific database instance where the transaction will be executed.
// fn: The function to be executed as part of the transaction.
// Returns an error.
func (t *sqlTransaction) ExecuteInInstance(ctx context.Context, instance *sql.DB, fn func(ctx context.Context) error) error {
	transaction, transactionChannel, err := t.beginTransaction(ctx, instance)
	if err != nil {
		return err
	}
	defer close(transactionChannel)

	ctx = context.WithValue(ctx, SqlTxContext, transaction)

	if err = fn(ctx); err != nil {
		if rbErr := transaction.Rollback(); rbErr != nil {
			fErr := fmt.Errorf(transactionRollbackErrorMsg, err, rbErr)
			logging.Error(ctx).Err(fErr)
			transactionChannel <- fErr
			return fErr
		}

		logging.Error(ctx).Err(err)
		transactionChannel <- err
		return err
	}

	if err = transaction.Commit(); err != nil {
		fErr := fmt.Errorf(transactionCommitErrorMsg, err)
		logging.Error(ctx).Err(fErr)
		transactionChannel <- fErr
		return fErr
	}

	return nil
}

// beginTransaction starts a new database transaction.
//
// ctx: The context for the transaction.
// instance: The specific database instance for the transaction.
// Returns the transaction, a channel for errors, and an error.
func (t *sqlTransaction) beginTransaction(ctx context.Context, instance *sql.DB) (*sql.Tx, chan error, error) {
	transaction, err := instance.BeginTx(ctx, &sql.TxOptions{Isolation: t.isolation})

	if err != nil {
		fErr := fmt.Errorf(transactionStartErrorMsg, err)
		logging.Error(ctx).Err(fErr)
		return nil, nil, fErr
	}

	return transaction, make(chan error, 1), nil
}
