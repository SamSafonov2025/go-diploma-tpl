package gophermart

import (
	"context"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/storage/postgres"
)

type ServiceOption func(*Service)

func WithStorage(ctx context.Context) ServiceOption {
	return func(s *Service) {
		s.Repository = postgres.NewStorage(ctx)
	}
}

func WithPostgresStorage(ctx context.Context) ServiceOption {
	return WithStorage(ctx)
}
