package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/service"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService domain.AuthService
	logger      *zap.Logger
}

func NewAuthHandler(authService domain.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	token, err := h.authService.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			http.Error(w, "Conflict", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to register", zap.Error(err), zap.String("login", req.Login))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to login", zap.Error(err), zap.String("login", req.Login))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}
