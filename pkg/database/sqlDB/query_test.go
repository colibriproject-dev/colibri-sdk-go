package sqlDB

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/database/cacheDB"
)

func TestQueryWithoutInitialize(t *testing.T) {
	ctx := context.Background()
	sqlDBInstance = nil

	t.Run("Should return error when execute query one without params with db not initialized error", func(t *testing.T) {
		result, err := NewQuery[User](ctx, query_base+" LIMIT 1").One()

		assert.Error(t, err, db_not_initialized_error)
		assert.Nil(t, result)
	})

	t.Run("Should return error when execute query one with params with db not initialized error", func(t *testing.T) {
		result, err := NewQuery[User](ctx, query_base+" WHERE u.id = $1", 1).One()

		assert.Error(t, err, db_not_initialized_error)
		assert.Nil(t, result)
	})

	t.Run("Should return error when execute query many without params with db not initialized error", func(t *testing.T) {
		result, err := NewQuery[User](ctx, query_base).Many()

		assert.Error(t, err, db_not_initialized_error)
		assert.Nil(t, result)
	})

	t.Run("Should return error when execute query many with params with db not initialized error", func(t *testing.T) {
		result, err := NewQuery[User](ctx, query_base+" WHERE u.name = $1", "ADMIN USER").Many()

		assert.Error(t, err, db_not_initialized_error)
		assert.Nil(t, result)
	})
}

func TestQuery(t *testing.T) {
	ctx := context.Background()
	InitializeSqlDBTest()

	t.Run("Should return error when execute one without query", func(t *testing.T) {
		result, err := NewQuery[User](ctx, "").One()

		assert.Error(t, err, query_is_empty_error)
		assert.Nil(t, result)
	})

	t.Run("Should execute one without params", func(t *testing.T) {
		result, err := NewQuery[User](ctx, query_base+" LIMIT 1").One()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ADMIN USER", result.Name)

	})

	t.Run("Should execute one with params", func(t *testing.T) {
		result, err := NewQuery[User](ctx, query_base+" WHERE u.name = $1", "ADMIN USER").One()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ADMIN USER", result.Name)
	})

	t.Run("Should return error when execute many without query", func(t *testing.T) {
		result, err := NewQuery[User](ctx, "").Many()

		assert.Error(t, err, query_is_empty_error)
		assert.Nil(t, result)
	})

	t.Run("Should execute many without params", func(t *testing.T) {
		result, err := NewQuery[User](ctx, query_base).Many()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, "ADMIN USER", result[0].Name)
	})

	t.Run("Should execute many with params", func(t *testing.T) {
		result, err := NewQuery[User](ctx, query_base+" WHERE u.name = $1", "ADMIN USER").Many()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "ADMIN USER", result[0].Name)
	})
}

func TestQueryWithoutCacheDBInitialize(t *testing.T) {
	cache := cacheDB.NewCache[User]("TestQueryWithoutCacheDBInitialize", time.Hour)
	ctx := context.Background()
	InitializeSqlDBTest()

	t.Run("Should return error when one without params with cache", func(t *testing.T) {
		dbResult, dbErr := NewCachedQuery(ctx, cache, query_base+" LIMIT 1").One()
		cacheResult, cacheErr := cache.One(ctx)

		assert.NoError(t, dbErr)
		assert.NotNil(t, dbResult)
		assert.Equal(t, "ADMIN USER", dbResult.Name)
		assert.Error(t, cacheErr, "Cache not initialized")
		assert.Nil(t, cacheResult)
	})

	t.Run("Should return error when one with params with cache", func(t *testing.T) {
		dbResult, dbErr := NewCachedQuery(ctx, cache, query_base+" WHERE u.name = $1", "ADMIN USER").One()
		cacheResult, cacheErr := cache.One(ctx)

		assert.NoError(t, dbErr)
		assert.NotNil(t, dbResult)
		assert.Equal(t, "ADMIN USER", dbResult.Name)
		assert.Error(t, cacheErr, "Cache not initialized")
		assert.Nil(t, cacheResult)
	})

	t.Run("Should return error when many without params with cache", func(t *testing.T) {
		dbResult, dbErr := NewCachedQuery(ctx, cache, query_base).Many()
		cacheResult, cacheErr := cache.Many(ctx)

		assert.NoError(t, dbErr)
		assert.NotNil(t, dbResult)
		assert.Len(t, dbResult, 2)
		assert.Equal(t, "ADMIN USER", dbResult[0].Name)
		assert.Error(t, cacheErr, "Cache not initialized")
		assert.Nil(t, cacheResult)
	})

	t.Run("Should return error when many with params with cache", func(t *testing.T) {
		dbResult, dbErr := NewCachedQuery(ctx, cache, query_base+" WHERE u.name = $1", "ADMIN USER").Many()
		cacheResult, cacheErr := cache.Many(ctx)

		assert.NoError(t, dbErr)
		assert.NotNil(t, dbResult)
		assert.Len(t, dbResult, 1)
		assert.Equal(t, "ADMIN USER", dbResult[0].Name)
		assert.Error(t, cacheErr, "Cache not initialized")
		assert.Nil(t, cacheResult)
	})
}

func TestCachedQuery(t *testing.T) {
	InitializeSqlDBTest()
	test.InitializeCacheDBTest()
	cacheDB.Initialize()

	cache := cacheDB.NewCache[User]("TestCachedQuery", time.Hour)
	ctx := context.Background()

	t.Run("Should return error when one without query with cache", func(t *testing.T) {
		cacheInitialData, cacheInitialErr := cache.One(ctx)
		result, err := NewCachedQuery(ctx, cache, "").One()

		assert.NoError(t, cacheInitialErr)
		assert.Error(t, err, query_is_empty_error)
		assert.Nil(t, cacheInitialData)
		assert.Nil(t, result)
	})

	t.Run("Should execute one without params with cache", func(t *testing.T) {
		cacheInitialData, cacheInitialErr := cache.One(ctx)
		result, err := NewCachedQuery(ctx, cache, query_base+" LIMIT 1").One()
		cacheFinalData, cacheFinalErr := cache.One(ctx)
		cacheDelErr := cache.Del(ctx)

		assert.NoError(t, cacheInitialErr)
		assert.Nil(t, cacheInitialData)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ADMIN USER", result.Name)
		assert.NoError(t, cacheFinalErr)
		assert.NotNil(t, cacheFinalData)
		assert.Equal(t, "ADMIN USER", cacheFinalData.Name)
		assert.NoError(t, cacheDelErr)
	})

	t.Run("Should execute one with params with cache", func(t *testing.T) {
		cacheInitialData, cacheInitialErr := cache.One(ctx)
		result, err := NewCachedQuery(ctx, cache, query_base+" WHERE u.name = $1", "ADMIN USER").One()
		cacheFinalData, cacheFinalErr := cache.One(ctx)
		cacheDelErr := cache.Del(ctx)

		assert.NoError(t, cacheInitialErr)
		assert.Nil(t, cacheInitialData)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ADMIN USER", result.Name)
		assert.NoError(t, cacheFinalErr)
		assert.NotNil(t, cacheFinalData)
		assert.Equal(t, "ADMIN USER", cacheFinalData.Name)
		assert.NoError(t, cacheDelErr)
	})

	t.Run("Should return error when many without query with cache", func(t *testing.T) {
		cacheInitialData, cacheInitialErr := cache.One(ctx)
		result, err := NewCachedQuery(ctx, cache, "").Many()

		assert.NoError(t, cacheInitialErr)
		assert.Nil(t, cacheInitialData)
		assert.Error(t, err, query_is_empty_error)
		assert.Nil(t, result)
	})

	t.Run("Should execute many without params with cache", func(t *testing.T) {
		cacheInitialData, cacheInitialErr := cache.One(ctx)
		result, err := NewCachedQuery(ctx, cache, query_base).Many()
		cacheFinalData, cacheFinalErr := cache.Many(ctx)
		cacheDelErr := cache.Del(ctx)

		assert.NoError(t, cacheInitialErr)
		assert.Nil(t, cacheInitialData)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, "ADMIN USER", result[0].Name)
		assert.NoError(t, cacheFinalErr)
		assert.NotNil(t, cacheFinalData)
		assert.Len(t, result, 2)
		assert.Equal(t, "ADMIN USER", result[0].Name)
		assert.NoError(t, cacheDelErr)
	})

	t.Run("Should execute many with params with cache", func(t *testing.T) {
		cacheInitialData, cacheInitialErr := cache.One(ctx)
		result, err := NewCachedQuery(ctx, cache, query_base+" WHERE u.name = $1", "ADMIN USER").Many()
		cacheFinalData, cacheFinalErr := cache.Many(ctx)
		cacheDelErr := cache.Del(ctx)

		assert.NoError(t, cacheInitialErr)
		assert.Nil(t, cacheInitialData)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "ADMIN USER", result[0].Name)
		assert.NoError(t, cacheFinalErr)
		assert.NotNil(t, cacheFinalData)
		assert.Len(t, result, 1)
		assert.Equal(t, "ADMIN USER", result[0].Name)
		assert.NoError(t, cacheDelErr)
	})
}
