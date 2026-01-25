package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	domainmocks "github.com/avc/loyalty-system-diploma/internal/domain/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBalanceService_GetBalance(t *testing.T) {
	mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
	svc := NewBalanceService(mockTxRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int64(1)
		balance := &domain.Balance{Current: 500.0, Withdrawn: 200.0}

		mockTxRepo.EXPECT().GetBalance(mock.Anything, userID).Return(balance, nil).Once()

		result, err := svc.GetBalance(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, balance, result)
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)

		mockTxRepo.EXPECT().GetBalance(mock.Anything, userID).Return(nil, errors.New("db error")).Once()

		result, err := svc.GetBalance(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestBalanceService_Withdraw(t *testing.T) {
	mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
	svc := NewBalanceService(mockTxRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713" // Valid Luhn
		amount := 100.0

		mockTxRepo.EXPECT().WithdrawWithLock(mock.Anything, userID, orderNumber, amount).Return(nil).Once()

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		require.NoError(t, err)
	})

	t.Run("Invalid order number", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345" // Invalid Luhn
		amount := 100.0

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		assert.ErrorIs(t, err, domain.ErrInvalidOrderNumber)
	})

	t.Run("Invalid amount - zero", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713"
		amount := 0.0

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		assert.Error(t, err)
	})

	t.Run("Invalid amount - negative", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713"
		amount := -100.0

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		assert.Error(t, err)
	})

	t.Run("Insufficient funds", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713"
		amount := 1000.0

		mockTxRepo.EXPECT().WithdrawWithLock(mock.Anything, userID, orderNumber, amount).Return(domain.ErrInsufficientFunds).Once()

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		assert.ErrorIs(t, err, domain.ErrInsufficientFunds)
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713"
		amount := 100.0

		mockTxRepo.EXPECT().WithdrawWithLock(mock.Anything, userID, orderNumber, amount).Return(errors.New("db error")).Once()

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		assert.Error(t, err)
	})
}

func TestBalanceService_GetWithdrawals(t *testing.T) {
	mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
	svc := NewBalanceService(mockTxRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int64(1)
		withdrawals := []*domain.Transaction{
			{ID: 1, UserID: userID, OrderNumber: "111", Amount: 100.0, Type: domain.TransactionTypeWithdrawal, ProcessedAt: time.Now()},
			{ID: 2, UserID: userID, OrderNumber: "222", Amount: 50.0, Type: domain.TransactionTypeWithdrawal, ProcessedAt: time.Now()},
		}

		mockTxRepo.EXPECT().GetWithdrawals(mock.Anything, userID).Return(withdrawals, nil).Once()

		result, err := svc.GetWithdrawals(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, withdrawals, result)
	})

	t.Run("No withdrawals", func(t *testing.T) {
		userID := int64(999)

		mockTxRepo.EXPECT().GetWithdrawals(mock.Anything, userID).Return([]*domain.Transaction{}, nil).Once()

		result, err := svc.GetWithdrawals(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)

		mockTxRepo.EXPECT().GetWithdrawals(mock.Anything, userID).Return(nil, errors.New("db error")).Once()

		result, err := svc.GetWithdrawals(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
