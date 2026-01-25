package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost стоимость хеширования по умолчанию
	DefaultCost = bcrypt.DefaultCost
)

// Hasher интерфейс для хеширования паролей
type Hasher interface {
	Hash(password string) (string, error)
	Check(hash, password string) error
}

// BCryptHasher реализация хеширования через bcrypt
type BCryptHasher struct {
	cost int
}

// NewBCryptHasher создает новый hasher с заданной стоимостью
func NewBCryptHasher(cost int) *BCryptHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = DefaultCost
	}
	return &BCryptHasher{
		cost: cost,
	}
}

// Hash хеширует пароль
func (h *BCryptHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// Check проверяет соответствие пароля хешу
func (h *BCryptHasher) Check(hash, password string) error {
	if hash == "" || password == "" {
		return fmt.Errorf("hash and password cannot be empty")
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return fmt.Errorf("password does not match")
		}
		return fmt.Errorf("failed to check password: %w", err)
	}

	return nil
}

// HashPassword хеширует пароль с дефолтной стоимостью (удобная функция)
func HashPassword(password string) (string, error) {
	hasher := NewBCryptHasher(DefaultCost)
	return hasher.Hash(password)
}

// CheckPassword проверяет пароль (удобная функция)
func CheckPassword(hash, password string) error {
	hasher := NewBCryptHasher(DefaultCost)
	return hasher.Check(hash, password)
}
