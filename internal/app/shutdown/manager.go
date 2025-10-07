package shutdown

import (
	"sync"
	"time"
)

var (
	defaultCoordinator *Coordinator
	initOnce           sync.Once
)

// Initialize sets up the global shutdown coordinator
func Initialize(timeout time.Duration) {
	initOnce.Do(func() {
		defaultCoordinator = NewCoordinator(timeout)
		defaultCoordinator.Start()
	})
}

// Manager returns the global shutdown coordinator
// Initializes with default timeout if not already initialized
func Manager() *Coordinator {
	if defaultCoordinator == nil {
		Initialize(defaultTimeout)
	}
	return defaultCoordinator
}
