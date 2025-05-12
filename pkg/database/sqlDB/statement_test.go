package sqlDB

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatementWithoutInitialize(t *testing.T) {
	ctx := context.Background()
	sqlDBInstance = nil

	t.Run("Should return error when execute statement", func(t *testing.T) {
		birth, _ := time.Parse("2006-01-02", "2021-11-22")
		user := User{123, "Usuário teste stmt", birth, Profile{100, "ADMIN"}}

		err := NewStatement(ctx, "INSERT INTO users VALUES ($1, $2, $3, $4)", user.Id, user.Name, user.Birthday, user.Profile.Id).Execute()

		assert.Error(t, err, dbNotInitializedError)
	})
}

func TestStatement(t *testing.T) {
	InitializeSqlDBTest()
	ctx := context.Background()

	t.Run("Should return error when execute statement without query", func(t *testing.T) {
		err := NewStatement(ctx, "").Execute()

		assert.Error(t, err, queryIsEmptyError)
	})

	t.Run("Should execute statement", func(t *testing.T) {
		birth, _ := time.Parse("2006-01-02", "2021-11-22")
		user := User{123, "Usuário teste stmt", birth, Profile{100, "ADMIN"}}

		statementErr := NewStatement(ctx, "INSERT INTO users VALUES ($1, $2, $3, $4)", user.Id, user.Name, user.Birthday, user.Profile.Id).Execute()
		result, err := NewQuery[User](ctx, query_base+" WHERE u.id = $1", user.Id).One()

		assert.NoError(t, statementErr)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, user.Id, result.Id)
		assert.Equal(t, user.Name, result.Name)
		assert.Equal(t, user.Birthday.Local(), result.Birthday.Local())
		assert.Equal(t, user.Profile, result.Profile)
	})
}
