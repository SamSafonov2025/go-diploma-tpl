package shutdown

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutor(t *testing.T) {
	executor := NewExecutor()

	// Track execution order
	var order []int

	// Register closers
	executor.Register(func(ctx context.Context) error {
		order = append(order, 1)
		return nil
	})

	executor.Register(func(ctx context.Context) error {
		order = append(order, 2)
		return nil
	})

	executor.Register(func(ctx context.Context) error {
		order = append(order, 3)
		return nil
	})

	// Execute
	ctx := context.Background()
	errors := executor.Execute(ctx)

	// Check no errors
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}

	// Check LIFO order
	if len(order) != 3 || order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Errorf("Expected LIFO order [3,2,1], got %v", order)
	}
}

func TestExecutorWithErrors(t *testing.T) {
	executor := NewExecutor()

	// Register closers with errors
	executor.Register(func(ctx context.Context) error {
		return errors.New("error 1")
	})

	executor.Register(func(ctx context.Context) error {
		return nil
	})

	executor.Register(func(ctx context.Context) error {
		return errors.New("error 2")
	})

	// Execute
	ctx := context.Background()
	errs := executor.Execute(ctx)

	// Check errors were collected
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errs))
	}
}

func TestCoordinatorTimeout(t *testing.T) {
	coordinator := NewCoordinator(100 * time.Millisecond)

	// Register slow closer
	coordinator.Register(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			return nil
		}
	})

	// This test would need a way to trigger shutdown manually
	// In real usage, it waits for OS signals
}
