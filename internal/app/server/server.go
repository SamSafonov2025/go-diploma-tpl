package server

import (
	"context"
	"net/http"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/config"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/router"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/shutdown"
	"go.uber.org/zap"
)

type HTTPServer struct {
	srv http.Server
}

func NewHTTPServer(ctx context.Context) *HTTPServer {
	cfg := config.Load()
	return &HTTPServer{
		srv: http.Server{
			Addr:    cfg.Server.Addr,
			Handler: router.Setup(ctx),
		},
	}
}

func Start(ctx context.Context) {
	httpServer := NewHTTPServer(ctx)
	go httpServer.Serve()
	httpServer.registerShutdown()
}

func (s *HTTPServer) Serve() {
	zap.L().Info("Starting HTTP server", zap.String("addr", s.srv.Addr))
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zap.L().Fatal("Failed to start server", zap.Error(err))
	}
}

func (s *HTTPServer) registerShutdown() {
	// Register shutdown handler with the coordinator
	shutdown.Manager().Register(func(ctx context.Context) error {
		zap.L().Info("Shutting down HTTP server")
		return s.srv.Shutdown(ctx)
	})
}
