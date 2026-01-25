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
	ctx := context.Background()

	tests := []struct {
		name         string
		userID       int64
		setupMock    func(*domainmocks.TransactionRepositoryMock) *domain.Balance
		wantErr      bool
	}{
		{
			name:   "Success",
			userID: 1,
			setupMock: func(m *domainmocks.TransactionRepositoryMock) *domain.Balance {
				balance := &domain.Balance{Current: 500.0, Withdrawn: 200.0}
				m.EXPECT().GetBalance(mock.Anything, int64(1)).Return(balance, nil).Once()
				return balance
			},
		},
		{
			name:   "Database error",
			userID: 1,
			setupMock: func(m *domainmocks.TransactionRepositoryMock) *domain.Balance {
				m.EXPECT().GetBalance(mock.Anything, int64(1)).Return(nil, errors.New("db error")).Once()
				return nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
			svc := NewBalanceService(mockTxRepo)

			expectedBalance := tt.setupMock(mockTxRepo)

			result, err := svc.GetBalance(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, expectedBalance, result)
			}
		})
	}
}

func TestBalanceService_Withdraw(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      int64
		orderNumber string
		amount      float64
		setupMock   func(*domainmocks.TransactionRepositoryMock)
		wantErr     error
	}{
		{
			name:        "Success",
			userID:      1,
			orderNumber: "79927398713", // Valid Luhn
			amount:      100.0,
			setupMock: func(m *domainmocks.TransactionRepositoryMock) {
				m.EXPECT().WithdrawWithLock(mock.Anything, int64(1), "79927398713", 100.0).Return(nil).Once()
			},
		},
		{
			name:        "Invalid order number - fails Luhn",
			userID:      1,
			orderNumber: "12345", // Invalid Luhn
			amount:      100.0,
			setupMock:   func(m *domainmocks.TransactionRepositoryMock) {},
			wantErr:     domain.ErrInvalidOrderNumber,
		},
		{
			name:        "Invalid amount - zero",
			userID:      1,
			orderNumber: "79927398713",
			amount:      0.0,
			setupMock:   func(m *domainmocks.TransactionRepositoryMock) {},
			wantErr:     nil, // Generic error
		},
		{
			name:        "Invalid amount - negative",
			userID:      1,
			orderNumber: "79927398713",
			amount:      -100.0,
			setupMock:   func(m *domainmocks.TransactionRepositoryMock) {},
			wantErr:     nil, // Generic error
		},
		{
			name:        "Insufficient funds",
			userID:      1,
			orderNumber: "79927398713",
			amount:      1000.0,
			setupMock: func(m *domainmocks.TransactionRepositoryMock) {
				m.EXPECT().WithdrawWithLock(mock.Anything, int64(1), "79927398713", 1000.0).Return(domain.ErrInsufficientFunds).Once()
			},
			wantErr: domain.ErrInsufficientFunds,
		},
		{
			name:        "Database error",
			userID:      1,
			orderNumber: "79927398713",
			amount:      100.0,
			setupMock: func(m *domainmocks.TransactionRepositoryMock) {
				m.EXPECT().WithdrawWithLock(mock.Anything, int64(1), "79927398713", 100.0).Return(errors.New("db error")).Once()
			},
			wantErr: nil, // Generic error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
			svc := NewBalanceService(mockTxRepo)

			tt.setupMock(mockTxRepo)

			err := svc.Withdraw(ctx, tt.userID, tt.orderNumber, tt.amount)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else if tt.name == "Invalid amount - zero" || tt.name == "Invalid amount - negative" || tt.name == "Database error" {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBalanceService_GetWithdrawals(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		userID          int64
		setupMock       func(*domainmocks.TransactionRepositoryMock) []*domain.Transaction
		wantWithdrawals int
		wantErr         bool
	}{
		{
			name:   "Success with withdrawals",
			userID: 1,
			setupMock: func(m *domainmocks.TransactionRepositoryMock) []*domain.Transaction {
				withdrawals := []*domain.Transaction{
					{ID: 1, UserID: 1, OrderNumber: "111", Amount: 100.0, Type: domain.TransactionTypeWithdrawal, ProcessedAt: time.Now()},
					{ID: 2, UserID: 1, OrderNumber: "222", Amount: 50.0, Type: domain.TransactionTypeWithdrawal, ProcessedAt: time.Now()},
				}
				m.EXPECT().GetWithdrawals(mock.Anything, int64(1)).Return(withdrawals, nil).Once()
				return withdrawals
			},
			wantWithdrawals: 2,
		},
		{
			name:   "No withdrawals",
			userID: 999,
			setupMock: func(m *domainmocks.TransactionRepositoryMock) []*domain.Transaction {
				m.EXPECT().GetWithdrawals(mock.Anything, int64(999)).Return([]*domain.Transaction{}, nil).Once()
				return nil
			},
			wantWithdrawals: 0,
		},
		{
			name:   "Database error",
			userID: 1,
			setupMock: func(m *domainmocks.TransactionRepositoryMock) []*domain.Transaction {
				m.EXPECT().GetWithdrawals(mock.Anything, int64(1)).Return(nil, errors.New("db error")).Once()
				return nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
			svc := NewBalanceService(mockTxRepo)

			expectedWithdrawals := tt.setupMock(mockTxRepo)

			result, err := svc.GetWithdrawals(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.wantWithdrawals)
				if expectedWithdrawals != nil {
					assert.Equal(t, expectedWithdrawals, result)
				}
			}
		})
	}
}
