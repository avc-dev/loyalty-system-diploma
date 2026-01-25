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

func newTestAuthService(t *testing.T) (*AuthService, *domainmocks.UserRepositoryMock, *passwordmocks.HasherMock) {
	mockUserRepo := domainmocks.NewUserRepositoryMock(t)
	mockHasher := passwordmocks.NewHasherMock(t)
	jwtManager := jwt.NewManager("test-secret", time.Hour)
	config := AuthServiceConfig{MinPasswordLength: 6}
	svc := NewAuthService(mockUserRepo, mockHasher, jwtManager, config)
	return svc, mockUserRepo, mockHasher
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		login      string
		password   string
		setupMocks func(*domainmocks.UserRepositoryMock, *passwordmocks.HasherMock)
		wantToken  bool
		wantErr    error
	}{
		{
			name:     "Success",
			login:    "testuser",
			password: "password123",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {
				hasher.EXPECT().Hash("password123").Return("hashed_password", nil).Once()
				userRepo.EXPECT().CreateUser(mock.Anything, "testuser", "hashed_password").
					Return(&domain.User{ID: 1, Login: "testuser", PasswordHash: "hashed_password"}, nil).Once()
			},
			wantToken: true,
		},
		{
			name:       "Empty login",
			login:      "",
			password:   "password",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {},
			wantErr:    domain.ErrInvalidInput,
		},
		{
			name:       "Empty password",
			login:      "testuser",
			password:   "",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {},
			wantErr:    domain.ErrInvalidInput,
		},
		{
			name:       "Password too short",
			login:      "testuser",
			password:   "12345", // < 6 characters
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {},
			wantErr:    domain.ErrInvalidInput,
		},
		{
			name:     "Hash password error",
			login:    "testuser",
			password: "password123",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {
				hasher.EXPECT().Hash("password123").Return("", errors.New("hash error")).Once()
			},
			wantErr: nil, // generic error, not sentinel
		},
		{
			name:     "User already exists",
			login:    "existinguser",
			password: "password123",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {
				hasher.EXPECT().Hash("password123").Return("hashed_password", nil).Once()
				userRepo.EXPECT().CreateUser(mock.Anything, "existinguser", "hashed_password").
					Return(nil, domain.ErrUserExists).Once()
			},
			wantErr: domain.ErrUserExists,
		},
		{
			name:     "Database error",
			login:    "testuser",
			password: "password123",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {
				hasher.EXPECT().Hash("password123").Return("hashed_password", nil).Once()
				userRepo.EXPECT().CreateUser(mock.Anything, "testuser", "hashed_password").
					Return(nil, errors.New("db error")).Once()
			},
			wantErr: nil, // generic error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, userRepo, hasher := newTestAuthService(t)
			tt.setupMocks(userRepo, hasher)

			token, err := svc.Register(ctx, tt.login, tt.password)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, token)
			} else if tt.wantToken {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
			} else {
				assert.Error(t, err)
				assert.Empty(t, token)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		login      string
		password   string
		setupMocks func(*domainmocks.UserRepositoryMock, *passwordmocks.HasherMock)
		wantToken  bool
		wantErr    error
	}{
		{
			name:     "Success",
			login:    "testuser",
			password: "password123",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {
				user := &domain.User{ID: 1, Login: "testuser", PasswordHash: "hashed_password"}
				userRepo.EXPECT().GetUserByLogin(mock.Anything, "testuser").Return(user, nil).Once()
				hasher.EXPECT().Check("hashed_password", "password123").Return(nil).Once()
			},
			wantToken: true,
		},
		{
			name:       "Empty login",
			login:      "",
			password:   "password",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {},
			wantErr:    domain.ErrInvalidInput,
		},
		{
			name:       "Empty password",
			login:      "testuser",
			password:   "",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {},
			wantErr:    domain.ErrInvalidInput,
		},
		{
			name:     "User not found",
			login:    "nonexistent",
			password: "password123",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {
				userRepo.EXPECT().GetUserByLogin(mock.Anything, "nonexistent").Return(nil, domain.ErrUserNotFound).Once()
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:     "Wrong password",
			login:    "testuser",
			password: "wrongpassword",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {
				user := &domain.User{ID: 1, Login: "testuser", PasswordHash: "hashed_password"}
				userRepo.EXPECT().GetUserByLogin(mock.Anything, "testuser").Return(user, nil).Once()
				hasher.EXPECT().Check("hashed_password", "wrongpassword").Return(errors.New("password mismatch")).Once()
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:     "Database error",
			login:    "testuser",
			password: "password123",
			setupMocks: func(userRepo *domainmocks.UserRepositoryMock, hasher *passwordmocks.HasherMock) {
				userRepo.EXPECT().GetUserByLogin(mock.Anything, "testuser").Return(nil, errors.New("db error")).Once()
			},
			wantErr: nil, // generic error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, userRepo, hasher := newTestAuthService(t)
			tt.setupMocks(userRepo, hasher)

			token, err := svc.Login(ctx, tt.login, tt.password)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, token)
			} else if tt.wantToken {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
			} else {
				assert.Error(t, err)
				assert.Empty(t, token)
			}
		})
	}
}
