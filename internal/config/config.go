package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config содержит конфигурацию приложения
type Config struct {
	RunAddress           string        // Адрес и порт запуска сервиса
	DatabaseURI          string        // URI подключения к БД
	AccrualSystemAddress string        // Адрес системы расчета начислений
	JWTSecret            string        // Секретный ключ для JWT
	JWTTokenTTL          time.Duration // Время жизни JWT токена
	LogLevel             string        // Уровень логирования

	// Worker Pool конфигурация
	WorkerPoolSize     int           // Количество воркеров
	WorkerQueueSize    int           // Размер очереди заказов
	WorkerScanInterval time.Duration // Интервал сканирования pending заказов

	// Валидация
	MinPasswordLength int // Минимальная длина пароля
}

// Load загружает конфигурацию из переменных окружения и флагов
// Приоритет: env переменные > флаги > дефолтные значения
func Load() (*Config, error) {
	cfg := &Config{
		JWTTokenTTL:        24 * time.Hour,
		LogLevel:           "info",
		WorkerPoolSize:     3,
		WorkerQueueSize:    100,
		WorkerScanInterval: 10 * time.Second,
		MinPasswordLength:  6,
	}

	// Определяем флаги
	flag.StringVar(&cfg.RunAddress, "a", ":8080", "address and port to run server")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "database URI")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "accrual system address")
	flag.Parse()

	// Переменные окружения имеют приоритет над флагами
	if envRunAddr := os.Getenv("RUN_ADDRESS"); envRunAddr != "" {
		cfg.RunAddress = envRunAddr
	}

	if envDBURI := os.Getenv("DATABASE_URI"); envDBURI != "" {
		cfg.DatabaseURI = envDBURI
	}

	if envAccrualAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualAddr != "" {
		cfg.AccrualSystemAddress = envAccrualAddr
	}

	// JWT секрет (только из env, не из флагов для безопасности)
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "default-secret-key-change-in-production"
	}

	// Уровень логирования
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	// Worker Pool конфигурация из env
	if envWorkerPoolSize := os.Getenv("WORKER_POOL_SIZE"); envWorkerPoolSize != "" {
		if size, err := strconv.Atoi(envWorkerPoolSize); err == nil && size > 0 {
			cfg.WorkerPoolSize = size
		}
	}

	if envWorkerQueueSize := os.Getenv("WORKER_QUEUE_SIZE"); envWorkerQueueSize != "" {
		if size, err := strconv.Atoi(envWorkerQueueSize); err == nil && size > 0 {
			cfg.WorkerQueueSize = size
		}
	}

	if envScanInterval := os.Getenv("WORKER_SCAN_INTERVAL"); envScanInterval != "" {
		if interval, err := time.ParseDuration(envScanInterval); err == nil && interval > 0 {
			cfg.WorkerScanInterval = interval
		}
	}

	// Валидация обязательных параметров
	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("database URI is required (use -d flag or DATABASE_URI env)")
	}

	if cfg.AccrualSystemAddress == "" {
		return nil, fmt.Errorf("accrual system address is required (use -r flag or ACCRUAL_SYSTEM_ADDRESS env)")
	}

	return cfg, nil
}
