package observer

import (
	"sync"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/stretchr/testify/assert"
)

func TestGetWaitGroup(t *testing.T) {
	logging.Initialize()

	for i := 0; i < 10; i++ {
		GetWaitGroup()
	}
	wg := GetWaitGroup()

	if _, ok := any(wg).(*sync.WaitGroup); !ok {
		t.Errorf("GetWaitGroup was incorrect, got: %T, want: *sync.WaitGroup.", wg)
	}
}

func TestGetWaitGroupShouldReturnSameInstance(t *testing.T) {
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
	for i := 0; i <= 50; i++ {
		go func() {
			process(1)
			process(1)
		}()
	}
	wg1 := GetWaitGroup()

	if WaitRunningTimeout() {
		t.Error("WaitRunningTimeout should return false, but it returned true.")
	}

	if _, ok := any(wg1).(*sync.WaitGroup); !ok {
		t.Errorf("GetWaitGroup was incorrect, got: %T, want: *sync.WaitGroup.", wg1)
	}

	wg2 := GetWaitGroup()
	if wg1 != wg2 {
		t.Errorf("GetWaitGroup does not return the same instance, got: %p and %p.", wg1, wg2)
	}
}

func TestWaitRunningTimeout(t *testing.T) {
	// restored so a repeated run does not start with the shortened timeout, which the tests
	// waiting for work to complete would then give up on
	previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
	t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

	config.WAIT_GROUP_TIMEOUT_SECONDS = 2
	for i := 0; i <= 50; i++ {
		go func() {
			process(1)
			process(2)
			process(3)
		}()
	}

	time.Sleep(1 * time.Second)
	isTimeout := WaitRunningTimeout()
	assert.True(t, isTimeout)
}

func process(delaySeconds int) {
	AddRunning()
	defer DoneRunning()
	time.Sleep(time.Duration(delaySeconds) * time.Second)
}

func TestWaitRunningTimeoutIsReusableAfterTimeout(t *testing.T) {
	previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
	t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

	// earlier tests in this package leave work running for a few seconds; start from idle
	config.WAIT_GROUP_TIMEOUT_SECONDS = 30
	if WaitRunningTimeout() {
		t.Fatal("previous tests left work running")
	}

	config.WAIT_GROUP_TIMEOUT_SECONDS = 1
	release := make(chan struct{})
	go func() {
		AddRunning()
		defer DoneRunning()
		<-release
	}()

	// give the goroutine time to register before waiting on it
	time.Sleep(100 * time.Millisecond)
	assert.True(t, WaitRunningTimeout(), "should have given up on the work still running")

	close(release)
	assert.False(t, WaitRunningTimeout(), "should observe the drain after the previous wait timed out")

	// a new round must be observable after the counter went back to zero
	AddRunning()
	DoneRunning()
	assert.False(t, WaitRunningTimeout())
}

func TestGetWaitGroupConcurrent(t *testing.T) {
	logging.Initialize()

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
