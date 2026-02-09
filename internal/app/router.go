package app

import (
	"github.com/avc/loyalty-system-diploma/internal/handlers"
	"github.com/avc/loyalty-system-diploma/internal/utils/jwt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// setupRouter создает и настраивает роутер
func setupRouter(deps *dependencies, jwtManager *jwt.Manager, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Глобальные middleware
	setupMiddleware(r, logger)

	// Маршруты
	setupRoutes(r, deps, jwtManager)

	return r
}

// setupMiddleware настраивает middleware для роутера
func setupMiddleware(r *chi.Mux, logger *zap.Logger) {
	r.Use(handlers.RequestIDMiddleware())
	r.Use(handlers.LoggingMiddleware(logger))
	r.Use(handlers.RecoveryMiddleware(logger))
	r.Use(middleware.Compress(5))
}

// setupRoutes настраивает маршруты приложения
func setupRoutes(r *chi.Mux, deps *dependencies, jwtManager *jwt.Manager) {
	// Health check эндпоинты
	r.Get("/health", deps.handlers.health.Health)
	r.Get("/ready", deps.handlers.health.Ready)

	// Публичные эндпоинты
	r.Post("/api/user/register", deps.handlers.auth.Register)
	r.Post("/api/user/login", deps.handlers.auth.Login)

	// Защищенные эндпоинты
	r.Group(func(r chi.Router) {
		r.Use(handlers.AuthMiddleware(jwtManager))
		r.Post("/api/user/orders", deps.handlers.orders.SubmitOrder)
		r.Get("/api/user/orders", deps.handlers.orders.GetOrders)
		r.Get("/api/user/balance", deps.handlers.balance.GetBalance)
		r.Post("/api/user/balance/withdraw", deps.handlers.balance.Withdraw)
		r.Get("/api/user/withdrawals", deps.handlers.balance.GetWithdrawals)
	})
}
