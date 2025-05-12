package sqlDB

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSqlTransactionWithoutInitialize(t *testing.T) {
	sqlDBInstance = nil
	ctx := context.Background()

	t.Run("Should return error when instance is nil", func(t *testing.T) {
		err := NewStatement(ctx, "", "Contact Name 1", "em@il.com").Execute()

		assert.Error(t, err, dbNotInitializedError)
	})
}

func TestSqlTransaction(t *testing.T) {
	type contact struct {
		Name  string
		Email string
	}

	ctx := context.Background()
	InitializeSqlDBTest()

	t.Run("Should return error when query is nil", func(t *testing.T) {
		err := NewStatement(ctx, "", "Contact Name 1", "em@il.com").Execute()

		assert.Error(t, err, queryIsEmptyError)
	})

	t.Run("Should execute transaction and commit", func(t *testing.T) {
		transaction := NewTransaction()
		err := transaction.Execute(ctx, func(ctx context.Context) error {
			insertContact1 := "INSERT INTO contacts (name, email) VALUES ($1, $2) "
			stmt1 := NewStatement(ctx, insertContact1, "Contact Name 1 with Commit", "email1@email.com")
			if err := stmt1.Execute(); err != nil {
				return err
			}

			insertContact2 := "INSERT INTO contacts (name, email) VALUES ($1, $2) "
			stmt2 := NewStatement(ctx, insertContact2, "Contact Name 2 with commit", "email2@email.com")
			if err := stmt2.Execute(); err != nil {
				return err
			}

			return nil
		})
		query1Result, query1Err := NewQuery[contact](ctx, "SELECT name, email FROM contacts WHERE email = $1", "email1@email.com").One()
		query2Result, query2Err := NewQuery[contact](ctx, "SELECT name, email FROM contacts WHERE email = $1", "email2@email.com").One()

		assert.NoError(t, err)
		assert.NoError(t, query1Err)
		assert.NotNil(t, query1Result)
		assert.Equal(t, "email1@email.com", query1Result.Email)
		assert.NoError(t, query2Err)
		assert.NotNil(t, query2Result)
		assert.Equal(t, "email2@email.com", query2Result.Email)
	})

	t.Run("Should execute transaction with fail and rollback", func(t *testing.T) {
		transaction := NewTransaction()
		query1Result, query1Err := NewQuery[contact](ctx, "SELECT name, email FROM contacts WHERE email = $1", "email2@email.com").One()
		err := transaction.Execute(ctx, func(ctx context.Context) error {
			insertContact1 := "INSERT INTO contacts (name, email) VALUES ($1, $2) "
			stmt1 := NewStatement(ctx, insertContact1, "Contact Name 1 with fail", "email1-with-fail@email.com")
			if err := stmt1.Execute(); err != nil {
				return err
			}

			insertContact2 := "INSERT INTO contacts (name, email) VALUES ($1, $2) "
			stmt2 := NewStatement(ctx, insertContact2, "Contact Name 2 with fail", "email2@email.com")
			if err := stmt2.Execute(); err != nil {
				return err
			}

			return nil
		})
		query2Result, query2Err := NewQuery[contact](ctx, "SELECT name, email FROM contacts WHERE email = $1", "email1-with-fail@email.com").One()

		assert.NoError(t, query1Err)
		assert.NotNil(t, query1Result)
		assert.Error(t, err)
		assert.NoError(t, query2Err)
		assert.Nil(t, query2Result)
	})
}

func TestSqlTransactionIsolationLevel(t *testing.T) {
	t.Run("Should return isolation level default", func(t *testing.T) {
		tx := NewTransaction()

		assert.NotNil(t, tx)
		assert.Equal(t, sql.LevelDefault, tx.(*sqlTransaction).isolation)
	})

	t.Run("Should return isolation level serializable", func(t *testing.T) {
		tx := NewTransaction(sql.LevelSerializable)

		assert.NotNil(t, tx)
		assert.Equal(t, sql.LevelSerializable, tx.(*sqlTransaction).isolation)
	})

	t.Run("Should return multiple isolations, only first is used", func(t *testing.T) {
		tx := NewTransaction(sql.LevelLinearizable, sql.LevelSerializable)

		assert.NotNil(t, tx)
		assert.Equal(t, sql.LevelLinearizable, tx.(*sqlTransaction).isolation)
	})
}
