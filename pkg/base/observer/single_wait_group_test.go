package observer

import (
	"sync"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/stretchr/testify/assert"
)

func TestGetWaitGroup(t *testing.T) {
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
	wg := GetWaitGroup()
	wg.Add(1)
	defer wg.Done()
	time.Sleep(time.Duration(delaySeconds) * time.Second)
}
