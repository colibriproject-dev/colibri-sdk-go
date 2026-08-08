package observer

import (
	"cmp"
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"slices"
	"sync"
	"syscall"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

// Priority defines when an observer is closed during the graceful shutdown. Observers are
// closed from the lowest value to the highest, so the shutdown order comes from what each
// module is, not from the order the application happened to call Initialize in.
//
// It only classifies the closing phases: stopping and draining have no ordering between
// modules, because every module stops before anything drains.
type Priority int

const (
	// PriorityDefault closes the resources whose use has a side effect, such as the message
	// broker connection, the databases and the storage client. This is the level Attach uses.
	//
	// The rest client is not here on purpose: its clients are built ad hoc by the application,
	// so there is no instance to attach, and an idle http.Client needs no closing.
	PriorityDefault Priority = 0
	// PriorityLast closes what the resources above still need while they are closing, such as
	// the monitoring exporter, which flushes only once nothing else can emit telemetry.
	PriorityLast Priority = 1
)

type subject interface {
	attach(observer Observer, priority Priority)
	notify()
}

var (
	servicesMu sync.Mutex
	services   subject
	signalCh   chan os.Signal
)

// Initialize starts the observation of system signals to trigger graceful shutdown.
func Initialize() {
	reset()

	ch := make(chan os.Signal, 1)
	s := &service{observers: make([]registeredObserver, 0)}

	servicesMu.Lock()
	services = s
	signalCh = ch
	servicesMu.Unlock()

	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL, os.Interrupt)

	go func() {
		sig, ok := <-ch
		if !ok {
			// a later Initialize replaced this handler, there is nothing to shut down
			return
		}

		logging.Warn(context.Background()).Msgf("notify shutdown: %+v", sig)
		s.notify()
	}()
}

// reset stops the signal handling started by a previous Initialize so its goroutine can
// exit. Without it every Initialize leaves one goroutine parked forever on a channel that
// nobody writes to, which the test suites reinitializing the package pay on every run.
func reset() {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	if signalCh != nil {
		signal.Stop(signalCh)
		close(signalCh)
		signalCh = nil
	}
}

// Attach adds an observer to the notification list for graceful shutdown with the default
// priority. Use AttachWithPriority when the component has to be closed after everything else.
func Attach(o Observer) {
	AttachWithPriority(o, PriorityDefault)
}

// AttachWithPriority adds an observer to the notification list for graceful shutdown,
// closed at the given priority. Observers sharing a priority are closed in the order they
// were attached in.
func AttachWithPriority(o Observer, p Priority) {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	services.attach(o, p)
}

// Notify runs the whole graceful shutdown, the same way a termination signal does: every
// attached observer is stopped, then the running work is drained, then everything is closed.
func Notify() {
	servicesMu.Lock()
	s := services
	servicesMu.Unlock()

	if s != nil {
		s.notify()
	}
}

type registeredObserver struct {
	observer Observer
	priority Priority
}

type service struct {
	mu        sync.Mutex
	observers []registeredObserver
}

func (s *service) attach(observer Observer, priority Priority) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.observers = append(s.observers, registeredObserver{observer: observer, priority: priority})
}

// notify runs the graceful shutdown as a pipeline of phases. Every module goes through the
// same phase before any of them moves to the next one, which is what keeps a module from
// closing a resource another one still needs to drain through.
//
// It runs on a copy of the observers because one of them may attach another while it is
// being notified.
func (s *service) notify() {
	registered := s.snapshot()

	// phase 1: block new work from entering. Nothing waits here, so the components still
	// draining keep every resource they need available.
	for _, r := range registered {
		if stopper, ok := r.observer.(Stopper); ok {
			safely("stop", r.observer, stopper.Stop)
		}
	}

	// phase 2: drain the work in flight, once for the whole application. A module waiting on
	// its own would spend another timeout on work this wait already covers.
	if WaitRunningTimeout() {
		logging.Warn(context.Background()).Msg("waiting timed out, forcing the shutdown of the running work")
	}

	// phase 3 and 4: close the resources, by priority. The sort is stable so the attach order
	// is kept inside a priority.
	slices.SortStableFunc(registered, func(a, b registeredObserver) int {
		return cmp.Compare(a.priority, b.priority)
	})

	for _, r := range registered {
		safely("close", r.observer, r.observer.Close)
	}
}

// safely runs one phase of one observer, keeping a module that panics from taking the whole
// shutdown down with it. Before the pipeline that cost the panicking module alone; now a panic
// in the stopping phase would mean nothing is ever drained or closed.
func safely(phase string, o Observer, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logging.
				Error(context.Background()).
				Msgf("panic on %s of observer %T: %v\n%s", phase, o, r, debug.Stack())
		}
	}()

	fn()
}

func (s *service) snapshot() []registeredObserver {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.observers)
}
