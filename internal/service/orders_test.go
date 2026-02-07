package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	domainmocks "github.com/avc/loyalty-system-diploma/internal/domain/mocks"
	"github.com/avc/loyalty-system-diploma/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOrderService_SubmitOrder(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      int64
		orderNumber string
		setupMock   func(*domainmocks.OrderRepositoryMock)
		wantErr     error
	}{
		{
			name:        "Success",
			userID:      1,
			orderNumber: "79927398713", // Valid Luhn
			setupMock: func(m *domainmocks.OrderRepositoryMock) {
				order := &domain.Order{ID: 1, UserID: 1, Number: "79927398713", Status: domain.OrderStatusNew}
				m.EXPECT().CreateOrder(mock.Anything, int64(1), "79927398713").Return(order, nil).Once()
			},
			wantErr: nil,
		},
		{
			name:        "Invalid order number - fails Luhn",
			userID:      1,
			orderNumber: "12345", // Invalid Luhn
			setupMock:   func(m *domainmocks.OrderRepositoryMock) {},
			wantErr:     ErrInvalidOrderNumber,
		},
		{
			name:        "Order already exists - same user",
			userID:      1,
			orderNumber: "79927398713",
			setupMock: func(m *domainmocks.OrderRepositoryMock) {
				m.EXPECT().CreateOrder(mock.Anything, int64(1), "79927398713").Return(nil, postgres.ErrOrderExists).Once()
			},
			wantErr: ErrOrderExists,
		},
		{
			name:        "Order owned by another user",
			userID:      1,
			orderNumber: "79927398713",
			setupMock: func(m *domainmocks.OrderRepositoryMock) {
				m.EXPECT().CreateOrder(mock.Anything, int64(1), "79927398713").Return(nil, postgres.ErrOrderOwnedByAnother).Once()
			},
			wantErr: ErrOrderOwnedByAnother,
		},
		{
			name:        "Database error",
			userID:      1,
			orderNumber: "79927398713",
			setupMock: func(m *domainmocks.OrderRepositoryMock) {
				m.EXPECT().CreateOrder(mock.Anything, int64(1), "79927398713").Return(nil, errors.New("db error")).Once()
			},
			wantErr: nil, // Generic error, just check error exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
			svc := NewOrderService(mockOrderRepo)

			tt.setupMock(mockOrderRepo)

			err := svc.SubmitOrder(ctx, tt.userID, tt.orderNumber)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else if tt.name == "Database error" {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOrderService_GetOrders(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		userID     int64
		setupMock  func(*domainmocks.OrderRepositoryMock) []*domain.Order
		wantOrders int
		wantErr    bool
	}{
		{
			name:   "Success with orders",
			userID: 1,
			setupMock: func(m *domainmocks.OrderRepositoryMock) []*domain.Order {
				orders := []*domain.Order{
					{ID: 1, UserID: 1, Number: "111", Status: domain.OrderStatusProcessed, UploadedAt: time.Now()},
					{ID: 2, UserID: 1, Number: "222", Status: domain.OrderStatusNew, UploadedAt: time.Now()},
				}
				m.EXPECT().GetOrdersByUserID(mock.Anything, int64(1)).Return(orders, nil).Once()
				return orders
			},
			wantOrders: 2,
		},
		{
			name:   "No orders",
			userID: 999,
			setupMock: func(m *domainmocks.OrderRepositoryMock) []*domain.Order {
				m.EXPECT().GetOrdersByUserID(mock.Anything, int64(999)).Return([]*domain.Order{}, nil).Once()
				return nil
			},
			wantOrders: 0,
		},
		{
			name:   "Database error",
			userID: 1,
			setupMock: func(m *domainmocks.OrderRepositoryMock) []*domain.Order {
				m.EXPECT().GetOrdersByUserID(mock.Anything, int64(1)).Return(nil, errors.New("db error")).Once()
				return nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
			svc := NewOrderService(mockOrderRepo)

			expectedOrders := tt.setupMock(mockOrderRepo)

			result, err := svc.GetOrders(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.wantOrders)
				if expectedOrders != nil {
					assert.Equal(t, expectedOrders, result)
				}
			}
		})
	}
}
