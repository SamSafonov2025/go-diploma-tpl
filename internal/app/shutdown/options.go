package shutdown

import "time"

// Option is a configuration option for the coordinator
type Option func(*Coordinator)

// WithTimeout sets a custom timeout for shutdown
func WithTimeout(timeout time.Duration) Option {
	return func(c *Coordinator) {
		c.timeout = timeout
	}
}

// WithCustomListener sets a custom signal listener
func WithCustomListener(listener *SignalListener) Option {
	return func(c *Coordinator) {
		c.listener = listener
	}
}

// NewCoordinatorWithOptions creates a coordinator with custom options
func NewCoordinatorWithOptions(opts ...Option) *Coordinator {
	c := &Coordinator{
		listener: NewSignalListener(),
		executor: NewExecutor(),
		timeout:  defaultTimeout,
		done:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}
