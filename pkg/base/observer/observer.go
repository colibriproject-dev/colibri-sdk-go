package observer

// Observer is the default contract for components that need to perform a graceful shutdown.
type Observer interface {
	// Close performs the cleanup or shutdown logic for the component.
	Close()
}
