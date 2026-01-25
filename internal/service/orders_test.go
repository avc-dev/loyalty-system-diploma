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

func TestOrderService_SubmitOrder(t *testing.T) {
	mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
	svc := NewOrderService(mockOrderRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713" // Valid Luhn
		order := &domain.Order{ID: 1, UserID: userID, Number: orderNumber, Status: domain.OrderStatusNew}

		mockOrderRepo.EXPECT().CreateOrder(mock.Anything, userID, orderNumber).Return(order, nil).Once()

		err := svc.SubmitOrder(ctx, userID, orderNumber)
		require.NoError(t, err)
	})

	t.Run("Invalid order number", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345" // Invalid Luhn

		err := svc.SubmitOrder(ctx, userID, orderNumber)
		assert.ErrorIs(t, err, domain.ErrInvalidOrderNumber)
	})

	t.Run("Order already exists", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713"

		mockOrderRepo.EXPECT().CreateOrder(mock.Anything, userID, orderNumber).Return(nil, domain.ErrOrderExists).Once()

		err := svc.SubmitOrder(ctx, userID, orderNumber)
		assert.ErrorIs(t, err, domain.ErrOrderExists)
	})

	t.Run("Order owned by another user", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713"

		mockOrderRepo.EXPECT().CreateOrder(mock.Anything, userID, orderNumber).Return(nil, domain.ErrOrderOwnedByAnother).Once()

		err := svc.SubmitOrder(ctx, userID, orderNumber)
		assert.ErrorIs(t, err, domain.ErrOrderOwnedByAnother)
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "79927398713"

		mockOrderRepo.EXPECT().CreateOrder(mock.Anything, userID, orderNumber).Return(nil, errors.New("db error")).Once()

		err := svc.SubmitOrder(ctx, userID, orderNumber)
		assert.Error(t, err)
	})
}

func TestOrderService_GetOrders(t *testing.T) {
	mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
	svc := NewOrderService(mockOrderRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int64(1)
		orders := []*domain.Order{
			{ID: 1, UserID: userID, Number: "111", Status: domain.OrderStatusProcessed, UploadedAt: time.Now()},
			{ID: 2, UserID: userID, Number: "222", Status: domain.OrderStatusNew, UploadedAt: time.Now()},
		}

		mockOrderRepo.EXPECT().GetOrdersByUserID(mock.Anything, userID).Return(orders, nil).Once()

		result, err := svc.GetOrders(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, orders, result)
	})

	t.Run("No orders", func(t *testing.T) {
		userID := int64(999)

		mockOrderRepo.EXPECT().GetOrdersByUserID(mock.Anything, userID).Return([]*domain.Order{}, nil).Once()

		result, err := svc.GetOrders(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)

		mockOrderRepo.EXPECT().GetOrdersByUserID(mock.Anything, userID).Return(nil, errors.New("db error")).Once()

		result, err := svc.GetOrders(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
