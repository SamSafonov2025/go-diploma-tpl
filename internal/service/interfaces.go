package service

import (
	"context"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/jmoiron/sqlx"
)

// AccrualClient defines the interface for accrual service
type AccrualClient interface {
	FetchOrderInfo(order models.Order) (models.Order, error)
}

// Storage defines the interface for data storage
type Storage interface {
	// Transaction support
	ExecuteTransaction(ctx context.Context, fn func(ctx context.Context, tx *sqlx.Tx) error) error

	// User operations
	CreateUser(ctx context.Context, user models.User) error
	FindUserByLogin(ctx context.Context, user models.User) (models.User, error)

	// Order operations
	CreateOrder(ctx context.Context, order models.Order, tx *sqlx.Tx) error
	ModifyOrder(ctx context.Context, order models.Order, tx *sqlx.Tx) error
	FetchOrdersByUserID(ctx context.Context, userID string) ([]models.Order, error)
	FetchPendingOrders(ctx context.Context) ([]models.Order, error)

	// Balance operations
	FetchBalanceByUserID(ctx context.Context, userID string) (models.Balance, error)
	FetchCurrentBalance(ctx context.Context, userID string, tx *sqlx.Tx) (float64, error)
	UpdateBalance(ctx context.Context, userID string, amount *float64, tx *sqlx.Tx) error
	IncreaseWithdrawnAmount(ctx context.Context, userID string, amount float64, tx *sqlx.Tx) error

	// Withdrawal operations
	CreateWithdrawal(ctx context.Context, withdrawal models.Withdrawal, tx *sqlx.Tx) error
	FetchWithdrawalsByUserID(ctx context.Context, userID string) ([]models.Withdrawal, error)
}
