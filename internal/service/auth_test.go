package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	domainmocks "github.com/avc/loyalty-system-diploma/internal/domain/mocks"
	"github.com/avc/loyalty-system-diploma/internal/utils/jwt"
	passwordmocks "github.com/avc/loyalty-system-diploma/internal/utils/password/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Register(t *testing.T) {
	mockUserRepo := domainmocks.NewUserRepositoryMock(t)
	mockHasher := passwordmocks.NewHasherMock(t)
	jwtManager := jwt.NewManager("test-secret", time.Hour)
	svc := NewAuthService(mockUserRepo, mockHasher, jwtManager)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		login := "testuser"
		pwd := "password123"
		passwordHash := "hashed_password"
		user := &domain.User{ID: 1, Login: login, PasswordHash: passwordHash}

		mockHasher.EXPECT().Hash(pwd).Return(passwordHash, nil).Once()
		mockUserRepo.EXPECT().CreateUser(mock.Anything, login, passwordHash).Return(user, nil).Once()

		token, err := svc.Register(ctx, login, pwd)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("Empty login", func(t *testing.T) {
		token, err := svc.Register(ctx, "", "password")
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	t.Run("Empty password", func(t *testing.T) {
		token, err := svc.Register(ctx, "testuser", "")
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	t.Run("Hash password error", func(t *testing.T) {
		login := "testuser"
		pwd := "password123"

		mockHasher.EXPECT().Hash(pwd).Return("", errors.New("hash error")).Once()

		token, err := svc.Register(ctx, login, pwd)
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	t.Run("User already exists", func(t *testing.T) {
		login := "existinguser"
		pwd := "password123"
		passwordHash := "hashed_password"

		mockHasher.EXPECT().Hash(pwd).Return(passwordHash, nil).Once()
		mockUserRepo.EXPECT().CreateUser(mock.Anything, login, passwordHash).Return(nil, domain.ErrUserExists).Once()

		token, err := svc.Register(ctx, login, pwd)
		assert.ErrorIs(t, err, domain.ErrUserExists)
		assert.Empty(t, token)
	})

	t.Run("Database error", func(t *testing.T) {
		login := "testuser"
		pwd := "password123"
		passwordHash := "hashed_password"

		mockHasher.EXPECT().Hash(pwd).Return(passwordHash, nil).Once()
		mockUserRepo.EXPECT().CreateUser(mock.Anything, login, passwordHash).Return(nil, errors.New("db error")).Once()

		token, err := svc.Register(ctx, login, pwd)
		assert.Error(t, err)
		assert.Empty(t, token)
	})
}

func TestAuthService_Login(t *testing.T) {
	mockUserRepo := domainmocks.NewUserRepositoryMock(t)
	mockHasher := passwordmocks.NewHasherMock(t)
	jwtManager := jwt.NewManager("test-secret", time.Hour)
	svc := NewAuthService(mockUserRepo, mockHasher, jwtManager)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		login := "testuser"
		pwd := "password123"
		passwordHash := "hashed_password"
		user := &domain.User{ID: 1, Login: login, PasswordHash: passwordHash}

		mockUserRepo.EXPECT().GetUserByLogin(mock.Anything, login).Return(user, nil).Once()
		mockHasher.EXPECT().Check(passwordHash, pwd).Return(nil).Once()

		token, err := svc.Login(ctx, login, pwd)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("Empty login", func(t *testing.T) {
		token, err := svc.Login(ctx, "", "password")
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	t.Run("Empty password", func(t *testing.T) {
		token, err := svc.Login(ctx, "testuser", "")
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	t.Run("User not found", func(t *testing.T) {
		login := "nonexistent"
		pwd := "password123"

		mockUserRepo.EXPECT().GetUserByLogin(mock.Anything, login).Return(nil, domain.ErrUserNotFound).Once()

		token, err := svc.Login(ctx, login, pwd)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		assert.Empty(t, token)
	})

	t.Run("Wrong password", func(t *testing.T) {
		login := "testuser"
		pwd := "wrongpassword"
		passwordHash := "hashed_password"
		user := &domain.User{ID: 1, Login: login, PasswordHash: passwordHash}

		mockUserRepo.EXPECT().GetUserByLogin(mock.Anything, login).Return(user, nil).Once()
		mockHasher.EXPECT().Check(passwordHash, pwd).Return(errors.New("password mismatch")).Once()

		token, err := svc.Login(ctx, login, pwd)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		assert.Empty(t, token)
	})

	t.Run("Database error", func(t *testing.T) {
		login := "testuser"
		pwd := "password123"

		mockUserRepo.EXPECT().GetUserByLogin(mock.Anything, login).Return(nil, errors.New("db error")).Once()

		token, err := svc.Login(ctx, login, pwd)
		assert.Error(t, err)
		assert.Empty(t, token)
	})
}
