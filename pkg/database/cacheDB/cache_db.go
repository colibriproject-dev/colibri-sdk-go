package cacheDB

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/go-redis/redis/v8"
	"github.com/newrelic/go-agent/v3/integrations/nrredis-v8"
)

type cacheDBObserver struct{}

var instance *redis.Client

// Initialize initializes the cache database connection.
//
// No parameters.
// No return values.
func Initialize() {
	if instance != nil {
		logging.Info(context.Background()).Msg("Cache database already connected")
		return
	}

	opts := &redis.Options{Addr: config.CACHE_URI, Password: config.CACHE_PASSWORD}

	redisClient := redis.NewClient(opts)
	redisClient.AddHook(nrredis.NewHook(opts))
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		logging.
			Fatal(context.Background()).
			Err(err).
			Msg("An error occurred while trying to connect to the cache database")
	}

	instance = redisClient
	observer.Attach(cacheDBObserver{})
	logging.Info(context.Background()).Msg("Cache database connected")
}

// Close closes the cache connection safely.
//
// No parameters.
// No return values.
func (o cacheDBObserver) Close() {
	logging.Info(context.Background()).Msg("waiting to safely close the cache connection")
	if observer.WaitRunningTimeout() {
		logging.Warn(context.Background()).Msg("WaitGroup timed out, forcing close the cache connection")
	}

	logging.Info(context.Background()).Msg("closing cache connection")
	if err := instance.Close(); err != nil {
		logging.
			Error(context.Background()).
			Err(err).
			Msg("error when closing cache connection")
	}
}
