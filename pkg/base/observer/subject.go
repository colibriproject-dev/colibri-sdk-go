package observer

import (
	"context"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

const (
	attachBeforeInitialize string = "observer.Initialize must be called before attaching %T"
	drainTimedOut          string = "waiting timed out, forcing the shutdown of the running work"
)

type subject interface {
	attach(observer Observer)
	notify()
}

var (
	servicesMu sync.Mutex
	services   subject
)

// Initialize starts the observation of system signals to trigger graceful shutdown.
func Initialize() {
	ch := make(chan os.Signal, 1)
	s := &service{observers: make([]Observer, 0)}

	servicesMu.Lock()
	services = s
	servicesMu.Unlock()

	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL, os.Interrupt)

	go func() {
		sig := <-ch
		logging.Warn(context.Background()).Msgf("notify shutdown: %+v", sig)
		s.notify()
	}()
}

// Attach adds an observer to the notification list for graceful shutdown.
func Attach(o Observer) {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	if services == nil {
		logging.Fatal(context.Background()).Msgf(attachBeforeInitialize, o)
		return
	}

	services.attach(o)
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

type service struct {
	mu        sync.Mutex
	observers []Observer
}

func (s *service) attach(observer Observer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.observers = append(s.observers, observer)
}

// notify runs the graceful shutdown as a pipeline of phases. Every module goes through the
// same phase before any of them moves to the next one, which is what keeps a module from
// closing a resource another one still needs to drain through.
//
// It runs on a copy of the observers because one of them may attach another while it is
// being notified.
func (s *service) notify() {
	observers := s.snapshot()

	// phase 1: block new work from entering. Nothing waits here, so the components still
	// draining keep every resource they need available.
	for _, o := range observers {
		if stopper, ok := o.(Stopper); ok {
			stopper.Stop()
		}
	}

	// phase 2: drain the work in flight, once for the whole application. A module waiting on
	// its own would spend another timeout on work this wait already covers.
	if WaitRunningTimeout() {
		logging.Warn(context.Background()).Msg(drainTimedOut)
	}

	// phase 3: release the resources, now that nothing is left using them
	for _, o := range observers {
		o.Close()
	}
}

func (s *service) snapshot() []Observer {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.observers)
}
