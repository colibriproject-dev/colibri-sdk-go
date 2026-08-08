package sqlDB

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
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

// notConnectingConnector builds a *sql.DB that never opens a connection, so the observer can
// be closed without a database behind it.
type notConnectingConnector struct{}

func (notConnectingConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errors.New("not connected")
}

func (notConnectingConnector) Driver() driver.Driver {
	return nil
}

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

func TestCloserWithError(t *testing.T) {
	t.Run("Should return an error to close the database", func(t *testing.T) {
		open = true

		closer(closeableError{})

		assert.True(t, open)
	})
}

func TestSqlDBObserverClose(t *testing.T) {
	t.Run("Should close without draining the work, which the shutdown pipeline already did", func(t *testing.T) {
		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 30
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		// work that outlives the close, the way a shutdown reaching the closing phase after
		// the drain timed out leaves it
		observer.AddRunning()
		t.Cleanup(observer.DoneRunning)

		o := sqlDBObserver{name: "test", instance: sql.OpenDB(notConnectingConnector{})}

		returned := make(chan struct{})
		go func() {
			defer close(returned)
			o.Close()
		}()

		select {
		case <-returned:
		case <-time.After(time.Second):
			t.Fatal("Close waited for the running work, which belongs to the drain phase")
		}
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
