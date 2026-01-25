package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionRepository_CreateTransaction(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewTransactionRepository(mock)
	ctx := context.Background()

	t.Run("Success - accrual", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345678903"
		amount := 100.0

		mock.ExpectExec(`INSERT INTO transactions`).
			WithArgs(userID, orderNumber, amount, domain.TransactionTypeAccrual).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.CreateTransaction(ctx, userID, orderNumber, amount, domain.TransactionTypeAccrual)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success - withdrawal", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345678903"
		amount := -50.0

		mock.ExpectExec(`INSERT INTO transactions`).
			WithArgs(userID, orderNumber, amount, domain.TransactionTypeWithdrawal).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.CreateTransaction(ctx, userID, orderNumber, amount, domain.TransactionTypeWithdrawal)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345678903"
		amount := 100.0

		mock.ExpectExec(`INSERT INTO transactions`).
			WithArgs(userID, orderNumber, amount, domain.TransactionTypeAccrual).
			WillReturnError(errors.New("database error"))

		err := repo.CreateTransaction(ctx, userID, orderNumber, amount, domain.TransactionTypeAccrual)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTransactionRepository_GetBalance(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewTransactionRepository(mock)
	ctx := context.Background()

	t.Run("Success - with balance", func(t *testing.T) {
		userID := int64(1)
		totalAccrued := 500.0
		totalWithdrawn := 200.0

		rows := pgxmock.NewRows([]string{"total_accrued", "total_withdrawn"}).
			AddRow(totalAccrued, totalWithdrawn)

		mock.ExpectQuery(`SELECT`).
			WithArgs(userID).
			WillReturnRows(rows)

		balance, err := repo.GetBalance(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 300.0, balance.Current) // 500 - 200
		assert.Equal(t, 200.0, balance.Withdrawn)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success - no transactions", func(t *testing.T) {
		userID := int64(999)

		rows := pgxmock.NewRows([]string{"total_accrued", "total_withdrawn"}).
			AddRow(0.0, 0.0)

		mock.ExpectQuery(`SELECT`).
			WithArgs(userID).
			WillReturnRows(rows)

		balance, err := repo.GetBalance(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 0.0, balance.Current)
		assert.Equal(t, 0.0, balance.Withdrawn)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectQuery(`SELECT`).
			WithArgs(userID).
			WillReturnError(errors.New("database error"))

		balance, err := repo.GetBalance(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, balance)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTransactionRepository_GetWithdrawals(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewTransactionRepository(mock)
	ctx := context.Background()

	t.Run("Success - with withdrawals", func(t *testing.T) {
		userID := int64(1)

		rows := pgxmock.NewRows([]string{"id", "user_id", "order_number", "amount", "type", "processed_at"}).
			AddRow(int64(1), userID, "111", 100.0, domain.TransactionTypeWithdrawal, time.Now()).
			AddRow(int64(2), userID, "222", 50.0, domain.TransactionTypeWithdrawal, time.Now())

		mock.ExpectQuery(`SELECT id, user_id, order_number, ABS\(amount\) as amount, type, processed_at FROM transactions WHERE user_id`).
			WithArgs(userID, domain.TransactionTypeWithdrawal).
			WillReturnRows(rows)

		transactions, err := repo.GetWithdrawals(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, transactions, 2)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success - no withdrawals", func(t *testing.T) {
		userID := int64(999)

		rows := pgxmock.NewRows([]string{"id", "user_id", "order_number", "amount", "type", "processed_at"})

		mock.ExpectQuery(`SELECT id, user_id, order_number, ABS\(amount\) as amount, type, processed_at FROM transactions WHERE user_id`).
			WithArgs(userID, domain.TransactionTypeWithdrawal).
			WillReturnRows(rows)

		transactions, err := repo.GetWithdrawals(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, transactions)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTransactionRepository_WithdrawWithLock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewTransactionRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345678903"
		amount := 100.0
		currentBalance := 500.0

		mock.ExpectBegin()

		mock.ExpectExec(`SELECT pg_advisory_xact_lock`).
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("SELECT", 1))

		balanceRows := pgxmock.NewRows([]string{"balance"}).AddRow(currentBalance)
		mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions WHERE user_id`).
			WithArgs(userID).
			WillReturnRows(balanceRows)

		mock.ExpectExec(`INSERT INTO transactions`).
			WithArgs(userID, orderNumber, -amount, domain.TransactionTypeWithdrawal).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		mock.ExpectCommit()

		err := repo.WithdrawWithLock(ctx, userID, orderNumber, amount)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Insufficient funds", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345678903"
		amount := 500.0
		currentBalance := 100.0

		mock.ExpectBegin()

		mock.ExpectExec(`SELECT pg_advisory_xact_lock`).
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("SELECT", 1))

		balanceRows := pgxmock.NewRows([]string{"balance"}).AddRow(currentBalance)
		mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions WHERE user_id`).
			WithArgs(userID).
			WillReturnRows(balanceRows)

		mock.ExpectRollback()

		err := repo.WithdrawWithLock(ctx, userID, orderNumber, amount)
		assert.ErrorIs(t, err, domain.ErrInsufficientFunds)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Begin transaction error", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345678903"
		amount := 100.0

		mock.ExpectBegin().WillReturnError(errors.New("begin error"))

		err := repo.WithdrawWithLock(ctx, userID, orderNumber, amount)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Get balance error", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345678903"
		amount := 100.0

		mock.ExpectBegin()

		mock.ExpectExec(`SELECT pg_advisory_xact_lock`).
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("SELECT", 1))

		mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions WHERE user_id`).
			WithArgs(userID).
			WillReturnError(errors.New("query error"))

		mock.ExpectRollback()

		err := repo.WithdrawWithLock(ctx, userID, orderNumber, amount)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Insert transaction error", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "12345678903"
		amount := 100.0
		currentBalance := 500.0

		mock.ExpectBegin()

		mock.ExpectExec(`SELECT pg_advisory_xact_lock`).
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("SELECT", 1))

		balanceRows := pgxmock.NewRows([]string{"balance"}).AddRow(currentBalance)
		mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions WHERE user_id`).
			WithArgs(userID).
			WillReturnRows(balanceRows)

		mock.ExpectExec(`INSERT INTO transactions`).
			WithArgs(userID, orderNumber, -amount, domain.TransactionTypeWithdrawal).
			WillReturnError(errors.New("insert error"))

		mock.ExpectRollback()

		err := repo.WithdrawWithLock(ctx, userID, orderNumber, amount)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
