package cloud

import (
	"testing"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	t.Run("Should nil if not initialize", func(t *testing.T) {
		assert.Nil(t, instance)
	})

	t.Run("Should initialize AWS with local enviroment", func(t *testing.T) {
		config.ENVIRONMENT = config.ENVIRONMENT_DEVELOPMENT
		config.CLOUD = config.CLOUD_AWS

		Initialize()

		assert.NotNil(t, instance)
		assert.NotNil(t, instance.aws)
		assert.Nil(t, instance.firebase)
		assert.NotNil(t, GetAwsSession())
	})

	t.Run("Should initialize AWS with cloud enviroment", func(t *testing.T) {
		config.ENVIRONMENT = config.ENVIRONMENT_PRODUCTION
		config.CLOUD = config.CLOUD_AWS

		Initialize()

		assert.NotNil(t, instance)
		assert.NotNil(t, instance.aws)
		assert.Nil(t, instance.firebase)
		assert.NotNil(t, GetAwsSession())
	})

	t.Run("Should initialize FIREBASE with local enviroment", func(t *testing.T) {
		config.ENVIRONMENT = config.ENVIRONMENT_DEVELOPMENT
		config.CLOUD = config.CLOUD_FIREBASE

		Initialize()

		assert.NotNil(t, instance)
		assert.Nil(t, instance.aws)
		assert.NotNil(t, instance.firebase)
		assert.NotNil(t, GetFirebaseSession())
	})

	t.Run("Should initialize FIREBASE with cloud enviroment", func(t *testing.T) {
		config.ENVIRONMENT = config.ENVIRONMENT_PRODUCTION
		config.CLOUD = config.CLOUD_FIREBASE

		Initialize()

		assert.NotNil(t, instance)
		assert.Nil(t, instance.aws)
		assert.NotNil(t, instance.firebase)
		assert.NotNil(t, GetFirebaseSession())
	})

	t.Run("Should initialize GCP with local enviroment", func(t *testing.T) {
		config.ENVIRONMENT = config.ENVIRONMENT_DEVELOPMENT
		config.CLOUD = config.CLOUD_GCP

		Initialize()

		assert.NotNil(t, instance)
		assert.Nil(t, instance.aws)
		assert.Nil(t, instance.firebase)
	})

	t.Run("Should initialize GCP with cloud enviroment", func(t *testing.T) {
		config.ENVIRONMENT = config.ENVIRONMENT_PRODUCTION
		config.CLOUD = config.CLOUD_GCP

		Initialize()

		assert.NotNil(t, instance)
		assert.Nil(t, instance.aws)
		assert.Nil(t, instance.firebase)
	})
}
