package shutdown

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// SignalListener listens for OS signals
type SignalListener struct {
	signals chan os.Signal
}

// NewSignalListener creates a new signal listener
func NewSignalListener() *SignalListener {
	listener := &SignalListener{
		signals: make(chan os.Signal, 1),
	}
	signal.Notify(listener.signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	return listener
}

// Wait blocks until a shutdown signal is received
func (sl *SignalListener) Wait() {
	sig := <-sl.signals
	zap.L().Info("Shutdown signal received", zap.String("signal", sig.String()))
}

// Stop stops listening for signals
func (sl *SignalListener) Stop() {
	signal.Stop(sl.signals)
	close(sl.signals)
}
