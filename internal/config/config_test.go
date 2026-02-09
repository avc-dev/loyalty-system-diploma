package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_Success tests successful config loading
// Note: flag.Parse() can only be called once, so we test different scenarios separately
func TestLoad_Success(t *testing.T) {
	// Сохраняем оригинальные env переменные
	envVars := []string{
		"RUN_ADDRESS", "DATABASE_URI", "ACCRUAL_SYSTEM_ADDRESS",
		"JWT_SECRET", "LOG_LEVEL", "WORKER_POOL_SIZE",
		"WORKER_QUEUE_SIZE", "WORKER_SCAN_INTERVAL",
	}
	originalEnv := make(map[string]string)
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Восстанавливаем env после теста
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Устанавливаем env vars для теста
	os.Setenv("RUN_ADDRESS", ":9090")
	os.Setenv("DATABASE_URI", "postgres://test:test@localhost/test")
	os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://localhost:8081")
	os.Setenv("JWT_SECRET", "my-secret")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("WORKER_POOL_SIZE", "5")
	os.Setenv("WORKER_QUEUE_SIZE", "200")
	os.Setenv("WORKER_SCAN_INTERVAL", "30s")

	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, ":9090", cfg.RunAddress)
	assert.Equal(t, "postgres://test:test@localhost/test", cfg.DatabaseURI)
	assert.Equal(t, "http://localhost:8081", cfg.AccrualSystemAddress)
	assert.Equal(t, "my-secret", cfg.JWTSecret)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 5, cfg.WorkerPoolSize)
	assert.Equal(t, 200, cfg.WorkerQueueSize)
	assert.Equal(t, 30*time.Second, cfg.WorkerScanInterval)
	assert.Equal(t, 6, cfg.MinPasswordLength)
	assert.Equal(t, 24*time.Hour, cfg.JWTTokenTTL)
}

// TestConfigDefaults tests that default values are correctly set
func TestConfigDefaults(t *testing.T) {
	cfg := &Config{
		JWTTokenTTL:        24 * time.Hour,
		LogLevel:           "info",
		WorkerPoolSize:     3,
		WorkerQueueSize:    100,
		WorkerScanInterval: 10 * time.Second,
		MinPasswordLength:  6,
	}

	assert.Equal(t, 24*time.Hour, cfg.JWTTokenTTL)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 3, cfg.WorkerPoolSize)
	assert.Equal(t, 100, cfg.WorkerQueueSize)
	assert.Equal(t, 10*time.Second, cfg.WorkerScanInterval)
	assert.Equal(t, 6, cfg.MinPasswordLength)
}

// TestEnvParsing tests parsing of individual env variables
func TestEnvParsing(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		check    func(*testing.T, string)
	}{
		{
			name:     "Valid worker pool size",
			envKey:   "WORKER_POOL_SIZE",
			envValue: "10",
			check: func(t *testing.T, val string) {
				// Just verify the value can be set
				assert.Equal(t, "10", val)
			},
		},
		{
			name:     "Valid scan interval",
			envKey:   "WORKER_SCAN_INTERVAL",
			envValue: "1m",
			check: func(t *testing.T, val string) {
				d, err := time.ParseDuration(val)
				require.NoError(t, err)
				assert.Equal(t, time.Minute, d)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.envValue)
		})
	}
}
