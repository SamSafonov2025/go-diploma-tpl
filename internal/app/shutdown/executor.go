package shutdown

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Closer is a function that performs cleanup
type Closer func(context.Context) error

// Executor manages and executes shutdown functions
type Executor struct {
	mu      sync.RWMutex
	closers []Closer
}

// NewExecutor creates a new shutdown executor
func NewExecutor() *Executor {
	return &Executor{
		closers: make([]Closer, 0),
	}
}

// Register adds a new closer function
func (e *Executor) Register(closer Closer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closers = append(e.closers, closer)
}

// Execute runs all registered closers in LIFO order
func (e *Executor) Execute(ctx context.Context) []error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var errors []error

	// Execute in reverse order (LIFO)
	for i := len(e.closers) - 1; i >= 0; i-- {
		if err := e.closers[i](ctx); err != nil {
			zap.L().Error("Closer execution failed",
				zap.Int("index", i),
				zap.Error(err))
			errors = append(errors, err)
		}
	}

	return errors
}

// Count returns the number of registered closers
func (e *Executor) Count() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.closers)
}
