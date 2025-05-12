package cacheDB

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
)

const (
	errRedisNil   string = "redis: nil"
	errRedisMoved string = "MOVED"
)

// Cache struct
type Cache[T any] struct {
	name string
	ttl  time.Duration
}

// NewCache creates a new pointer to Cache struct.
//
// Parameters:
// - name: a string representing the name of the cache.
// - ttl: a time.Duration representing the time to live for the cache items.
// Returns a pointer to Cache[T].
func NewCache[T any](name string, ttl time.Duration) *Cache[T] {
	return &Cache[T]{name, ttl}
}

// Many retrieves multiple items of type T from the cache.
//
// ctx: The context for the cache operation.
// Returns a slice of retrieved items of type T and an error.
func (c *Cache[T]) Many(ctx context.Context) ([]T, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	result, err := c.get(ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	list := make([]T, 0)
	if err = json.Unmarshal(result, &list); err != nil {
		return nil, err
	}

	return list, nil
}

// One retrieves a single item of type T from the cache.
//
// ctx: The context for the cache operation.
// Returns a pointer to the retrieved item of type T and an error.
func (c *Cache[T]) One(ctx context.Context) (*T, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	result, err := c.get(ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	model := new(T)
	if err = json.Unmarshal(result, &model); err != nil {
		return nil, err
	}

	return model, nil
}

// Set save data in cacheDB.
//
// ctx: The context for the cache operation.
// data: The data to be saved in the cache.
// Returns an error.
func (c *Cache[T]) Set(ctx context.Context, data any) error {
	if err := c.validate(); err != nil {
		return err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.set(ctx, jsonData)
}

// Del delete data in cachedDB.
//
// ctx: The context for the cache operation.
// Returns an error.
func (c *Cache[T]) Del(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return err
	}

	return c.del(ctx)
}

// validate checks if the cache is initialized and has a name.
//
// No parameters.
// Returns an error.
func (c *Cache[T]) validate() error {
	if instance == nil {
		return errors.New("cache not initialized")
	}

	if c.name == "" {
		return errors.New("cache without name")
	}

	return nil
}

// getNamePrefixed returns a string with the prefixed name using the application name and cache name.
//
// No parameters.
// Returns a string.
func (c *Cache[T]) getNamePrefixed() string {
	return fmt.Sprintf("%s::%s", config.APP_NAME, c.name)
}

// isErrRedisMoved checks if the error contains the string "MOVED".
//
// Parameter:
// - err: The error to check.
// Return type: bool
func (c *Cache[T]) isErrRedisMoved(err error) bool {
	return strings.Contains(err.Error(), errRedisMoved)
}

// reconnectInstanceAfterError updates the address of the instance based on the last element of the error message.
//
// Parameter:
// - err: The error that triggered the reconnection.
func (c *Cache[T]) reconnectInstanceAfterError(err error) {
	movedSetInfo := strings.Split(err.Error(), " ")
	instance.Options().Addr = movedSetInfo[len(movedSetInfo)-1]
}

// get retrieves data from the cache and handles errors including redis MOVED error.
//
// ctx: The context for the cache operation.
// Returns a byte slice and an error.
func (c *Cache[T]) get(ctx context.Context) ([]byte, error) {
	for {
		result, err := instance.Get(ctx, c.getNamePrefixed()).Bytes()
		if err != nil {
			if err.Error() == errRedisNil {
				return nil, nil
			} else if c.isErrRedisMoved(err) {
				c.reconnectInstanceAfterError(err)
				continue
			} else {
				return nil, err
			}
		}
		return result, nil
	}
}

// set saves data in the cacheDB.
//
// ctx: The context for the cache operation.
// data: The data to be saved in the cache.
// Returns an error.
func (c *Cache[T]) set(ctx context.Context, data []byte) error {
	for {
		err := instance.Set(ctx, c.getNamePrefixed(), data, c.ttl).Err()
		if err != nil {
			if c.isErrRedisMoved(err) {
				c.reconnectInstanceAfterError(err)
				continue
			} else {
				return err
			}
		}
		return nil
	}
}

// del deletes data in cachedDB.
//
// ctx: The context for the cache operation.
// Returns an error.
func (c *Cache[T]) del(ctx context.Context) error {
	for {
		err := instance.Del(ctx, c.getNamePrefixed()).Err()
		if err != nil {
			if c.isErrRedisMoved(err) {
				c.reconnectInstanceAfterError(err)
				continue
			} else {
				return err
			}
		}
		return nil
	}
}
