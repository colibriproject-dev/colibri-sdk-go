package observer

// Observer is the default contract for components that need to perform a graceful shutdown.
type Observer interface {
	// Close performs the cleanup or shutdown logic for the component.
	Close()
}

// Stopper blocks new work from entering the application. It is called in the first phase of
// the graceful shutdown and must return without waiting: draining belongs to phase 2.
//
// It is optional and additive: an observer may implement both interfaces, and is attached
// once. An observer that only implements Observer is closed in the closing phase as always.
type Stopper interface {
	// Stop blocks new work from entering the component, without waiting for the work already
	// in flight.
	Stop()
}
