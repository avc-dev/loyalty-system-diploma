package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"go.uber.org/zap"
)

type OrdersHandler struct {
	orderService domain.OrderService
	logger       *zap.Logger
}

func NewOrdersHandler(orderService domain.OrderService, logger *zap.Logger) *OrdersHandler {
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
		if errors.Is(err, domain.ErrInvalidOrderNumber) {
			http.Error(w, "Unprocessable Entity", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, domain.ErrOrderExists) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, domain.ErrOrderOwnedByAnother) {
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
	json.NewEncoder(w).Encode(orders)
}
