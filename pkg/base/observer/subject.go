package observer

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

type subject interface {
	attach(observer Observer)
	notify()
}

var services subject

// Initialize starts the observation of system signals to trigger graceful shutdown.
func Initialize() {
	ch := make(chan os.Signal, 1)
	services = &service{
		observers: make([]Observer, 0, 0),
	}
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL, os.Interrupt)

	go func() {
		sig := <-ch
		logging.Warn(context.Background()).Msgf("notify shutdown: %+v", sig)
		services.notify()
	}()
}

// Attach adds an observer to the notification list for graceful shutdown.
func Attach(o Observer) {
	services.attach(o)
}

type service struct {
	observers []Observer
}

func (s *service) attach(observer Observer) {
	s.observers = append(s.observers, observer)
}

func (s *service) notify() {
	for _, observer := range s.observers {
		observer.Close()
	}
}
