package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	domainmocks "github.com/avc/loyalty-system-diploma/internal/domain/mocks"
	"github.com/avc/loyalty-system-diploma/internal/service"
	"github.com/avc/loyalty-system-diploma/internal/utils/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setupMock      func(*domainmocks.AuthServiceMock)
		expectedStatus int
		checkAuth      bool
	}{
		{
			name: "Success",
			body: `{"login":"user","password":"pass"}`,
			setupMock: func(m *domainmocks.AuthServiceMock) {
				m.EXPECT().Register(mock.Anything, "user", "pass").Return("token", nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkAuth:      true,
		},
		{
			name: "User exists",
			body: `{"login":"user","password":"pass"}`,
			setupMock: func(m *domainmocks.AuthServiceMock) {
				m.EXPECT().Register(mock.Anything, "user", "pass").Return("", service.ErrUserExists).Once()
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "Invalid input",
			body: `{"login":"user","password":"pass"}`,
			setupMock: func(m *domainmocks.AuthServiceMock) {
				m.EXPECT().Register(mock.Anything, "user", "pass").Return("", service.ErrInvalidInput).Once()
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			body:           `{"login":}`,
			setupMock:      func(m *domainmocks.AuthServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Internal error",
			body: `{"login":"user","password":"pass"}`,
			setupMock: func(m *domainmocks.AuthServiceMock) {
				m.EXPECT().Register(mock.Anything, "user", "pass").Return("", errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := domainmocks.NewAuthServiceMock(t)
			logger, _ := zap.NewDevelopment()
			handler := NewAuthHandler(mockService, logger)

			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			handler.Register(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkAuth {
				assert.Contains(t, w.Header().Get("Authorization"), "Bearer token")
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setupMock      func(*domainmocks.AuthServiceMock)
		expectedStatus int
		checkAuth      bool
	}{
		{
			name: "Success",
			body: `{"login":"user","password":"pass"}`,
			setupMock: func(m *domainmocks.AuthServiceMock) {
				m.EXPECT().Login(mock.Anything, "user", "pass").Return("token", nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkAuth:      true,
		},
		{
			name: "Invalid credentials",
			body: `{"login":"user","password":"wrong"}`,
			setupMock: func(m *domainmocks.AuthServiceMock) {
				m.EXPECT().Login(mock.Anything, "user", "wrong").Return("", service.ErrInvalidCredentials).Once()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid input",
			body: `{"login":"","password":"pass"}`,
			setupMock: func(m *domainmocks.AuthServiceMock) {
				m.EXPECT().Login(mock.Anything, "", "pass").Return("", service.ErrInvalidInput).Once()
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			body:           `{"login":}`,
			setupMock:      func(m *domainmocks.AuthServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := domainmocks.NewAuthServiceMock(t)
			logger, _ := zap.NewDevelopment()
			handler := NewAuthHandler(mockService, logger)

			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			handler.Login(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkAuth {
				assert.Contains(t, w.Header().Get("Authorization"), "Bearer token")
			}
		})
	}
}

func TestOrdersHandler_SubmitOrder(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		userID         *int64
		setupMock      func(*domainmocks.OrderServiceMock)
		expectedStatus int
	}{
		{
			name:   "Success - new order",
			body:   "79927398713",
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.OrderServiceMock) {
				m.EXPECT().SubmitOrder(mock.Anything, int64(1), "79927398713").Return(nil).Once()
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:   "Order exists - same user",
			body:   "79927398713",
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.OrderServiceMock) {
				m.EXPECT().SubmitOrder(mock.Anything, int64(1), "79927398713").Return(service.ErrOrderExists).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Order owned by another user",
			body:   "79927398713",
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.OrderServiceMock) {
				m.EXPECT().SubmitOrder(mock.Anything, int64(1), "79927398713").Return(service.ErrOrderOwnedByAnother).Once()
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "Invalid order number",
			body:   "12345",
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.OrderServiceMock) {
				m.EXPECT().SubmitOrder(mock.Anything, int64(1), "12345").Return(service.ErrInvalidOrderNumber).Once()
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "Unauthorized - no user ID",
			body:           "79927398713",
			userID:         nil,
			setupMock:      func(m *domainmocks.OrderServiceMock) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Empty body",
			body:           "",
			userID:         ptrInt64(1),
			setupMock:      func(m *domainmocks.OrderServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := domainmocks.NewOrderServiceMock(t)
			logger, _ := zap.NewDevelopment()
			handler := NewOrdersHandler(mockService, logger)

			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString(tt.body))
			if tt.userID != nil {
				ctx := context.WithValue(req.Context(), UserIDKey, *tt.userID)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			handler.SubmitOrder(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestOrdersHandler_GetOrders(t *testing.T) {
	tests := []struct {
		name           string
		userID         *int64
		setupMock      func(*domainmocks.OrderServiceMock)
		expectedStatus int
		checkBody      bool
	}{
		{
			name:   "Success with orders",
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.OrderServiceMock) {
				orders := []*domain.Order{
					{Number: "111", Status: domain.OrderStatusProcessed},
				}
				m.EXPECT().GetOrders(mock.Anything, int64(1)).Return(orders, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:   "No orders",
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.OrderServiceMock) {
				m.EXPECT().GetOrders(mock.Anything, int64(1)).Return([]*domain.Order{}, nil).Once()
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Unauthorized",
			userID:         nil,
			setupMock:      func(m *domainmocks.OrderServiceMock) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := domainmocks.NewOrderServiceMock(t)
			logger, _ := zap.NewDevelopment()
			handler := NewOrdersHandler(mockService, logger)

			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			if tt.userID != nil {
				ctx := context.WithValue(req.Context(), UserIDKey, *tt.userID)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			handler.GetOrders(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkBody {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestBalanceHandler_GetBalance(t *testing.T) {
	tests := []struct {
		name           string
		userID         *int64
		setupMock      func(*domainmocks.BalanceServiceMock)
		expectedStatus int
		checkBalance   *domain.Balance
	}{
		{
			name:   "Success",
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.BalanceServiceMock) {
				balance := &domain.Balance{Current: 500.0, Withdrawn: 200.0}
				m.EXPECT().GetBalance(mock.Anything, int64(1)).Return(balance, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBalance:   &domain.Balance{Current: 500.0, Withdrawn: 200.0},
		},
		{
			name:           "Unauthorized",
			userID:         nil,
			setupMock:      func(m *domainmocks.BalanceServiceMock) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := domainmocks.NewBalanceServiceMock(t)
			logger, _ := zap.NewDevelopment()
			handler := NewBalanceHandler(mockService, logger)

			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			if tt.userID != nil {
				ctx := context.WithValue(req.Context(), UserIDKey, *tt.userID)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			handler.GetBalance(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkBalance != nil {
				var result domain.Balance
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, tt.checkBalance.Current, result.Current)
				assert.Equal(t, tt.checkBalance.Withdrawn, result.Withdrawn)
			}
		})
	}
}

func TestBalanceHandler_Withdraw(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		userID         *int64
		setupMock      func(*domainmocks.BalanceServiceMock)
		expectedStatus int
	}{
		{
			name:   "Success",
			body:   `{"order":"79927398713","sum":100}`,
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.BalanceServiceMock) {
				m.EXPECT().Withdraw(mock.Anything, int64(1), "79927398713", 100.0).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Insufficient funds",
			body:   `{"order":"79927398713","sum":1000}`,
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.BalanceServiceMock) {
				m.EXPECT().Withdraw(mock.Anything, int64(1), "79927398713", 1000.0).Return(service.ErrInsufficientFunds).Once()
			},
			expectedStatus: http.StatusPaymentRequired,
		},
		{
			name:   "Invalid order number",
			body:   `{"order":"12345","sum":100}`,
			userID: ptrInt64(1),
			setupMock: func(m *domainmocks.BalanceServiceMock) {
				m.EXPECT().Withdraw(mock.Anything, int64(1), "12345", 100.0).Return(service.ErrInvalidOrderNumber).Once()
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "Unauthorized",
			body:           `{"order":"79927398713","sum":100}`,
			userID:         nil,
			setupMock:      func(m *domainmocks.BalanceServiceMock) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid JSON",
			body:           `{"order":}`,
			userID:         ptrInt64(1),
			setupMock:      func(m *domainmocks.BalanceServiceMock) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := domainmocks.NewBalanceServiceMock(t)
			logger, _ := zap.NewDevelopment()
			handler := NewBalanceHandler(mockService, logger)

			tt.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(tt.body))
			if tt.userID != nil {
				ctx := context.WithValue(req.Context(), UserIDKey, *tt.userID)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			handler.Withdraw(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		checkUserID    bool
	}{
		{
			name:           "Valid token",
			authHeader:     "valid", // Will be replaced with actual token
			expectedStatus: http.StatusOK,
			checkUserID:    true,
		},
		{
			name:           "Missing Authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid format - no Bearer",
			authHeader:     "InvalidToken",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			authHeader:     "Bearer invalid.token.string",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	jwtManager := jwt.NewManager("test-secret", time.Hour)
	validToken, _ := jwtManager.Generate(123)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := AuthMiddleware(jwtManager)
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkUserID {
					userID, ok := GetUserID(r.Context())
					assert.True(t, ok)
					assert.Equal(t, int64(123), userID)
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader == "valid" {
				req.Header.Set("Authorization", "Bearer "+validToken)
			} else if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Helper function
func ptrInt64(i int64) *int64 {
	return &i
}
