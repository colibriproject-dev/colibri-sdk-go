package cacheDB

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type cacheDBObserver struct{}

var instance *redis.Client

// Initialize starts the connection with the cache database.
func Initialize() {
	if instance != nil {
		logging.Info(context.Background()).Msg("Cache database already connected")
		return
	}

	opts := &redis.Options{Addr: config.CACHE_URI, Password: config.CACHE_PASSWORD}

	redisClient := redis.NewClient(opts)

	if monitoring.UseOTELMonitoring() {
		if err := redisotel.InstrumentTracing(redisClient); err != nil {
			logging.Fatal(context.Background()).Err(err).Msg("An error occurred while trying to instrument tracing")
		}
	}

	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		logging.
			Fatal(context.Background()).
			Err(err).
			Msg("An error occurred while trying to connect to the cache database")
	}

	instance = redisClient
	observer.AttachWithPriority(cacheDBObserver{}, observer.PriorityDefault)
	logging.Info(context.Background()).Msg("Cache database connected")
}

// Close closes the cache connection safely. It runs in the closing phase of the graceful
// shutdown, so the work that reads and writes the cache has already been drained.
//
// No parameters.
// No return values.
func (o cacheDBObserver) Close() {
	logging.Info(context.Background()).Msg("closing cache connection")
	if err := instance.Close(); err != nil {
		logging.
			Error(context.Background()).
			Err(err).
			Msg("error when closing cache connection")
	}
}
