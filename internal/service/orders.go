package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/repository/postgres"
	"github.com/avc/loyalty-system-diploma/internal/utils/luhn"
)

// OrderService реализует domain.OrderService
type OrderService struct {
	orderRepo domain.OrderRepository
}

// NewOrderService создает новый OrderService
func NewOrderService(orderRepo domain.OrderRepository) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
	}
}

// SubmitOrder принимает номер заказа для обработки
func (s *OrderService) SubmitOrder(ctx context.Context, userID int64, orderNumber string) error {
	// Валидация номера заказа по алгоритму Луна
	if !luhn.Validate(orderNumber) {
		return ErrInvalidOrderNumber
	}

	// Создание заказа
	_, err := s.orderRepo.CreateOrder(ctx, userID, orderNumber)
	if err != nil {
		// Не оборачиваем sentinel errors
		if errors.Is(err, postgres.ErrOrderExists) {
			return ErrOrderExists
		}
		if errors.Is(err, postgres.ErrOrderOwnedByAnother) {
			return ErrOrderOwnedByAnother
		}
		return fmt.Errorf("order service: failed to submit order %q: %w", orderNumber, err)
	}

	return nil
}

// GetOrders получает все заказы пользователя
func (s *OrderService) GetOrders(ctx context.Context, userID int64) ([]*domain.Order, error) {
	orders, err := s.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("order service: failed to get orders for user %d: %w", userID, err)
	}

	return orders, nil
}
