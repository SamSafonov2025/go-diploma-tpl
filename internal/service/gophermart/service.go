package gophermart

import (
	"context"
	"time"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/accrual"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/storage"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

const updateInterval = 10 * time.Second

type Service struct {
	AccrualAPI accrual.Client
	Repository storage.Storage
}

func NewService(ctx context.Context, options ...ServiceOption) *Service {
	service := &Service{
		AccrualAPI: accrual.NewClient(),
	}

	// Apply options
	for _, opt := range options {
		opt(service)
	}

	// Use default storage if not provided
	if service.Repository == nil {
		WithStorage(ctx)(service)
	}

	// Start background order processing
	service.startBackgroundProcessing(ctx)

	return service
}

func (s *Service) startBackgroundProcessing(ctx context.Context) {
	ticker := time.NewTicker(updateInterval)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				zap.L().Error("Recovered from panic in background processing", zap.Any("panic", r))
			}
		}()

		for {
			select {
			case <-ticker.C:
				s.processOrders(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Service) processOrders(ctx context.Context) {
	// Fetch unprocessed orders
	pendingOrders, err := s.Repository.FetchPendingOrders(ctx)
	if err != nil {
		zap.L().Warn("Failed to fetch pending orders", zap.Error(err))
		return
	}

	if len(pendingOrders) == 0 {
		return
	}

	// Process each order
	for _, order := range pendingOrders {
		// Get order info from accrual service
		updatedOrder, err := s.AccrualAPI.FetchOrderInfo(order)
		if err != nil {
			zap.L().Warn("Failed to fetch order info from accrual",
				zap.String("orderID", order.ID),
				zap.Error(err))
			continue
		}

		// Update order in transaction
		if err = s.Repository.ExecuteTransaction(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
			// Update order status
			if err := s.Repository.ModifyOrder(ctx, updatedOrder, tx); err != nil {
				return err
			}

			// Update balance if accrual exists
			if updatedOrder.Accrual == nil {
				return nil
			}

			return s.Repository.UpdateBalance(ctx, updatedOrder.UID, updatedOrder.Accrual, tx)
		}); err != nil {
			zap.L().Warn("Failed to update order",
				zap.String("orderID", order.ID),
				zap.Error(err))
		}
	}
}
