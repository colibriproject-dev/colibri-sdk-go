package cacheDB

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	t.Run("Should nil if not initialize", func(t *testing.T) {
		assert.Nil(t, instance)
	})

	t.Run("Should initialize", func(t *testing.T) {
		test.InitializeCacheDBTest()

		Initialize()

		assert.NotNil(t, instance)
	})
}
