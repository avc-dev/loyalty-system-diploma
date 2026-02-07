package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderRepository_CreateOrder(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewOrderRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int64(1)
		number := "12345678903"
		now := time.Now()

		rows := pgxmock.NewRows([]string{"id", "uploaded_at"}).
			AddRow(int64(1), now)

		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(userID, number, domain.OrderStatusNew).
			WillReturnRows(rows)

		order, err := repo.CreateOrder(ctx, userID, number)
		require.NoError(t, err)
		assert.Equal(t, int64(1), order.ID)
		assert.Equal(t, userID, order.UserID)
		assert.Equal(t, number, order.Number)
		assert.Equal(t, domain.OrderStatusNew, order.Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Order exists - same user", func(t *testing.T) {
		userID := int64(1)
		number := "12345678903"

		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(userID, number, domain.OrderStatusNew).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		// Мокируем GetOrderByNumber
		existingOrder := &domain.Order{
			ID:         1,
			UserID:     userID,
			Number:     number,
			Status:     domain.OrderStatusProcessing,
			UploadedAt: time.Now(),
		}

		rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
			AddRow(existingOrder.ID, existingOrder.UserID, existingOrder.Number, existingOrder.Status, existingOrder.Accrual, existingOrder.UploadedAt)

		mock.ExpectQuery(`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE number`).
			WithArgs(number).
			WillReturnRows(rows)

		order, err := repo.CreateOrder(ctx, userID, number)
		assert.ErrorIs(t, err, ErrOrderExists)
		assert.Equal(t, existingOrder.ID, order.ID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Order exists - different user", func(t *testing.T) {
		userID := int64(1)
		otherUserID := int64(2)
		number := "12345678903"

		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(userID, number, domain.OrderStatusNew).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		// Мокируем GetOrderByNumber - заказ принадлежит другому пользователю
		existingOrder := &domain.Order{
			ID:         1,
			UserID:     otherUserID,
			Number:     number,
			Status:     domain.OrderStatusProcessing,
			UploadedAt: time.Now(),
		}

		rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
			AddRow(existingOrder.ID, existingOrder.UserID, existingOrder.Number, existingOrder.Status, existingOrder.Accrual, existingOrder.UploadedAt)

		mock.ExpectQuery(`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE number`).
			WithArgs(number).
			WillReturnRows(rows)

		order, err := repo.CreateOrder(ctx, userID, number)
		assert.ErrorIs(t, err, ErrOrderOwnedByAnother)
		assert.Nil(t, order)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrderRepository_GetOrderByNumber(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewOrderRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		number := "12345678903"
		accrual := 100.50
		expectedOrder := &domain.Order{
			ID:         1,
			UserID:     1,
			Number:     number,
			Status:     domain.OrderStatusProcessed,
			Accrual:    &accrual,
			UploadedAt: time.Now(),
		}

		rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
			AddRow(expectedOrder.ID, expectedOrder.UserID, expectedOrder.Number, expectedOrder.Status, expectedOrder.Accrual, expectedOrder.UploadedAt)

		mock.ExpectQuery(`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE number`).
			WithArgs(number).
			WillReturnRows(rows)

		order, err := repo.GetOrderByNumber(ctx, number)
		require.NoError(t, err)
		assert.Equal(t, expectedOrder.Number, order.Number)
		assert.Equal(t, expectedOrder.Status, order.Status)
		assert.Equal(t, *expectedOrder.Accrual, *order.Accrual)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Order not found", func(t *testing.T) {
		number := "99999999999"

		mock.ExpectQuery(`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE number`).
			WithArgs(number).
			WillReturnError(pgx.ErrNoRows)

		order, err := repo.GetOrderByNumber(ctx, number)
		assert.ErrorIs(t, err, ErrOrderNotFound)
		assert.Nil(t, order)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrderRepository_GetOrdersByUserID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewOrderRepository(mock)
	ctx := context.Background()

	t.Run("Success - multiple orders", func(t *testing.T) {
		userID := int64(1)
		accrual := 100.0

		rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
			AddRow(int64(1), userID, "111", domain.OrderStatusProcessed, &accrual, time.Now()).
			AddRow(int64(2), userID, "222", domain.OrderStatusProcessing, nil, time.Now()).
			AddRow(int64(3), userID, "333", domain.OrderStatusNew, nil, time.Now())

		mock.ExpectQuery(`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE user_id`).
			WithArgs(userID).
			WillReturnRows(rows)

		orders, err := repo.GetOrdersByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, orders, 3)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success - no orders", func(t *testing.T) {
		userID := int64(999)

		rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"})

		mock.ExpectQuery(`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE user_id`).
			WithArgs(userID).
			WillReturnRows(rows)

		orders, err := repo.GetOrdersByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, orders)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectQuery(`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE user_id`).
			WithArgs(userID).
			WillReturnError(errors.New("database error"))

		orders, err := repo.GetOrdersByUserID(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, orders)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrderRepository_UpdateOrderStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewOrderRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		number := "12345678903"
		status := domain.OrderStatusProcessed
		accrual := 100.0

		mock.ExpectExec(`UPDATE orders SET status`).
			WithArgs(status, &accrual, number).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateOrderStatus(ctx, number, status, &accrual)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Order not found", func(t *testing.T) {
		number := "99999999999"
		status := domain.OrderStatusProcessed
		accrual := 100.0

		mock.ExpectExec(`UPDATE orders SET status`).
			WithArgs(status, &accrual, number).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.UpdateOrderStatus(ctx, number, status, &accrual)
		assert.ErrorIs(t, err, ErrOrderNotFound)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOrderRepository_GetPendingOrders(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewOrderRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
			AddRow(int64(1), int64(1), "111", domain.OrderStatusNew, nil, time.Now()).
			AddRow(int64(2), int64(2), "222", domain.OrderStatusProcessing, nil, time.Now())

		mock.ExpectQuery(`SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE status IN`).
			WithArgs(domain.OrderStatusNew, domain.OrderStatusProcessing).
			WillReturnRows(rows)

		orders, err := repo.GetPendingOrders(ctx)
		require.NoError(t, err)
		assert.Len(t, orders, 2)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
