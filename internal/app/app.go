package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/config"
	"github.com/avc/loyalty-system-diploma/internal/handlers"
	"github.com/avc/loyalty-system-diploma/internal/repository/postgres"
	"github.com/avc/loyalty-system-diploma/internal/service"
	"github.com/avc/loyalty-system-diploma/internal/utils/jwt"
	"github.com/avc/loyalty-system-diploma/internal/utils/password"
	"github.com/avc/loyalty-system-diploma/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Инициализация logger
	var logger *zap.Logger
	if cfg.LogLevel == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	// Подключение к БД
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Проверка подключения
	if err := dbPool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("connected to database")

	// Выполнение миграций
	if err := postgres.RunMigrations(context.Background(), dbPool, logger); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("migrations completed successfully")

	// Создание репозиториев
	userRepo := postgres.NewUserRepository(dbPool)
	orderRepo := postgres.NewOrderRepository(dbPool)
	transactionRepo := postgres.NewTransactionRepository(dbPool)

	// Создание утилит
	passwordHasher := password.NewBCryptHasher(password.DefaultCost)
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.JWTTokenTTL)

	// Создание сервисов
	authServiceConfig := service.AuthServiceConfig{
		MinPasswordLength: cfg.MinPasswordLength,
	}
	authService := service.NewAuthService(userRepo, passwordHasher, jwtManager, authServiceConfig)
	orderService := service.NewOrderService(orderRepo)
	balanceService := service.NewBalanceService(transactionRepo)
	accrualClient := service.NewAccrualClient(cfg.AccrualSystemAddress)

	// Создание handlers
	authHandler := handlers.NewAuthHandler(authService, logger)
	ordersHandler := handlers.NewOrdersHandler(orderService, logger)
	balanceHandler := handlers.NewBalanceHandler(balanceService, logger)
	healthHandler := handlers.NewHealthHandler(dbPool, logger)

	// Создание worker pool
	workerPoolConfig := worker.PoolConfig{
		Workers:      cfg.WorkerPoolSize,
		QueueSize:    cfg.WorkerQueueSize,
		ScanInterval: cfg.WorkerScanInterval,
	}
	workerPool := worker.NewPool(workerPoolConfig, orderRepo, transactionRepo, accrualClient, logger)

	// Настройка роутера
	r := chi.NewRouter()

	// Middleware
	r.Use(handlers.RequestIDMiddleware())
	r.Use(handlers.LoggingMiddleware(logger))
	r.Use(handlers.RecoveryMiddleware(logger))
	r.Use(middleware.Compress(5))

	// Health check эндпоинты (без middleware для быстрого ответа)
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	// Публичные эндпоинты
	r.Post("/api/user/register", authHandler.Register)
	r.Post("/api/user/login", authHandler.Login)

	// Защищенные эндпоинты
	r.Group(func(r chi.Router) {
		r.Use(handlers.AuthMiddleware(jwtManager))
		r.Post("/api/user/orders", ordersHandler.SubmitOrder)
		r.Get("/api/user/orders", ordersHandler.GetOrders)
		r.Get("/api/user/balance", balanceHandler.GetBalance)
		r.Post("/api/user/balance/withdraw", balanceHandler.Withdraw)
		r.Get("/api/user/withdrawals", balanceHandler.GetWithdrawals)
	})

	// HTTP сервер
	server := &http.Server{
		Addr:         cfg.RunAddress,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		config:     cfg,
		logger:     logger,
		db:         dbPool,
		router:     r,
		workerPool: workerPool,
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

	// Запуск HTTP сервера в горутине
	go func() {
		a.logger.Info("starting HTTP server", zap.String("address", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.logger.Info("shutting down server...")

	// Останавливаем прием новых запросов
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("server shutdown error", zap.Error(err))
	}

	// Останавливаем worker pool
	cancel()
	a.workerPool.Stop()
	a.logger.Info("worker pool stopped")

	// Закрываем соединение с БД
	a.db.Close()
	a.logger.Info("database connection closed")

	a.logger.Info("server stopped gracefully")
	return nil
}
