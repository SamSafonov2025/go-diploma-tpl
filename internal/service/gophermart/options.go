package gophermart

import (
	"context"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/storage/postgres"
)

// Option is a function that configures the Service
type Option func(*Service)

// WithPostgresStorage sets PostgreSQL storage
func WithPostgresStorage(ctx context.Context) Option {
	return func(s *Service) {
		s.storage = postgres.NewStorage(ctx)
	}
}

// WithDefaultStorage sets the default storage (PostgreSQL)
func WithDefaultStorage(ctx context.Context) Option {
	return WithPostgresStorage(ctx)
}
