package observer

import (
	"context"
	"sync"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

var once sync.Once
var singleInstance *sync.WaitGroup

// Running work is tracked by a counter with an idle channel rather than by waiting on the
// WaitGroup directly. A wait that can time out has to be abandoned when it does, and a
// goroutine left parked in WaitGroup.Wait races with the next Add that raises the counter
// from zero.
var (
	runningMu    sync.Mutex
	runningCount int
	runningIdle  chan struct{}
)

// GetWaitGroup returns the process wide WaitGroup used to track work that a graceful
// shutdown must drain. The check lives inside the Once because callers reach it from
// concurrent goroutines: guarding it with a plain nil check races with the assignment.
//
// Deprecated: use AddRunning and DoneRunning. WaitRunningTimeout reads its own counter, so
// work registered directly on the returned WaitGroup is invisible to the shutdown.
func GetWaitGroup() *sync.WaitGroup {
	once.Do(func() {
		logging.Debug(context.Background()).Msg("Creating single WaitGroup instance now.")
		singleInstance = &sync.WaitGroup{}
	})

	return singleInstance
}

// AddRunning registers one unit of work that a graceful shutdown must wait for.
func AddRunning() {
	runningMu.Lock()
	defer runningMu.Unlock()

	runningCount++
	GetWaitGroup().Add(1)
}

// DoneRunning marks one unit of registered work as finished.
func DoneRunning() {
	runningMu.Lock()
	defer runningMu.Unlock()

	runningCount--
	GetWaitGroup().Done()

	if runningCount == 0 && runningIdle != nil {
		close(runningIdle)
		runningIdle = nil
	}
}

// idleSignal returns a channel that is closed once no registered work remains.
func idleSignal() <-chan struct{} {
	runningMu.Lock()
	defer runningMu.Unlock()

	if runningCount == 0 {
		idle := make(chan struct{})
		close(idle)
		return idle
	}

	if runningIdle == nil {
		runningIdle = make(chan struct{})
	}

	return runningIdle
}

// WaitRunningTimeout waits for the registered work to finish and reports whether it gave up
// before that happened. The graceful shutdown calls it once, in its drain phase, so the whole
// shutdown costs one timeout: a module observer that waits on its own would spend another one
// on the work this wait already covers.
func WaitRunningTimeout() bool {
	select {
	case <-idleSignal():
		return false
	case <-time.After(time.Duration(config.WAIT_GROUP_TIMEOUT_SECONDS) * time.Second):
		return true
	}
}
