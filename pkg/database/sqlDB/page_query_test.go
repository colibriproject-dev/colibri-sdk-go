package sqlDB

import (
	"context"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/types"
	"github.com/stretchr/testify/assert"
)

func TestPageQueryWithoutInitialize(t *testing.T) {
	sqlDBInstance = nil

	t.Run("Should return error when execute page query with db not initialized error", func(t *testing.T) {
		page := types.NewPageRequest(1, 1, []types.Sort{
			{Direction: types.DESC, Field: "name"},
			{Direction: types.ASC, Field: "birthday"},
		})

		result, err := NewPageQuery[User](context.Background(), page, query_base).Execute()

		assert.Error(t, err, dbNotInitializedError)
		assert.Nil(t, result)
	})
}

func TestPageQuery(t *testing.T) {
	InitializeSqlDBTest()
	ctx := context.Background()
	page := types.NewPageRequest(1, 1, []types.Sort{
		{Direction: types.DESC, Field: "u.name"},
		{Direction: types.ASC, Field: "u.birthday"},
	})

	t.Run("Should return error when execute page query without page info", func(t *testing.T) {
		result, err := NewPageQuery[User](ctx, nil, query_base).Execute()

		assert.Error(t, err, pageIsEmptyError)
		assert.Nil(t, result)
	})

	t.Run("Should return error when execute page query without query", func(t *testing.T) {
		result, err := NewPageQuery[User](ctx, page, "").Execute()

		assert.Error(t, err, queryIsEmptyError)
		assert.Nil(t, result)
	})

	t.Run("Should execute page query", func(t *testing.T) {
		result, err := NewPageQuery[User](ctx, page, query_base).Execute()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "OTHER USER", result.Items[0].Name)
		assert.Equal(t, uint64(2), result.TotalItems)
	})
}
