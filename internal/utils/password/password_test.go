package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// testCost используется в тестах для ускорения выполнения
const testCost = bcrypt.MinCost

func TestBCryptHasher_Hash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "Valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "Long password",
			password: "verylongpasswordwithmanychars123456789!@#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "Password with special characters",
			password: "p@ssw0rd!#$%",
			wantErr:  false,
		},
		{
			name:     "Empty password",
			password: "",
			wantErr:  true,
		},
	}

	hasher := NewBCryptHasher(testCost)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
				// Проверяем, что хеш валидный bcrypt
				assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password)))
			}
		})
	}
}

func TestBCryptHasher_Check(t *testing.T) {
	hasher := NewBCryptHasher(testCost)
	password := "mypassword123"
	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	tests := []struct {
		name     string
		hash     string
		password string
		wantErr  bool
	}{
		{
			name:     "Correct password",
			hash:     hash,
			password: password,
			wantErr:  false,
		},
		{
			name:     "Wrong password",
			hash:     hash,
			password: "wrongpassword",
			wantErr:  true,
		},
		{
			name:     "Empty password",
			hash:     hash,
			password: "",
			wantErr:  true,
		},
		{
			name:     "Empty hash",
			hash:     "",
			password: password,
			wantErr:  true,
		},
		{
			name:     "Invalid hash format",
			hash:     "invalid-hash",
			password: password,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.Check(tt.hash, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBCryptHasher_DifferentCosts(t *testing.T) {
	password := "testpassword"

	tests := []struct {
		name string
		cost int
	}{
		{
			name: "Min cost",
			cost: bcrypt.MinCost,
		},
		{
			name: "Default cost",
			cost: DefaultCost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cost > bcrypt.MinCost && testing.Short() {
				t.Skip("skipping expensive bcrypt test in short mode")
			}
			hasher := NewBCryptHasher(tt.cost)
			hash, err := hasher.Hash(password)
			require.NoError(t, err)

			err = hasher.Check(hash, password)
			assert.NoError(t, err)
		})
	}
}

func TestBCryptHasher_InvalidCost(t *testing.T) {
	// Слишком низкая стоимость должна быть заменена на DefaultCost
	hasher := NewBCryptHasher(0)
	assert.Equal(t, DefaultCost, hasher.cost)

	// Слишком высокая стоимость должна быть заменена на DefaultCost
	hasher = NewBCryptHasher(100)
	assert.Equal(t, DefaultCost, hasher.cost)
}

func TestBCryptHasher_UniqueHashes(t *testing.T) {
	hasher := NewBCryptHasher(testCost)
	password := "testpassword"

	// Один и тот же пароль должен давать разные хеши (из-за соли)
	hash1, err := hasher.Hash(password)
	require.NoError(t, err)

	hash2, err := hasher.Hash(password)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)

	// Но оба должны проходить проверку
	assert.NoError(t, hasher.Check(hash1, password))
	assert.NoError(t, hasher.Check(hash2, password))
}

func TestHashPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping expensive bcrypt test in short mode")
	}
	password := "testpassword"
	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = CheckPassword(hash, password)
	assert.NoError(t, err)
}

func TestCheckPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping expensive bcrypt test in short mode")
	}
	password := "testpassword"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	t.Run("Correct password", func(t *testing.T) {
		err := CheckPassword(hash, password)
		assert.NoError(t, err)
	})

	t.Run("Wrong password", func(t *testing.T) {
		err := CheckPassword(hash, "wrongpassword")
		assert.Error(t, err)
	})
}

func BenchmarkBCryptHasher_Hash(b *testing.B) {
	hasher := NewBCryptHasher(testCost)
	password := "testpassword"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Hash(password)
	}
}

func BenchmarkBCryptHasher_Check(b *testing.B) {
	hasher := NewBCryptHasher(testCost)
	password := "testpassword"
	hash, _ := hasher.Hash(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.Check(hash, password)
	}
}
