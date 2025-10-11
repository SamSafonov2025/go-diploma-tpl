package gophermart

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Mock implementations for testing

type mockAccrualClient struct {
	fetchFunc func(models.Order) (models.Order, error)
}

func (m *mockAccrualClient) FetchOrderInfo(order models.Order) (models.Order, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(order)
	}
	return order, nil
}

type mockStorage struct {
	pendingOrders []models.Order
	modifyError   error
	balanceError  error
}

func (m *mockStorage) ExecuteTransaction(ctx context.Context, fn func(context.Context, *sqlx.Tx) error) error {
	// Simulate transaction execution
	return fn(ctx, nil)
}

func (m *mockStorage) FetchPendingOrders(ctx context.Context) ([]models.Order, error) {
	return m.pendingOrders, nil
}

func (m *mockStorage) ModifyOrder(ctx context.Context, order models.Order, tx *sqlx.Tx) error {
	return m.modifyError
}

func (m *mockStorage) UpdateBalance(ctx context.Context, userID string, amount *float64, tx *sqlx.Tx) error {
	return m.balanceError
}

// Implement other required methods with empty implementations
func (m *mockStorage) CreateUser(context.Context, models.User) error { return nil }
func (m *mockStorage) FindUserByLogin(context.Context, models.User) (models.User, error) {
	return models.User{}, nil
}
func (m *mockStorage) CreateOrder(context.Context, models.Order, *sqlx.Tx) error { return nil }
func (m *mockStorage) FetchOrdersByUserID(context.Context, string) ([]models.Order, error) {
	return nil, nil
}
func (m *mockStorage) FetchBalanceByUserID(context.Context, string) (models.Balance, error) {
	return models.Balance{}, nil
}
func (m *mockStorage) FetchCurrentBalance(context.Context, string, *sqlx.Tx) (float64, error) {
	return 0, nil
}
func (m *mockStorage) IncreaseWithdrawnAmount(context.Context, string, float64, *sqlx.Tx) error {
	return nil
}
func (m *mockStorage) CreateWithdrawal(context.Context, models.Withdrawal, *sqlx.Tx) error {
	return nil
}
func (m *mockStorage) FetchWithdrawalsByUserID(context.Context, string) ([]models.Withdrawal, error) {
	return nil, nil
}

// Tests

func TestNewService(t *testing.T) {
	ctx := context.Background()

	mockAccrual := &mockAccrualClient{}
	mockStore := &mockStorage{}

	// Create service using options pattern
	svc := NewService(ctx,
		WithStorage(mockStore),
		WithAccrualClient(mockAccrual), // *Client implements service.AccrualClient
		WithLogger(zap.NewNop()),
		WithUpdateInterval(1*time.Second),
	)

	if svc.storage != mockStore {
		t.Error("Storage not set correctly")
	}

	if svc.accrualClient != mockAccrual {
		t.Error("AccrualClient not set correctly")
	}

	if svc.updateInterval != 1*time.Second {
		t.Error("UpdateInterval not set correctly")
	}
}

func TestServiceWithNilAccrualClient(t *testing.T) {
	ctx := context.Background()

	svc := NewService(ctx,
		WithStorage(&mockStorage{}),
		WithLogger(zap.NewNop()),
	)

	if svc.storage == nil {
		t.Error("Storage should not be nil")
	}
}

func TestProcessOrder(t *testing.T) {
	ctx := context.Background()

	accrualValue := 100.0
	processedOrder := models.Order{
		ID:            "123",
		UID:           "user1",
		AccrualStatus: models.StatusProcessed,
		Accrual:       &accrualValue,
	}

	mockAccrual := &mockAccrualClient{
		fetchFunc: func(order models.Order) (models.Order, error) {
			return processedOrder, nil
		},
	}

	mockStore := &mockStorage{}

	svc := NewService(ctx,
		WithStorage(mockStore),
		WithAccrualClient(mockAccrual), // *Client implements service.AccrualClient
		WithLogger(zap.NewNop()),
	)

	// Process order
	svc.processOrder(ctx, models.Order{ID: "123", UID: "user1"})

	// Test should complete without errors
}

func TestProcessOrderWithError(t *testing.T) {
	ctx := context.Background()

	mockAccrual := &mockAccrualClient{
		fetchFunc: func(order models.Order) (models.Order, error) {
			return order, errors.New("accrual error")
		},
	}

	mockStore := &mockStorage{}

	svc := NewService(ctx,
		WithStorage(mockStore),
		WithAccrualClient(mockAccrual), // *Client implements service.AccrualClient
		WithLogger(zap.NewNop()),
	)

	// Should not panic on error
	svc.processOrder(ctx, models.Order{ID: "123"})
}
