package observer

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

// TestMain initializes the logger once for the whole package. Every path under test logs, so
// a test that ran before the logger existed panicked on a nil handler. Leaving that to the
// individual tests made the package pass only in declaration order, which -shuffle breaks.
func TestMain(m *testing.M) {
	logging.Initialize()

	os.Exit(m.Run())
}

// resetRunning gives a test empty counters to start from and asserts it gave them back empty.
// They are package wide, so without it a test that leaks work decides whether the next one
// passes, and which test that is depends on the shuffle seed.
func resetRunning(t *testing.T) {
	t.Helper()

	clearRunning()
	t.Cleanup(func() {
		t.Helper()

		runningMu.Lock()
		app, module, wg := appRunning, moduleRunning, legacyWaitGroup
		runningMu.Unlock()

		if app != 0 || module != 0 {
			t.Errorf("test leaked running work: app=%d module=%d", app, module)
		}

		if !waitGroupIsEmpty(wg) {
			t.Error("test leaked application work on the deprecated WaitGroup")
		}

		clearRunning()
	})
}

// clearRunning zeroes the counters and hands back a fresh WaitGroup: a leaked Add can only be
// undone by replacing the instance.
func clearRunning() {
	runningMu.Lock()
	defer runningMu.Unlock()

	appRunning = 0
	moduleRunning = 0
	legacyWaitGroup = &sync.WaitGroup{}
	signalIdle()
}

// waitGroupIsEmpty reports whether the WaitGroup counter reached zero. A WaitGroup exposes no
// counter, so it is waited on with a short bound; the goroutine is left behind on purpose,
// the instance is discarded right after.
func waitGroupIsEmpty(wg *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)

		wg.Wait()
	}()

	select {
	case <-done:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}
