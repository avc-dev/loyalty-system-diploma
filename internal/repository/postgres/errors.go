package postgres

import "errors"

// Ошибки пользователей
var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

// Ошибки заказов
var (
	ErrOrderExists         = errors.New("order already exists")
	ErrOrderOwnedByAnother = errors.New("order owned by another user")
	ErrOrderNotFound       = errors.New("order not found")
)

// Ошибки транзакций и баланса
var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrDuplicateAccrual  = errors.New("accrual already exists for this order")
)
