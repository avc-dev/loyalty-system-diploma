package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	domainmocks "github.com/avc/loyalty-system-diploma/internal/domain/mocks"
	"github.com/avc/loyalty-system-diploma/internal/utils/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAuthHandler_Register(t *testing.T) {
	mockService := domainmocks.NewAuthServiceMock(t)
	logger, _ := zap.NewDevelopment()
	handler := NewAuthHandler(mockService, logger)

	t.Run("Success", func(t *testing.T) {
		mockService.EXPECT().Register(mock.Anything, "user", "pass").Return("token", nil).Once()

		body := `{"login":"user","password":"pass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		handler.Register(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Authorization"), "Bearer token")
	})

	t.Run("User exists", func(t *testing.T) {
		mockService.EXPECT().Register(mock.Anything, "user", "pass").Return("", domain.ErrUserExists).Once()

		body := `{"login":"user","password":"pass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		handler.Register(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		body := `{"login":}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		handler.Register(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOrdersHandler_SubmitOrder(t *testing.T) {
	mockService := domainmocks.NewOrderServiceMock(t)
	logger, _ := zap.NewDevelopment()
	handler := NewOrdersHandler(mockService, logger)

	t.Run("Success", func(t *testing.T) {
		mockService.EXPECT().SubmitOrder(mock.Anything, int64(1), "79927398713").Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
		ctx := context.WithValue(req.Context(), UserIDKey, int64(1))
		w := httptest.NewRecorder()

		handler.SubmitOrder(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("Order exists - same user", func(t *testing.T) {
		mockService.EXPECT().SubmitOrder(mock.Anything, int64(1), "79927398713").Return(domain.ErrOrderExists).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
		ctx := context.WithValue(req.Context(), UserIDKey, int64(1))
		w := httptest.NewRecorder()

		handler.SubmitOrder(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Order owned by another user", func(t *testing.T) {
		mockService.EXPECT().SubmitOrder(mock.Anything, int64(1), "79927398713").Return(domain.ErrOrderOwnedByAnother).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
		ctx := context.WithValue(req.Context(), UserIDKey, int64(1))
		w := httptest.NewRecorder()

		handler.SubmitOrder(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Invalid order number", func(t *testing.T) {
		mockService.EXPECT().SubmitOrder(mock.Anything, int64(1), "12345").Return(domain.ErrInvalidOrderNumber).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("12345"))
		ctx := context.WithValue(req.Context(), UserIDKey, int64(1))
		w := httptest.NewRecorder()

		handler.SubmitOrder(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Unauthorized - no user ID in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
		w := httptest.NewRecorder()

		handler.SubmitOrder(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestBalanceHandler_GetBalance(t *testing.T) {
	mockService := domainmocks.NewBalanceServiceMock(t)
	logger, _ := zap.NewDevelopment()
	handler := NewBalanceHandler(mockService, logger)

	t.Run("Success", func(t *testing.T) {
		balance := &domain.Balance{Current: 500.0, Withdrawn: 200.0}
		mockService.EXPECT().GetBalance(mock.Anything, int64(1)).Return(balance, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
		ctx := context.WithValue(req.Context(), UserIDKey, int64(1))
		w := httptest.NewRecorder()

		handler.GetBalance(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusOK, w.Code)

		var result domain.Balance
		err := json.NewDecoder(w.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, balance.Current, result.Current)
		assert.Equal(t, balance.Withdrawn, result.Withdrawn)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
		w := httptest.NewRecorder()

		handler.GetBalance(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthMiddleware(t *testing.T) {
	jwtManager := jwt.NewManager("test-secret", time.Hour)
	token, _ := jwtManager.Generate(123)

	middleware := AuthMiddleware(jwtManager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		assert.True(t, ok)
		assert.Equal(t, int64(123), userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
