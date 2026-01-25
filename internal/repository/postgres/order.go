package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OrderRepository реализует domain.OrderRepository
type OrderRepository struct {
	db DBTX
}

// NewOrderRepository создает новый OrderRepository
func NewOrderRepository(db DBTX) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateOrder создает новый заказ
func (r *OrderRepository) CreateOrder(ctx context.Context, userID int64, number string) (*domain.Order, error) {
	order := &domain.Order{
		UserID: userID,
		Number: number,
		Status: domain.OrderStatusNew,
	}

	err := r.db.QueryRow(ctx,
		`INSERT INTO orders (user_id, number, status) 
		 VALUES ($1, $2, $3) 
		 RETURNING id, uploaded_at`,
		userID, number, order.Status,
	).Scan(&order.ID, &order.UploadedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Проверяем, кому принадлежит заказ
			existingOrder, getErr := r.GetOrderByNumber(ctx, number)
			if getErr != nil {
				return nil, fmt.Errorf("repository: failed to check existing order: %w", getErr)
			}
			if existingOrder.UserID != userID {
				return nil, domain.ErrOrderOwnedByAnother
			}
			return existingOrder, domain.ErrOrderExists
		}
		return nil, fmt.Errorf("repository: failed to create order %q: %w", number, err)
	}

	return order, nil
}

// GetOrderByNumber получает заказ по номеру
func (r *OrderRepository) GetOrderByNumber(ctx context.Context, number string) (*domain.Order, error) {
	order := &domain.Order{}

	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at 
		 FROM orders 
		 WHERE number = $1`,
		number,
	).Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("repository: failed to get order by number %q: %w", number, err)
	}

	return order, nil
}

// GetOrdersByUserID получает все заказы пользователя
func (r *OrderRepository) GetOrdersByUserID(ctx context.Context, userID int64) ([]*domain.Order, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at 
		 FROM orders 
		 WHERE user_id = $1 
		 ORDER BY uploaded_at DESC`,
		userID,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: failed to get orders for user %d: %w", userID, err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, fmt.Errorf("repository: failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: error iterating orders: %w", err)
	}

	return orders, nil
}

// UpdateOrderStatus обновляет статус заказа и начисление
func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, number string, status domain.OrderStatus, accrual *float64) error {
	result, err := r.db.Exec(ctx,
		`UPDATE orders 
		 SET status = $1, accrual = $2 
		 WHERE number = $3`,
		status, accrual, number,
	)

	if err != nil {
		return fmt.Errorf("repository: failed to update order %q status: %w", number, err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}

	return nil
}

// GetPendingOrders получает все заказы со статусом NEW или PROCESSING
func (r *OrderRepository) GetPendingOrders(ctx context.Context) ([]*domain.Order, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at 
		 FROM orders 
		 WHERE status IN ($1, $2) 
		 ORDER BY uploaded_at ASC`,
		domain.OrderStatusNew, domain.OrderStatusProcessing,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: failed to get pending orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, fmt.Errorf("repository: failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: error iterating pending orders: %w", err)
	}

	return orders, nil
}
