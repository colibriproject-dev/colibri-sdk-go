package sqlDB

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
)

const (
	query_base = "SELECT u.id, u.name, u.birthday, p.id, p.name FROM users u JOIN profiles p ON u.profile_id = p.id"
)

type Profile struct {
	Id   int
	Name string
}

type User struct {
	Id       int
	Name     string
	Birthday time.Time
	Profile  Profile
}

type Dog struct {
	ID              uint
	Name            string
	Characteristics []string
}

var (
	open = true
)

type closeable struct{}
type closeableError struct{}

func (c closeable) Close() error {
	open = false
	return nil
}

func (c closeableError) Close() error {
	return errors.New("error")
}

func TestCloser(t *testing.T) {
	t.Run("Should close the database observer", func(t *testing.T) {
		c := closeable{}
		assert.NotNil(t, c)
		assert.True(t, open)

		closer(c)
		assert.False(t, open)
	})
}

func TestCloserWithTimeout(t *testing.T) {
	t.Run("Should close the database observer with timed out", func(t *testing.T) {
		open = true
		c := closeable{}
		assert.NotNil(t, c)
		assert.True(t, open)
		config.WAIT_GROUP_TIMEOUT_SECONDS = 1
		observer.GetWaitGroup().Add(1)
		defer observer.GetWaitGroup().Done()

		closer(c)
		assert.False(t, open)
	})

	t.Run("Should return an error to close the database", func(t *testing.T) {
		open = true

		closer(closeableError{})

		assert.True(t, open)
	})
}

func InitializeSqlDBTest() {
	ctx := context.Background()
	basePath := test.MountAbsolutPath(test.DATABASE_ENVIRONMENT_PATH)

	test.InitializeSqlDBTest()
	pc := test.UsePostgresContainer(ctx)

	if err := pc.Dataset(basePath, "schema.sql"); err != nil {
		logging.Fatal(ctx).Err(err)
	}

	datasets := []string{"clear-database.sql", "add-users.sql", "add-contacts.sql", "add-dogs.sql"}
	pc.Dataset(basePath, datasets...)

	Initialize()
}
