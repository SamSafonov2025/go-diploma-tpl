package models

import "errors"

var (
	// Balance errors
	ErrNotEnoughBalance = errors.New("insufficient balance for withdrawal")

	// Order errors
	ErrEmptyOrderList      = errors.New("no orders found")
	ErrOrderExists         = errors.New("order already exists")
	ErrOrderOwnedByAnother = errors.New("order belongs to another user")
	ErrInvalidOrderNum     = errors.New("invalid order number format")

	// User errors
	ErrUserExists         = errors.New("user already exists")
	ErrUnauthorized       = errors.New("user is not authorized")
	ErrInvalidCredentials = errors.New("invalid login credentials")

	// Token errors
	ErrInvalidToken       = errors.New("invalid authentication token")
	ErrInvalidTokenFormat = errors.New("invalid bearer token format")

	// Withdrawal errors
	ErrEmptyWithdrawalList = errors.New("no withdrawals found")
)
