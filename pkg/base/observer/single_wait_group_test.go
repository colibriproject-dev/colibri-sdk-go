package observer

import (
	"sync"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/stretchr/testify/assert"
)

func TestGetWaitGroup(t *testing.T) {
	resetRunning(t)

	for i := 0; i < 10; i++ {
		GetWaitGroup()
	}
	wg := GetWaitGroup()

	if _, ok := any(wg).(*sync.WaitGroup); !ok {
		t.Errorf("GetWaitGroup was incorrect, got: %T, want: *sync.WaitGroup.", wg)
	}
}

func TestGetWaitGroupShouldReturnSameInstance(t *testing.T) {
	resetRunning(t)

	wg1 := GetWaitGroup()
	for i := 0; i < 10; i++ {
		GetWaitGroup()
	}
	wg2 := GetWaitGroup()

	if wg1 != wg2 {
		t.Errorf("GetWaitGroup does not return the same instance, got: %p and %p.", wg1, wg2)
	}
}

func TestWaitGroup(t *testing.T) {
	resetRunning(t)

	var work sync.WaitGroup
	for i := 0; i <= 50; i++ {
		work.Add(1)
		go func() {
			defer work.Done()
			process(1)
			process(1)
		}()
	}

	if WaitRunningTimeout() {
		t.Error("WaitRunningTimeout should return false, but it returned true.")
	}

	// the goroutines must not outlive the test: leaving them running leaks work into
	// whichever test the shuffle happens to schedule next
	work.Wait()
}

// TestAppIdleIgnoresModuleTasks is the item the split of the counters exists for: a consumer
// listener holds a module task for the whole life of the process, so counting it as
// application work would make a wait for that bucket never return.
func TestAppIdleIgnoresModuleTasks(t *testing.T) {
	resetRunning(t)

	AddModuleTask()
	t.Cleanup(DoneModuleTask)

	select {
	case <-appIdleSignal():
	case <-time.After(time.Second):
		t.Fatal("the application bucket blocked on a module task, which only ends during the shutdown")
	}
}

// TestWaitRunningTimeoutCoversTheDeprecatedWaitGroup pins that an application still using
// GetWaitGroup keeps being drained.
func TestWaitRunningTimeoutCoversTheDeprecatedWaitGroup(t *testing.T) {
	resetRunning(t)

	withDrainTimeout(t, 1)

	wg := GetWaitGroup()
	wg.Add(1)

	assert.True(t, WaitRunningTimeout(), "work on the deprecated WaitGroup must still be drained")

	wg.Done()
	assert.False(t, WaitRunningTimeout())
}

// TestWaitRunningTimeoutCoversModuleTasks is the other half: invisible to the application
// Wait, but still drained by the shutdown.
func TestWaitRunningTimeoutCoversModuleTasks(t *testing.T) {
	resetRunning(t)

	withDrainTimeout(t, 1)

	AddModuleTask()
	assert.True(t, WaitRunningTimeout(), "the drain must wait for the module tasks too")

	DoneModuleTask()
	assert.False(t, WaitRunningTimeout())
}

// TestWaitRunningTimeoutSharesOneBudget pins that the two waits are not one timeout each: a
// module task alone must not be able to hold the drain for longer than the whole budget.
func TestWaitRunningTimeoutSharesOneBudget(t *testing.T) {
	resetRunning(t)

	withDrainTimeout(t, 1)

	AddModuleTask()
	t.Cleanup(DoneModuleTask)

	start := time.Now()
	assert.True(t, WaitRunningTimeout())
	assert.Less(t, time.Since(start), 2*time.Second, "the drain must spend one budget, not one per bucket")
}

func TestDrainBudget(t *testing.T) {
	previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
	t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

	t.Run("Should return the requested fraction of the configured timeout", func(t *testing.T) {
		config.WAIT_GROUP_TIMEOUT_SECONDS = 90

		assert.Equal(t, 90*time.Second, DrainBudget(1, 1))
		assert.Equal(t, 45*time.Second, DrainBudget(1, 2))
	})

	t.Run("Should stay below the drain for every fraction under one", func(t *testing.T) {
		for _, seconds := range []int{1, 5, 30, 90, 600} {
			config.WAIT_GROUP_TIMEOUT_SECONDS = seconds

			assert.Less(t, DrainBudget(1, 2), DrainBudget(1, 1),
				"a component budget must expire before the drain gives up")
		}
	})

	t.Run("Should fall back to the default when the timeout is not positive", func(t *testing.T) {
		for _, seconds := range []int{0, -1} {
			config.WAIT_GROUP_TIMEOUT_SECONDS = seconds

			assert.Equal(t, defaultDrainTimeout, DrainBudget(1, 1))
			assert.Less(t, DrainBudget(1, 2), DrainBudget(1, 1))
		}
	})

	t.Run("Should return zero for a fraction that is not positive", func(t *testing.T) {
		config.WAIT_GROUP_TIMEOUT_SECONDS = 90

		assert.Zero(t, DrainBudget(0, 1))
		assert.Zero(t, DrainBudget(1, 0))
	})
}

func TestWaitRunningTimeout(t *testing.T) {
	resetRunning(t)

	withDrainTimeout(t, 2)

	var work sync.WaitGroup
	for i := 0; i <= 50; i++ {
		work.Add(1)
		go func() {
			defer work.Done()
			process(1)
			process(2)
			process(3)
		}()
	}
	// the work outlives the wait on purpose, so it has to be joined before the test returns
	t.Cleanup(work.Wait)

	time.Sleep(1 * time.Second)
	isTimeout := WaitRunningTimeout()
	assert.True(t, isTimeout)
}

func process(delaySeconds int) {
	AddRunning()
	defer DoneRunning()

	time.Sleep(time.Duration(delaySeconds) * time.Second)
}

func TestGetWaitGroupConcurrent(t *testing.T) {
	resetRunning(t)

	const goroutines = 32

	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	done.Add(goroutines)

	instances := make([]*sync.WaitGroup, goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer done.Done()
			start.Wait()
			instances[idx] = GetWaitGroup()
		}(i)
	}

	start.Done()
	done.Wait()

	for i := 1; i < goroutines; i++ {
		assert.Same(t, instances[0], instances[i])
	}
}

// TestModuleTaskCounterDoesNotUnderflow pins the guard on the counter: an unbalanced Done must
// not drive it negative and leave every later drain hanging.
func TestModuleTaskCounterDoesNotUnderflow(t *testing.T) {
	resetRunning(t)

	withDrainTimeout(t, 1)

	// drives the counter to -1
	DoneModuleTask()
	assert.False(t, WaitRunningTimeout(), "an unbalanced Done must not make the drain hang")

	// back to 1, so a real task is still waited for even after the underflow
	AddModuleTask()
	AddModuleTask()
	assert.True(t, WaitRunningTimeout())

	DoneModuleTask()
	assert.False(t, WaitRunningTimeout())
}

func withDrainTimeout(t *testing.T, seconds int) {
	t.Helper()

	previous := config.WAIT_GROUP_TIMEOUT_SECONDS
	config.WAIT_GROUP_TIMEOUT_SECONDS = seconds
	t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previous })
}
