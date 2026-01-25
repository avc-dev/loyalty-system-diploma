package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/utils/jwt"
	"github.com/avc/loyalty-system-diploma/internal/utils/password"
)

// AuthService реализует domain.AuthService
type AuthService struct {
	userRepo       domain.UserRepository
	passwordHasher password.Hasher
	jwtManager     *jwt.Manager
}

// NewAuthService создает новый AuthService
func NewAuthService(
	userRepo domain.UserRepository,
	passwordHasher password.Hasher,
	jwtManager *jwt.Manager,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		jwtManager:     jwtManager,
	}
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(ctx context.Context, login, userPassword string) (string, error) {
	// Валидация входных данных
	if login == "" || userPassword == "" {
		return "", fmt.Errorf("auth service: empty login or password")
	}

	// Хеширование пароля
	hash, err := s.passwordHasher.Hash(userPassword)
	if err != nil {
		return "", fmt.Errorf("auth service: failed to hash password for user %q: %w", login, err)
	}

	// Создание пользователя
	user, err := s.userRepo.CreateUser(ctx, login, hash)
	if err != nil {
		// Не оборачиваем sentinel error
		if errors.Is(err, domain.ErrUserExists) {
			return "", err
		}
		return "", fmt.Errorf("auth service: failed to register user %q: %w", login, err)
	}

	// Генерация JWT токена
	token, err := s.jwtManager.Generate(user.ID)
	if err != nil {
		return "", fmt.Errorf("auth service: failed to generate token for user %d: %w", user.ID, err)
	}

	return token, nil
}

// Login аутентифицирует пользователя
func (s *AuthService) Login(ctx context.Context, login, userPassword string) (string, error) {
	// Валидация входных данных
	if login == "" || userPassword == "" {
		return "", fmt.Errorf("auth service: empty login or password")
	}

	// Получение пользователя по логину
	user, err := s.userRepo.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", fmt.Errorf("auth service: failed to get user %q: %w", login, err)
	}

	// Проверка пароля
	err = s.passwordHasher.Check(user.PasswordHash, userPassword)
	if err != nil {
		return "", domain.ErrInvalidCredentials
	}

	// Генерация JWT токена
	token, err := s.jwtManager.Generate(user.ID)
	if err != nil {
		return "", fmt.Errorf("auth service: failed to generate token for user %d: %w", user.ID, err)
	}

	return token, nil
}
