package domain

import "errors"

// Ошибки пользователей
var (
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Ошибки заказов
var (
	ErrOrderExists         = errors.New("order already exists")
	ErrOrderOwnedByAnother = errors.New("order owned by another user")
	ErrInvalidOrderNumber  = errors.New("invalid order number")
	ErrOrderNotFound       = errors.New("order not found")
)

// Ошибки транзакций и баланса
var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)
