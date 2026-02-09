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
	if envRunAddr, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		cfg.RunAddress = envRunAddr
	}

	if envDBURI, ok := os.LookupEnv("DATABASE_URI"); ok {
		cfg.DatabaseURI = envDBURI
	}

	if envAccrualAddr, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		cfg.AccrualSystemAddress = envAccrualAddr
	}

	// JWT секрет (только из env, не из флагов для безопасности)
	if envJWTSecret, ok := os.LookupEnv("JWT_SECRET"); ok {
		cfg.JWTSecret = envJWTSecret
	} else {
		cfg.JWTSecret = "default-secret-key-change-in-production"
	}

	// Уровень логирования
	if envLogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.LogLevel = envLogLevel
	}

	// Worker Pool конфигурация из env
	if envWorkerPoolSize, ok := os.LookupEnv("WORKER_POOL_SIZE"); ok {
		if size, err := strconv.Atoi(envWorkerPoolSize); err == nil && size > 0 {
			cfg.WorkerPoolSize = size
		}
	}

	if envWorkerQueueSize, ok := os.LookupEnv("WORKER_QUEUE_SIZE"); ok {
		if size, err := strconv.Atoi(envWorkerQueueSize); err == nil && size > 0 {
			cfg.WorkerQueueSize = size
		}
	}

	if envScanInterval, ok := os.LookupEnv("WORKER_SCAN_INTERVAL"); ok {
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
