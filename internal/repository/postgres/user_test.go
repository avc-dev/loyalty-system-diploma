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

func TestUserRepository_CreateUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		login := "testuser"
		passwordHash := "hashedpassword"
		expectedUser := &domain.User{
			ID:           1,
			Login:        login,
			PasswordHash: passwordHash,
			CreatedAt:    time.Now(),
		}

		rows := pgxmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
			AddRow(expectedUser.ID, expectedUser.Login, expectedUser.PasswordHash, expectedUser.CreatedAt)

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(login, passwordHash).
			WillReturnRows(rows)

		user, err := repo.CreateUser(ctx, login, passwordHash)
		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Login, user.Login)
		assert.Equal(t, expectedUser.PasswordHash, user.PasswordHash)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User already exists", func(t *testing.T) {
		login := "existinguser"
		passwordHash := "hashedpassword"

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(login, passwordHash).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		user, err := repo.CreateUser(ctx, login, passwordHash)
		assert.ErrorIs(t, err, domain.ErrUserExists)
		assert.Nil(t, user)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		login := "testuser"
		passwordHash := "hashedpassword"

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(login, passwordHash).
			WillReturnError(errors.New("database error"))

		user, err := repo.CreateUser(ctx, login, passwordHash)
		assert.Error(t, err)
		assert.Nil(t, user)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetUserByLogin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		login := "testuser"
		expectedUser := &domain.User{
			ID:           1,
			Login:        login,
			PasswordHash: "hashedpassword",
			CreatedAt:    time.Now(),
		}

		rows := pgxmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
			AddRow(expectedUser.ID, expectedUser.Login, expectedUser.PasswordHash, expectedUser.CreatedAt)

		mock.ExpectQuery(`SELECT id, login, password_hash, created_at FROM users WHERE login`).
			WithArgs(login).
			WillReturnRows(rows)

		user, err := repo.GetUserByLogin(ctx, login)
		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Login, user.Login)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User not found", func(t *testing.T) {
		login := "nonexistent"

		mock.ExpectQuery(`SELECT id, login, password_hash, created_at FROM users WHERE login`).
			WithArgs(login).
			WillReturnError(pgx.ErrNoRows)

		user, err := repo.GetUserByLogin(ctx, login)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, user)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		login := "testuser"

		mock.ExpectQuery(`SELECT id, login, password_hash, created_at FROM users WHERE login`).
			WithArgs(login).
			WillReturnError(errors.New("database error"))

		user, err := repo.GetUserByLogin(ctx, login)
		assert.Error(t, err)
		assert.Nil(t, user)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetUserByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int64(1)
		expectedUser := &domain.User{
			ID:           userID,
			Login:        "testuser",
			PasswordHash: "hashedpassword",
			CreatedAt:    time.Now(),
		}

		rows := pgxmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
			AddRow(expectedUser.ID, expectedUser.Login, expectedUser.PasswordHash, expectedUser.CreatedAt)

		mock.ExpectQuery(`SELECT id, login, password_hash, created_at FROM users WHERE id`).
			WithArgs(userID).
			WillReturnRows(rows)

		user, err := repo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Login, user.Login)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User not found", func(t *testing.T) {
		userID := int64(999)

		mock.ExpectQuery(`SELECT id, login, password_hash, created_at FROM users WHERE id`).
			WithArgs(userID).
			WillReturnError(pgx.ErrNoRows)

		user, err := repo.GetUserByID(ctx, userID)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, user)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectQuery(`SELECT id, login, password_hash, created_at FROM users WHERE id`).
			WithArgs(userID).
			WillReturnError(errors.New("database error"))

		user, err := repo.GetUserByID(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, user)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
