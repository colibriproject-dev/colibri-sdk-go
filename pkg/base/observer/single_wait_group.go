package observer

import (
	"sync"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
)

// defaultDrainTimeout bounds the drain when the configured timeout is missing or not
// positive, so a misconfigured application still shuts down gracefully instead of giving up
// on its running work immediately.
const defaultDrainTimeout = 90 * time.Second

// The running work is counted in two buckets because the lifetimes differ by orders of
// magnitude. A unit of application work lasts as long as the request or the message carrying
// it, while a module task, such as the goroutine listening on a queue, lasts as long as the
// process. Counting them together would make a wait for the application work block forever on
// a listener that only ends during the shutdown.
//
// Neither bucket is a sync.WaitGroup. A wait that can time out has to be abandoned when it
// does, and a WaitGroup gives no way to do that; worse, Wait may not run concurrently with an
// Add that raises the counter from zero, which is exactly what a drain running while the last
// units of work finish and restart does. The counters below are guarded and signal through
// channels, so the drain is a select and never touches the work being registered.
var (
	runningMu     sync.Mutex
	appRunning    int
	moduleRunning int
	idleApp       chan struct{}
	idleAll       chan struct{}

	// legacyWaitGroup backs GetWaitGroup. It is kept because it is public API: an application
	// registering work on it must still be drained, exactly as it is today.
	legacyWaitGroup = &sync.WaitGroup{}
)

// AddRunning registers one unit of application work that a graceful shutdown must wait for.
func AddRunning() {
	runningMu.Lock()
	defer runningMu.Unlock()

	appRunning++
}

// DoneRunning marks one unit of application work as finished.
func DoneRunning() {
	runningMu.Lock()
	defer runningMu.Unlock()

	appRunning--
	signalIdle()
}

// AddModuleTask registers one long lived task that a graceful shutdown must drain, such as
// the goroutine listening on a queue.
//
// It exists for the SDK modules. The work an application runs belongs on AddRunning: a task
// registered here is drained by the shutdown but is deliberately invisible to a wait for the
// application work, which would otherwise never return while a listener that only ends during
// the shutdown is up.
func AddModuleTask() {
	runningMu.Lock()
	defer runningMu.Unlock()

	moduleRunning++
}

// DoneModuleTask marks one registered module task as finished.
func DoneModuleTask() {
	runningMu.Lock()
	defer runningMu.Unlock()

	moduleRunning--
	signalIdle()
}

// GetWaitGroup returns the process wide WaitGroup that counts application work.
//
// Deprecated: use AddRunning and DoneRunning. A sync.WaitGroup cannot be waited on with a
// timeout, so the drain has to wait for it on a goroutine it abandons when it gives up, and
// its Wait may not run concurrently with an Add that raises the counter from zero. The
// counters behind AddRunning have neither problem. Work registered here is still drained.
func GetWaitGroup() *sync.WaitGroup {
	runningMu.Lock()
	defer runningMu.Unlock()

	return legacyWaitGroup
}

// signalIdle releases the waiters whose bucket has just emptied. It must be called with
// runningMu held.
//
// The counters are compared with <= rather than == so an unbalanced Done cannot drive one
// past zero and leave every later wait hanging on a channel nobody will ever close.
func signalIdle() {
	if appRunning <= 0 && idleApp != nil {
		close(idleApp)
		idleApp = nil
	}

	if appRunning <= 0 && moduleRunning <= 0 && idleAll != nil {
		close(idleAll)
		idleAll = nil
	}
}

// appIdleSignal returns a channel closed once no application work remains.
func appIdleSignal() <-chan struct{} {
	runningMu.Lock()
	defer runningMu.Unlock()

	if appRunning <= 0 {
		return closedSignal()
	}

	if idleApp == nil {
		idleApp = make(chan struct{})
	}

	return idleApp
}

// allIdleSignal returns a channel closed once neither application work nor module tasks
// remain.
func allIdleSignal() <-chan struct{} {
	runningMu.Lock()
	defer runningMu.Unlock()

	if appRunning <= 0 && moduleRunning <= 0 {
		return closedSignal()
	}

	if idleAll == nil {
		idleAll = make(chan struct{})
	}

	return idleAll
}

func closedSignal() <-chan struct{} {
	signal := make(chan struct{})
	close(signal)

	return signal
}

// WaitRunningTimeout waits for every registered unit of work, application and module alike,
// and reports whether it gave up before that happened. The graceful shutdown calls it once,
// in its drain phase, so the whole shutdown costs one timeout: a module observer that waits
// on its own would spend another one on the work this wait already covers.
func WaitRunningTimeout() bool {
	// one deadline for every wait below, so they share a single budget instead of taking one
	// each. The channel keeps its value once fired, so each select still observes it.
	deadline := time.After(DrainBudget(1, 1))

	select {
	case <-allIdleSignal():
	case <-deadline:
		return true
	}

	return !waitLegacyWaitGroup(deadline)
}

// waitLegacyWaitGroup waits for the work an application registered through GetWaitGroup and
// reports whether it emptied before the deadline.
//
// The wait happens on a goroutine this one abandons when it gives up, which is what the
// deprecation on GetWaitGroup is about: the goroutine stays parked until that work ends, and
// while it is parked an Add raising the counter from zero is a misuse of sync.WaitGroup. The
// counters have no such constraint, so an application that moves to AddRunning leaves this
// path unused.
func waitLegacyWaitGroup(deadline <-chan time.Time) bool {
	wg := GetWaitGroup()

	done := make(chan struct{})
	go func() {
		defer close(done)

		wg.Wait()
	}()

	select {
	case <-done:
		return true
	case <-deadline:
		return false
	}
}

// DrainBudget returns the num/den fraction of the time the shutdown drains for, so a
// component that has to finish before the drain gives up can bound itself with a budget that
// follows WAIT_GROUP_TIMEOUT_SECONDS instead of a constant that only fits its default.
//
// Every budget is derived here so they stay ordered against each other: a component given a
// fraction below 1 is guaranteed to reach its own bound before the drain reaches its.
func DrainBudget(num, den int) time.Duration {
	if num <= 0 || den <= 0 {
		return 0
	}

	total := time.Duration(config.WAIT_GROUP_TIMEOUT_SECONDS) * time.Second
	if total <= 0 {
		total = defaultDrainTimeout
	}

	return total * time.Duration(num) / time.Duration(den)
}
