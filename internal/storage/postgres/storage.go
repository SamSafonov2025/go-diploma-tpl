package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/config"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/shutdown"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/storage"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

const operationTimeout = time.Second

var (
	//go:embed sql/migrations/*.sql
	migrationsFS  embed.FS
	migrationsDir = "sql/migrations"

	//go:embed sql/queries
	queriesFS  embed.FS
	queriesDir = "sql/queries"
)

type PostgresStorage struct {
	conn    *sqlx.DB
	queries querySet
}

type querySet struct {
	// Balance queries
	insertOrUpdateBalance string
	updateWithdrawnAmount string
	selectBalance         string

	// Order queries
	insertOrder          string
	updateOrder          string
	selectOrderByID      string
	selectOrdersByUserID string
	selectPendingOrders  string

	// User queries
	insertUser        string
	selectUserByLogin string

	// Withdrawal queries
	insertWithdrawal          string
	selectWithdrawalsByUserID string
}

var _ storage.Storage = (*PostgresStorage)(nil)

func NewStorage(ctx context.Context) storage.Storage {
	cfg := config.Load()

	// Connect to database
	conn, err := sqlx.Connect("pgx", cfg.Database.DSN)
	if err != nil {
		zap.L().Fatal("Failed to connect to database", zap.Error(err))
	}

	store := &PostgresStorage{conn: conn}

	// Run migrations
	if err = store.runMigrations(ctx); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.Error(err))
	}

	// Load queries
	if err = store.loadQueries(ctx); err != nil {
		zap.L().Fatal("Failed to load queries", zap.Error(err))
	}

	// Register for shutdown
	store.registerShutdown()

	return store
}

// User operations

func (s *PostgresStorage) CreateUser(ctx context.Context, user models.User) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	result, err := s.conn.NamedExecContext(ctx, s.queries.insertUser, user)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return models.ErrUserExists
	}

	return nil
}

func (s *PostgresStorage) FindUserByLogin(ctx context.Context, user models.User) (models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	var foundUser models.User
	err := s.conn.GetContext(ctx, &foundUser, s.queries.selectUserByLogin, user.Login)
	return foundUser, err
}

// Order operations

func (s *PostgresStorage) CreateOrder(ctx context.Context, newOrder models.Order, tx *sqlx.Tx) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	// Check if order exists
	var existingOrder models.Order
	err := tx.GetContext(ctx, &existingOrder, s.queries.selectOrderByID, newOrder.ID)
	if err == nil {
		if existingOrder.UID == newOrder.UID {
			return models.ErrOrderExists
		}
		return models.ErrOrderOwnedByAnother
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// Insert new order
	ctx, cancel = context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	_, err = tx.NamedExecContext(ctx, s.queries.insertOrder, &newOrder)
	return err

}

func (s *PostgresStorage) ModifyOrder(ctx context.Context, order models.Order, tx *sqlx.Tx) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	_, err := tx.NamedExecContext(ctx, s.queries.updateOrder, &order)
	return err
}

func (s *PostgresStorage) FetchOrdersByUserID(ctx context.Context, userID string) ([]models.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	var orders []models.Order
	if err := s.conn.SelectContext(ctx, &orders, s.queries.selectOrdersByUserID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrEmptyOrderList
		}
		return nil, err
	}

	if len(orders) == 0 {
		return nil, models.ErrEmptyOrderList
	}

	return orders, nil
}

func (s *PostgresStorage) FetchPendingOrders(ctx context.Context) ([]models.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	var orders []models.Order
	err := s.conn.SelectContext(ctx, &orders, s.queries.selectPendingOrders,
		models.StatusNew, models.StatusProcessing)
	return orders, err
}

// Balance operations

func (s *PostgresStorage) FetchBalanceByUserID(ctx context.Context, userID string) (models.Balance, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	var balance models.Balance
	if err := s.conn.GetContext(ctx, &balance, s.queries.selectBalance, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// по ТЗ возвращаем 0/0
			return models.Balance{UID: userID, Current: 0, Withdrawn: 0}, nil
		}
		return models.Balance{}, err
	}
	return balance, nil
}

func (s *PostgresStorage) FetchCurrentBalance(ctx context.Context, userID string, tx *sqlx.Tx) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	var balance models.Balance
	if err := tx.GetContext(ctx, &balance, s.queries.selectBalance, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return balance.Current, nil
}

func (s *PostgresStorage) UpdateBalance(ctx context.Context, userID string, amount *float64, tx *sqlx.Tx) error {
	var value float64
	if amount != nil {
		value = *amount
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	_, err := tx.ExecContext(ctx, s.queries.insertOrUpdateBalance, value, userID)
	return err
}

func (s *PostgresStorage) IncreaseWithdrawnAmount(ctx context.Context, userID string, amount float64, tx *sqlx.Tx) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	_, err := tx.ExecContext(ctx, s.queries.updateWithdrawnAmount, amount, userID)
	return err
}

// Withdrawal operations

func (s *PostgresStorage) CreateWithdrawal(ctx context.Context, withdrawal models.Withdrawal, tx *sqlx.Tx) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	_, err := tx.NamedExecContext(ctx, s.queries.insertWithdrawal, &withdrawal)
	return err
}

func (s *PostgresStorage) FetchWithdrawalsByUserID(ctx context.Context, userID string) ([]models.Withdrawal, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	var withdrawals []models.Withdrawal
	if err := s.conn.SelectContext(ctx, &withdrawals, s.queries.selectWithdrawalsByUserID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrEmptyWithdrawalList
		}
		return nil, err
	}

	if len(withdrawals) == 0 {
		return nil, models.ErrEmptyWithdrawalList
	}

	return withdrawals, nil
}

// Transaction support

func (s *PostgresStorage) ExecuteTransaction(ctx context.Context, fn func(ctx context.Context, tx *sqlx.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, 2*operationTimeout)
	defer cancel()

	tx, err := s.conn.BeginTxx(ctx, nil)
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			zap.L().Error("Transaction panic", zap.Any("panic", p))
			panic(p)
		}

		if err != nil {
			_ = tx.Rollback()
			zap.L().Debug("Transaction rolled back", zap.Error(err))
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				zap.L().Error("Failed to commit transaction", zap.Error(commitErr))
				err = commitErr
			}
		}
	}()

	err = fn(ctx, tx)
	return err
}

// Helper methods

func (s *PostgresStorage) loadQueries(_ context.Context) error {
	files, err := queriesFS.ReadDir(queriesDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		content, err := queriesFS.ReadFile(fmt.Sprintf("%s/%s", queriesDir, file.Name()))
		if err != nil {
			return err
		}

		query := string(content)

		switch file.Name() {
		case "insert_or_update_balances_by_uid.sql":
			s.queries.insertOrUpdateBalance = query
		case "update_balance_withdrawn_by_uid.sql":
			s.queries.updateWithdrawnAmount = query
		case "select_balance_by_uid.sql":
			s.queries.selectBalance = query
		case "insert_order.sql":
			s.queries.insertOrder = query
		case "update_orders.sql":
			s.queries.updateOrder = query
		case "select_order_by_id.sql":
			s.queries.selectOrderByID = query
		case "select_orders_by_uid.sql":
			s.queries.selectOrdersByUserID = query
		case "select_orders_by_statuses.sql":
			s.queries.selectPendingOrders = query
		case "insert_user.sql":
			s.queries.insertUser = query
		case "select_user_by_login.sql":
			s.queries.selectUserByLogin = query
		case "insert_withdrawals.sql":
			s.queries.insertWithdrawal = query
		case "select_withdrawals_by_uid.sql":
			s.queries.selectWithdrawalsByUserID = query
		}
	}

	return nil
}

func (s *PostgresStorage) runMigrations(_ context.Context) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Up(s.conn.DB, migrationsDir)
}

func (s *PostgresStorage) registerShutdown() {
	shutdown.Manager().Register(func(_ context.Context) error {
		return s.conn.Close()
	})
}
