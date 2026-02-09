package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// UserRepository реализует репозиторий пользователей.
type UserRepository struct {
	db DBTX
}

// NewUserRepository создает новый UserRepository
func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser создает нового пользователя
func (r *UserRepository) CreateUser(ctx context.Context, login, passwordHash string) (*domain.User, error) {
	user := &domain.User{}

	err := r.db.QueryRow(ctx,
		`INSERT INTO users (login, password_hash) 
		 VALUES ($1, $2) 
		 RETURNING id, login, password_hash, created_at`,
		login, passwordHash,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		// Проверка на уникальность логина (код ошибки PostgreSQL)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("repository: failed to create user %q: %w", login, err)
	}

	return user, nil
}

// GetUserByLogin получает пользователя по логину
func (r *UserRepository) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	user := &domain.User{}

	err := r.db.QueryRow(ctx,
		`SELECT id, login, password_hash, created_at 
		 FROM users 
		 WHERE login = $1`,
		login,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: failed to get user by login %q: %w", login, err)
	}

	return user, nil
}

// GetUserByID получает пользователя по ID
func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	user := &domain.User{}

	err := r.db.QueryRow(ctx,
		`SELECT id, login, password_hash, created_at 
		 FROM users 
		 WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: failed to get user by id %d: %w", id, err)
	}

	return user, nil
}
