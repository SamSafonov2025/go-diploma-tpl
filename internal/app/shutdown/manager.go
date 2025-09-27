package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const shutdownTimeout = 5 * time.Second

type ShutdownManager struct {
	mu        sync.RWMutex
	closers   []func(context.Context) error
	done      chan struct{}
	completed chan struct{}
}

var (
	manager     *ShutdownManager
	managerOnce sync.Once
	closeOnce   sync.Once
)

func Manager() *ShutdownManager {
	managerOnce.Do(func() {
		manager = &ShutdownManager{
			done:      make(chan struct{}),
			completed: make(chan struct{}),
		}
		go manager.waitForSignal()
	})
	return manager
}

func (sm *ShutdownManager) Register(closer func(context.Context) error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.closers = append(sm.closers, closer)
}

func (sm *ShutdownManager) Done() <-chan struct{} {
	return sm.done
}

func (sm *ShutdownManager) performShutdown() {
	closeOnce.Do(func() {
		sm.mu.RLock()
		defer sm.mu.RUnlock()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		// Execute closers in reverse order (LIFO)
		for i := len(sm.closers) - 1; i >= 0; i-- {
			if err := sm.closers[i](ctx); err != nil {
				zap.L().Error("Shutdown error", zap.Error(err))
			}
		}

		close(sm.completed)
	})
}

func (sm *ShutdownManager) waitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-sigChan
	zap.L().Info("Shutdown signal received, starting graceful shutdown...")

	go sm.performShutdown()

	select {
	case <-sm.completed:
		zap.L().Info("Graceful shutdown completed")
		close(sm.done)
	case <-time.After(2 * shutdownTimeout):
		zap.L().Error("Graceful shutdown timeout exceeded, forcing exit")
		os.Exit(1)
	}
}
