package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/service"
	"go.uber.org/zap"
)

// BalanceService определяет методы работы с балансом.
type BalanceService interface {
	GetBalance(ctx context.Context, userID int64) (*domain.Balance, error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error
	GetWithdrawals(ctx context.Context, userID int64) ([]*domain.Transaction, error)
}

type BalanceHandler struct {
	balanceService BalanceService
	logger         *zap.Logger
}

func NewBalanceHandler(balanceService BalanceService, logger *zap.Logger) *BalanceHandler {
	return &BalanceHandler{
		balanceService: balanceService,
		logger:         logger,
	}
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	balance, err := h.balanceService.GetBalance(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get balance", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(balance); err != nil {
		h.logger.Error("failed to encode balance response", zap.Error(err))
	}
}

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	err := h.balanceService.Withdraw(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrderNumber) {
			http.Error(w, "Unprocessable Entity", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrInsufficientFunds) {
			http.Error(w, "Payment Required", http.StatusPaymentRequired)
			return
		}
		h.logger.Error("failed to withdraw", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.balanceService.GetWithdrawals(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get withdrawals", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
		h.logger.Error("failed to encode withdrawals response", zap.Error(err))
	}
}
