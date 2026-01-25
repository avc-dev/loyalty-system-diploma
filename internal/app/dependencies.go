package app

import (
	"github.com/avc/loyalty-system-diploma/internal/config"
	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/handlers"
	"github.com/avc/loyalty-system-diploma/internal/repository/postgres"
	"github.com/avc/loyalty-system-diploma/internal/service"
	"github.com/avc/loyalty-system-diploma/internal/utils/jwt"
	"github.com/avc/loyalty-system-diploma/internal/utils/password"
	"github.com/avc/loyalty-system-diploma/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// repositories содержит все репозитории приложения
type repositories struct {
	user        domain.UserRepository
	order       domain.OrderRepository
	transaction domain.TransactionRepository
}

// services содержит все сервисы приложения
type services struct {
	auth    domain.AuthService
	order   domain.OrderService
	balance domain.BalanceService
	accrual domain.AccrualClient
}

// handlerSet содержит все хендлеры приложения
type handlerSet struct {
	auth    *handlers.AuthHandler
	orders  *handlers.OrdersHandler
	balance *handlers.BalanceHandler
	health  *handlers.HealthHandler
}

// dependencies содержит все зависимости приложения
type dependencies struct {
	repos       *repositories
	services    *services
	handlers    *handlerSet
	jwtManager  *jwt.Manager
	workerPool  *worker.Pool
}

// initDependencies создает все зависимости приложения
func initDependencies(cfg *config.Config, dbPool *pgxpool.Pool, logger *zap.Logger) *dependencies {
	// Создание репозиториев
	repos := &repositories{
		user:        postgres.NewUserRepository(dbPool),
		order:       postgres.NewOrderRepository(dbPool),
		transaction: postgres.NewTransactionRepository(dbPool),
	}

	// Создание утилит
	passwordHasher := password.NewBCryptHasher(password.DefaultCost)
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.JWTTokenTTL)

	// Создание сервисов
	authServiceConfig := service.AuthServiceConfig{
		MinPasswordLength: cfg.MinPasswordLength,
	}
	svcs := &services{
		auth:    service.NewAuthService(repos.user, passwordHasher, jwtManager, authServiceConfig),
		order:   service.NewOrderService(repos.order),
		balance: service.NewBalanceService(repos.transaction),
		accrual: service.NewAccrualClient(cfg.AccrualSystemAddress),
	}

	// Создание handlers
	hdlrs := &handlerSet{
		auth:    handlers.NewAuthHandler(svcs.auth, logger),
		orders:  handlers.NewOrdersHandler(svcs.order, logger),
		balance: handlers.NewBalanceHandler(svcs.balance, logger),
		health:  handlers.NewHealthHandler(dbPool, logger),
	}

	// Создание worker pool
	workerPoolConfig := worker.PoolConfig{
		Workers:      cfg.WorkerPoolSize,
		QueueSize:    cfg.WorkerQueueSize,
		ScanInterval: cfg.WorkerScanInterval,
	}
	workerPool := worker.NewPool(workerPoolConfig, repos.order, repos.transaction, svcs.accrual, logger)

	return &dependencies{
		repos:      repos,
		services:   svcs,
		handlers:   hdlrs,
		jwtManager: jwtManager,
		workerPool: workerPool,
	}
}
