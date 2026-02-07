package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/repository/postgres"
	"github.com/avc/loyalty-system-diploma/internal/utils/luhn"
)

// OrderRepository определяет методы для работы с заказами.
type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, number string) (*domain.Order, error)
	GetOrderByNumber(ctx context.Context, number string) (*domain.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64) ([]*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, number string, status domain.OrderStatus, accrual *float64) error
	GetPendingOrders(ctx context.Context) ([]*domain.Order, error)
}

// OrderService предоставляет операции с заказами.
type OrderService struct {
	orderRepo OrderRepository
}

// NewOrderService создает новый OrderService
func NewOrderService(orderRepo OrderRepository) *OrderService {
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
