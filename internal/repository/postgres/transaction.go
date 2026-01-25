package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

// TransactionRepository реализует domain.TransactionRepository
type TransactionRepository struct {
	db DBTX
}

// NewTransactionRepository создает новый TransactionRepository
func NewTransactionRepository(db DBTX) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// CreateTransaction создает новую транзакцию (начисление или списание)
func (r *TransactionRepository) CreateTransaction(ctx context.Context, userID int64, orderNumber string, amount float64, txType domain.TransactionType) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO transactions (user_id, order_number, amount, type) 
		 VALUES ($1, $2, $3, $4)`,
		userID, orderNumber, amount, txType,
	)

	if err != nil {
		// Проверяем на дублирование начисления (unique constraint violation)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && txType == domain.TransactionTypeAccrual {
			return domain.ErrDuplicateAccrual
		}
		return fmt.Errorf("repository: failed to create transaction for user %d: %w", userID, err)
	}

	return nil
}

// GetBalance получает баланс пользователя через группировку транзакций
func (r *TransactionRepository) GetBalance(ctx context.Context, userID int64) (*domain.Balance, error) {
	balance := &domain.Balance{}

	err := r.db.QueryRow(ctx,
		`SELECT 
			COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) as total_accrued,
			COALESCE(SUM(CASE WHEN amount < 0 THEN ABS(amount) ELSE 0 END), 0) as total_withdrawn
		 FROM transactions 
		 WHERE user_id = $1`,
		userID,
	).Scan(&balance.Current, &balance.Withdrawn)

	if err != nil {
		return nil, fmt.Errorf("repository: failed to get balance for user %d: %w", userID, err)
	}

	// Current = accrued - withdrawn
	balance.Current = balance.Current - balance.Withdrawn

	return balance, nil
}

// GetWithdrawals получает историю списаний пользователя
func (r *TransactionRepository) GetWithdrawals(ctx context.Context, userID int64) ([]*domain.Transaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, order_number, ABS(amount) as amount, type, processed_at 
		 FROM transactions 
		 WHERE user_id = $1 AND type = $2 
		 ORDER BY processed_at DESC`,
		userID, domain.TransactionTypeWithdrawal,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: failed to get withdrawals for user %d: %w", userID, err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		tx := &domain.Transaction{}
		err := rows.Scan(&tx.ID, &tx.UserID, &tx.OrderNumber, &tx.Amount, &tx.Type, &tx.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("repository: failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: error iterating withdrawals: %w", err)
	}

	return transactions, nil
}

// WithdrawWithLock списывает средства с блокировкой для обеспечения атомарности
func (r *TransactionRepository) WithdrawWithLock(ctx context.Context, userID int64, orderNumber string, amount float64) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: failed to begin transaction for user %d: %w", userID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Rollback после Commit безопасен

	// Используем advisory lock для блокировки по user_id
	// Это предотвращает race condition при параллельных списаниях
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, userID)
	if err != nil {
		return fmt.Errorf("repository: failed to acquire lock for user %d: %w", userID, err)
	}

	// Получаем баланс
	var balance float64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) 
		FROM transactions 
		WHERE user_id = $1`, userID).Scan(&balance)

	if err != nil {
		return fmt.Errorf("repository: failed to get balance for user %d: %w", userID, err)
	}

	// Проверяем достаточность средств
	if balance < amount {
		return domain.ErrInsufficientFunds
	}

	// Создаем транзакцию списания (отрицательная сумма)
	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (user_id, order_number, amount, type) 
		 VALUES ($1, $2, $3, $4)`,
		userID, orderNumber, -amount, domain.TransactionTypeWithdrawal,
	)

	if err != nil {
		return fmt.Errorf("repository: failed to insert withdrawal transaction for order %s: %w", orderNumber, err)
	}

	// Коммитим транзакцию
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: failed to commit withdrawal transaction: %w", err)
	}

	return nil
}
