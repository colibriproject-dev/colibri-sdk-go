package cacheDB

import (
	"context"
	"testing"
	"time"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
)

type userCached struct {
	Id   int
	Name string
}

func TestCacheNotInitializedValidation(t *testing.T) {
	instance = nil

	t.Run("Should return error when cacheDB is not initialized", func(t *testing.T) {
		cache := NewCache[userCached]("cache-test", time.Hour)

		result, err := cache.Many(context.Background())

		assert.NotNil(t, err)
		assert.Nil(t, result)
	})
}

func TestCache(t *testing.T) {
	test.InitializeCacheDBTest()
	Initialize()

	ctx := context.Background()
	cacheWithEmptyName := Cache[userCached]{}
	cache := NewCache[userCached]("cache-test", time.Hour)
	expected := []userCached{
		{Id: 1, Name: "User 1"},
		{Id: 2, Name: "User 2"},
		{Id: 3, Name: "User 3"},
	}

	t.Run("Should return a new point of cache", func(t *testing.T) {
		result := NewCache[userCached]("cache-test", time.Hour)

		assert.NotNil(t, result)
		assert.NotNil(t, result.name)
		assert.NotNil(t, result.ttl)
	})

	t.Run("Should return error when cache name is empty value on call many", func(t *testing.T) {
		result, err := cacheWithEmptyName.Many(ctx)

		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("Should return error when cache name is empty value on call one", func(t *testing.T) {
		result, err := cacheWithEmptyName.One(ctx)

		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("Should return error when cache name is empty value on call set", func(t *testing.T) {
		err := cacheWithEmptyName.Set(ctx, cacheWithEmptyName)

		assert.NotNil(t, err)
	})

	t.Run("Should return error when cache name is empty value on call del", func(t *testing.T) {
		err := cacheWithEmptyName.Del(ctx)

		assert.NotNil(t, err)
	})

	t.Run("Should return error when occurred error in json unmarshal on set data in cache", func(t *testing.T) {
		invalid := map[string]interface{}{
			"invalid": make(chan int),
		}

		err := cache.Set(ctx, invalid)

		assert.NotNil(t, err)
	})

	t.Run("Should set many data in cache", func(t *testing.T) {
		setErr := cache.Set(ctx, expected)
		result, err := cache.Many(ctx)

		assert.Nil(t, setErr)
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 3)
		assert.Equal(t, expected, result)
	})

	t.Run("Should set one data in cache", func(t *testing.T) {
		setErr := cache.Set(ctx, expected[0])
		result, err := cache.One(ctx)

		assert.Nil(t, setErr)
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected[0], *result)
	})

	t.Run("Should del data in cache", func(t *testing.T) {
		setErr := cache.Set(ctx, expected)
		manyInitialResult, manyInitialErr := cache.Many(ctx)
		delErr := cache.Del(ctx)
		manyFinalResult, manyFinalErr := cache.Many(ctx)

		assert.NoError(t, setErr)
		assert.NoError(t, manyInitialErr)
		assert.NotNil(t, manyInitialResult)
		assert.Len(t, manyInitialResult, 3)
		assert.Equal(t, expected, manyInitialResult)
		assert.NoError(t, delErr)
		assert.NoError(t, manyFinalErr)
		assert.Nil(t, manyFinalResult)
	})
}
