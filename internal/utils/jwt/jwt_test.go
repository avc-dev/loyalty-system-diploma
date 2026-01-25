package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_Generate(t *testing.T) {
	tests := []struct {
		name      string
		secretKey string
		tokenTTL  time.Duration
		userID    int64
		wantErr   bool
	}{
		{
			name:      "Valid token generation",
			secretKey: "test-secret-key",
			tokenTTL:  time.Hour,
			userID:    12345,
			wantErr:   false,
		},
		{
			name:      "Generate with different user ID",
			secretKey: "another-secret",
			tokenTTL:  time.Minute * 30,
			userID:    99999,
			wantErr:   false,
		},
		{
			name:      "Generate with zero user ID",
			secretKey: "secret",
			tokenTTL:  time.Hour,
			userID:    0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.secretKey, tt.tokenTTL)
			token, err := m.Generate(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestManager_Validate(t *testing.T) {
	secretKey := "test-secret-key"
	tokenTTL := time.Hour
	userID := int64(12345)

	t.Run("Valid token", func(t *testing.T) {
		m := NewManager(secretKey, tokenTTL)
		token, err := m.Generate(userID)
		require.NoError(t, err)

		parsedUserID, err := m.Validate(token)
		require.NoError(t, err)
		assert.Equal(t, userID, parsedUserID)
	})

	t.Run("Invalid token - wrong secret", func(t *testing.T) {
		m1 := NewManager(secretKey, tokenTTL)
		token, err := m1.Generate(userID)
		require.NoError(t, err)

		m2 := NewManager("wrong-secret", tokenTTL)
		_, err = m2.Validate(token)
		assert.Error(t, err)
	})

	t.Run("Invalid token - malformed", func(t *testing.T) {
		m := NewManager(secretKey, tokenTTL)
		_, err := m.Validate("invalid.token.string")
		assert.Error(t, err)
	})

	t.Run("Invalid token - empty", func(t *testing.T) {
		m := NewManager(secretKey, tokenTTL)
		_, err := m.Validate("")
		assert.Error(t, err)
	})

	t.Run("Expired token", func(t *testing.T) {
		m := NewManager(secretKey, time.Nanosecond)
		token, err := m.Generate(userID)
		require.NoError(t, err)

		// Ждем, чтобы токен истек
		time.Sleep(time.Millisecond * 10)

		_, err = m.Validate(token)
		assert.Error(t, err)
	})

	t.Run("Multiple users", func(t *testing.T) {
		m := NewManager(secretKey, tokenTTL)

		userID1 := int64(100)
		userID2 := int64(200)

		token1, err := m.Generate(userID1)
		require.NoError(t, err)

		token2, err := m.Generate(userID2)
		require.NoError(t, err)

		parsedID1, err := m.Validate(token1)
		require.NoError(t, err)
		assert.Equal(t, userID1, parsedID1)

		parsedID2, err := m.Validate(token2)
		require.NoError(t, err)
		assert.Equal(t, userID2, parsedID2)
	})
}

func TestManager_ValidateWithInvalidSigningMethod(t *testing.T) {
	// Создаем токен с неправильным методом подписи
	m := NewManager("secret", time.Hour)

	// Попытка валидации токена с неправильной структурой
	_, err := m.Validate("eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoxMjM0NX0.")
	assert.Error(t, err)
}

func BenchmarkManager_Generate(b *testing.B) {
	m := NewManager("test-secret-key", time.Hour)
	userID := int64(12345)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Generate(userID)
	}
}

func BenchmarkManager_Validate(b *testing.B) {
	m := NewManager("test-secret-key", time.Hour)
	userID := int64(12345)
	token, _ := m.Generate(userID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Validate(token)
	}
}
