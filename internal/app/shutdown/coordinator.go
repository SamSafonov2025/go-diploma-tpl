package shutdown

import (
	"context"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultTimeout = 5 * time.Second
	forceExitDelay = 10 * time.Second
)

// Coordinator coordinates the shutdown process
type Coordinator struct {
	listener *SignalListener
	executor *Executor
	timeout  time.Duration
	done     chan struct{}
	once     sync.Once
}

// NewCoordinator creates a new shutdown coordinator
func NewCoordinator(timeout time.Duration) *Coordinator {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &Coordinator{
		listener: NewSignalListener(),
		executor: NewExecutor(),
		timeout:  timeout,
		done:     make(chan struct{}),
	}
}

// Register adds a closer function
func (c *Coordinator) Register(closer Closer) {
	c.executor.Register(closer)
}

// Start begins listening for shutdown signals
func (c *Coordinator) Start() {
	go c.waitAndShutdown()
}

// Done returns a channel that's closed when shutdown is complete
func (c *Coordinator) Done() <-chan struct{} {
	return c.done
}

// waitAndShutdown waits for signal and coordinates shutdown
func (c *Coordinator) waitAndShutdown() {
	// Wait for shutdown signal
	c.listener.Wait()

	// Initiate graceful shutdown
	c.once.Do(func() {
		c.performShutdown()
	})
}

// performShutdown executes the shutdown sequence
func (c *Coordinator) performShutdown() {
	zap.L().Info("Starting graceful shutdown",
		zap.Duration("timeout", c.timeout),
		zap.Int("closers", c.executor.Count()))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// Channel to signal completion
	completed := make(chan struct{})

	// Execute shutdown in goroutine
	go func() {
		errors := c.executor.Execute(ctx)
		if len(errors) > 0 {
			zap.L().Warn("Shutdown completed with errors",
				zap.Int("error_count", len(errors)))
		} else {
			zap.L().Info("Graceful shutdown completed successfully")
		}
		close(completed)
	}()

	// Wait for completion or timeout
	select {
	case <-completed:
		close(c.done)
		c.listener.Stop()

	case <-time.After(forceExitDelay):
		zap.L().Error("Force exit due to shutdown timeout")
		os.Exit(1)
	}
}
