package service

import (
	"errors"
	"fmt"
	"time"
)

// Ошибки аутентификации и ввода
var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Ошибки заказов и баланса
var (
	ErrInvalidOrderNumber  = errors.New("invalid order number")
	ErrOrderExists         = errors.New("order already exists")
	ErrOrderOwnedByAnother = errors.New("order owned by another user")
	ErrInsufficientFunds   = errors.New("insufficient funds")
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
