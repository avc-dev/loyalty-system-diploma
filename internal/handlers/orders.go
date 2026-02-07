package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/service"
	"go.uber.org/zap"
)

// OrderService определяет методы работы с заказами.
type OrderService interface {
	SubmitOrder(ctx context.Context, userID int64, orderNumber string) error
	GetOrders(ctx context.Context, userID int64) ([]*domain.Order, error)
}

type OrdersHandler struct {
	orderService OrderService
	logger       *zap.Logger
}

func NewOrdersHandler(orderService OrderService, logger *zap.Logger) *OrdersHandler {
	return &OrdersHandler{
		orderService: orderService,
		logger:       logger,
	}
}

func (h *OrdersHandler) SubmitOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	err = h.orderService.SubmitOrder(r.Context(), userID, orderNumber)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrderNumber) {
			http.Error(w, "Unprocessable Entity", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrOrderExists) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, service.ErrOrderOwnedByAnother) {
			http.Error(w, "Conflict", http.StatusConflict)
			return
		}
		h.logger.Error("failed to submit order", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *OrdersHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.orderService.GetOrders(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get orders", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		h.logger.Error("failed to encode orders response", zap.Error(err))
	}
}
