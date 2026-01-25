package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRequestIDMiddleware(t *testing.T) {
	middleware := RequestIDMiddleware()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что request ID добавлен в контекст
		requestID, ok := r.Context().Value(RequestIDKey).(string)
		assert.True(t, ok)
		assert.NotEmpty(t, requestID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestLoggingMiddleware(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	middleware := LoggingMiddleware(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecoveryMiddleware(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	middleware := RecoveryMiddleware(logger)

	t.Run("No panic", func(t *testing.T) {
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("With panic", func(t *testing.T) {
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		// Не должно паниковать
		assert.NotPanics(t, func() {
			handler.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestAuthMiddleware is in handlers_test.go with more comprehensive test cases
// This file focuses on other middleware tests

func TestGetUserID(t *testing.T) {
	t.Run("User ID present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), UserIDKey, int64(123))

		userID, ok := GetUserID(ctx)
		assert.True(t, ok)
		assert.Equal(t, int64(123), userID)
	})

	t.Run("User ID not present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		userID, ok := GetUserID(req.Context())
		assert.False(t, ok)
		assert.Equal(t, int64(0), userID)
	})

	t.Run("Wrong type in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), UserIDKey, "not-an-int")

		userID, ok := GetUserID(ctx)
		assert.False(t, ok)
		assert.Equal(t, int64(0), userID)
	})
}
