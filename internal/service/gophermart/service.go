package gophermart

import (
	"context"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"time"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/service"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

const defaultUpdateInterval = 10 * time.Second

// Service manages the gophermart business logic
type Service struct {
	accrualClient  service.AccrualClient
	storage        service.Storage
	updateInterval time.Duration
	logger         *zap.Logger
}

// NewService creates service with options pattern
func NewService(ctx context.Context, opts ...Option) *Service {
	svc := &Service{
		updateInterval: defaultUpdateInterval,
		logger:         zap.L(),
	}

	// Apply all options
	for _, opt := range opts {
		opt(svc)
	}

	// Set defaults if not provided by options
	if svc.storage == nil {
		WithDefaultStorage(ctx)(svc)
	}

	// Start background processing if accrual client is provided
	if svc.accrualClient != nil {
		svc.startBackgroundProcessing(ctx)
	}

	return svc
}

// Option functions for configuring the service

// WithAccrualClient sets the accrual client
func WithAccrualClient(client service.AccrualClient) Option {
	return func(s *Service) {
		s.accrualClient = client
	}
}

// WithStorage sets the storage
func WithStorage(storage service.Storage) Option {
	return func(s *Service) {
		s.storage = storage
	}
}

// WithUpdateInterval sets the update interval
func WithUpdateInterval(interval time.Duration) Option {
	return func(s *Service) {
		s.updateInterval = interval
	}
}

// WithLogger sets the logger
func WithLogger(logger *zap.Logger) Option {
	return func(s *Service) {
		s.logger = logger
	}
}

// Repository returns the storage interface (for handlers)
func (s *Service) Repository() service.Storage {
	return s.storage
}

// Rest of the methods remain the same...
func (s *Service) startBackgroundProcessing(ctx context.Context) {
	ticker := time.NewTicker(s.updateInterval)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("Recovered from panic in background processing",
					zap.Any("panic", r))
			}
		}()

		for {
			select {
			case <-ticker.C:
				s.processOrders(ctx)
			case <-ctx.Done():
				ticker.Stop()
				s.logger.Info("Background processing stopped")
				return
			}
		}
	}()
}

func (s *Service) processOrders(ctx context.Context) {
	// Same implementation as before
	pendingOrders, err := s.storage.FetchPendingOrders(ctx)
	if err != nil {
		s.logger.Warn("Failed to fetch pending orders", zap.Error(err))
		return
	}

	if len(pendingOrders) == 0 {
		return
	}

	s.logger.Debug("Processing pending orders", zap.Int("count", len(pendingOrders)))

	for _, order := range pendingOrders {
		s.processOrder(ctx, order)
	}
}

func (s *Service) processOrder(ctx context.Context, order models.Order) {
	updatedOrder, err := s.accrualClient.FetchOrderInfo(order)
	if err != nil {
		s.logger.Warn("Failed to fetch order info from accrual",
			zap.String("orderID", order.ID),
			zap.Error(err))
		return
	}

	err = s.storage.ExecuteTransaction(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		if err := s.storage.ModifyOrder(ctx, updatedOrder, tx); err != nil {
			return err
		}

		if updatedOrder.Accrual == nil {
			return nil
		}

		return s.storage.UpdateBalance(ctx, updatedOrder.UID, updatedOrder.Accrual, tx)
	})

	if err != nil {
		s.logger.Warn("Failed to update order",
			zap.String("orderID", order.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("Order processed successfully",
			zap.String("orderID", order.ID),
			zap.String("status", string(updatedOrder.AccrualStatus)))
	}
}
