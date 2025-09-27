package main

import (
	"context"
	"log"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/logger"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/server"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/shutdown"
)

func main() {
	// Initialize logger first
	logger.Initialize()

	// Create context for application
	ctx := context.Background()

	// Start HTTP server
	server.Start(ctx)

	// Wait for graceful shutdown
	<-shutdown.Manager().Done()

	log.Println("Application stopped successfully")
}
