package sqlDB

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
)

func TestMigrations(t *testing.T) {
	InitializeSqlDBTest()
	os.Setenv(config.ENV_SQL_DB_MIGRATION, "true")
	os.Setenv("MIGRATION_SOURCE_URL", fmt.Sprintf("%smigrations", test.DATABASE_ENVIRONMENT_PATH))

	t.Run("Should execute migration successfully and find users", func(t *testing.T) {
		const query = "SELECT u.id, u.name, u.birthday, p.id, p.name FROM users u INNER JOIN profiles p ON p.id = u.profile_id"

		result, err := NewQuery[User](context.Background(), query).Many()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
	})
}
