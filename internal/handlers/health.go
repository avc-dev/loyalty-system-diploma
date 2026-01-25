package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// HealthHandler обрабатывает health check запросы
type HealthHandler struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewHealthHandler создает новый HealthHandler
func NewHealthHandler(db *pgxpool.Pool, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// HealthResponse представляет ответ health check
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// Health возвращает статус приложения
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:   "ok",
		Database: "ok",
	}

	// Проверяем подключение к БД с таймаутом
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		response.Status = "degraded"
		response.Database = "unavailable"
		h.logger.Warn("health check: database unavailable", zap.Error(err))
	}

	w.Header().Set("Content-Type", "application/json")
	if response.Status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode health response", zap.Error(err))
	}
}

// Ready возвращает готовность приложения принимать трафик
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// Проверяем подключение к БД
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		h.logger.Warn("readiness check failed: database unavailable", zap.Error(err))
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
