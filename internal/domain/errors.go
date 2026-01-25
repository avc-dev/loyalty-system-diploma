package domain

import (
	"errors"
	"fmt"
	"time"
)

// Ошибки пользователей
var (
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
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
	ErrDuplicateAccrual  = errors.New("accrual already exists for this order")
)

// Ошибки accrual клиента
var (
	ErrAccrualNotRegistered = errors.New("order not registered in accrual system")
)

// RateLimitError представляет ошибку превышения лимита запросов
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retry after %s", e.RetryAfter)
}

// NewRateLimitError создает новую ошибку rate limit
func NewRateLimitError(retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{RetryAfter: retryAfter}
}
