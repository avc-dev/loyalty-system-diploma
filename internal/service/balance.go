package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/repository/postgres"
	"github.com/avc/loyalty-system-diploma/internal/utils/luhn"
)

// TransactionRepository определяет методы для работы с транзакциями.
type TransactionRepository interface {
	CreateTransaction(ctx context.Context, userID int64, orderNumber string, amount float64, txType domain.TransactionType) error
	GetBalance(ctx context.Context, userID int64) (*domain.Balance, error)
	GetWithdrawals(ctx context.Context, userID int64) ([]*domain.Transaction, error)
	WithdrawWithLock(ctx context.Context, userID int64, orderNumber string, amount float64) error
}

// BalanceService предоставляет операции с балансом.
type BalanceService struct {
	transactionRepo TransactionRepository
}

// NewBalanceService создает новый BalanceService
func NewBalanceService(transactionRepo TransactionRepository) *BalanceService {
	return &BalanceService{
		transactionRepo: transactionRepo,
	}
}

// GetBalance получает баланс пользователя
func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (*domain.Balance, error) {
	balance, err := s.transactionRepo.GetBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("balance service: failed to get balance for user %d: %w", userID, err)
	}

	return balance, nil
}

// Withdraw списывает средства со счета пользователя
func (s *BalanceService) Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error {
	// Валидация номера заказа по алгоритму Луна
	if !luhn.Validate(orderNumber) {
		return ErrInvalidOrderNumber
	}

	// Валидация суммы
	if amount <= 0 {
		return fmt.Errorf("balance service: invalid withdrawal amount: %f", amount)
	}

	// Списание средств с блокировкой
	err := s.transactionRepo.WithdrawWithLock(ctx, userID, orderNumber, amount)
	if err != nil {
		// Не оборачиваем sentinel errors
		if errors.Is(err, postgres.ErrInsufficientFunds) {
			return ErrInsufficientFunds
		}
		return fmt.Errorf("balance service: failed to withdraw %f for user %d: %w", amount, userID, err)
	}

	return nil
}

// GetWithdrawals получает историю списаний пользователя
func (s *BalanceService) GetWithdrawals(ctx context.Context, userID int64) ([]*domain.Transaction, error) {
	withdrawals, err := s.transactionRepo.GetWithdrawals(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("balance service: failed to get withdrawals for user %d: %w", userID, err)
	}

	return withdrawals, nil
}
