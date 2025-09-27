package logger

import (
	"context"
	"sync"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/shutdown"
	"go.uber.org/zap"
)

var setupOnce sync.Once

func Initialize() {
	setupOnce.Do(func() {
		log, err := zap.NewProduction()
		if err != nil {
			panic("cannot initialize logger: " + err.Error())
		}

		// Register logger for graceful shutdown
		registerShutdown(log)

		// Replace global logger
		zap.ReplaceGlobals(log)

		zap.L().Info("Logger initialized successfully")
	})
}

func registerShutdown(log *zap.Logger) {
	shutdown.Manager().Register(func(ctx context.Context) error {
		return log.Sync()
	})
}
