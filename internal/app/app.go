package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/avc/loyalty-system-diploma/internal/config"
	"github.com/avc/loyalty-system-diploma/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// App представляет приложение
type App struct {
	config     *config.Config
	logger     *zap.Logger
	db         *pgxpool.Pool
	router     *chi.Mux
	workerPool *worker.Pool
	server     *http.Server
}

// NewApp создает новое приложение
func NewApp() (*App, error) {
	ctx := context.Background()

	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Инициализация логгера
	logger, err := initLogger(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	// Инициализация базы данных
	dbPool, err := initDatabase(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}
	logger.Info("connected to database")

	// Выполнение миграций
	if err := runMigrations(ctx, dbPool, logger); err != nil {
		return nil, err
	}
	logger.Info("migrations completed successfully")

	// Инициализация зависимостей
	deps := initDependencies(cfg, dbPool, logger)

	// Настройка роутера
	router := setupRouter(deps, deps.jwtManager, logger)

	// Создание HTTP сервера
	server := createServer(cfg.RunAddress, router)

	return &App{
		config:     cfg,
		logger:     logger,
		db:         dbPool,
		router:     router,
		workerPool: deps.workerPool,
		server:     server,
	}, nil
}

// Run запускает приложение
func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск worker pool
	a.workerPool.Start(ctx)
	a.logger.Info("worker pool started")

	// Запуск HTTP сервера и ожидание сигнала завершения
	if err := a.runServer(ctx); err != nil {
		return err
	}

	// Graceful shutdown
	a.shutdown(cancel)

	return nil
}
