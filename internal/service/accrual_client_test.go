package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccrualClient_GetOrderAccrual(t *testing.T) {
	ctx := context.Background()

	t.Run("Success - order processed", func(t *testing.T) {
		accrual := 100.0
		response := domain.AccrualResponse{
			Order:   "12345678903",
			Status:  domain.OrderStatusProcessed,
			Accrual: &accrual,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/orders/12345678903", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewAccrualClient(server.URL)
		result, err := client.GetOrderAccrual(ctx, "12345678903")
		require.NoError(t, err)
		assert.Equal(t, response.Order, result.Order)
		assert.Equal(t, response.Status, result.Status)
		assert.Equal(t, *response.Accrual, *result.Accrual)
	})

	t.Run("Success - order processing", func(t *testing.T) {
		response := domain.AccrualResponse{
			Order:  "12345678903",
			Status: domain.OrderStatusProcessing,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewAccrualClient(server.URL)
		result, err := client.GetOrderAccrual(ctx, "12345678903")
		require.NoError(t, err)
		assert.Equal(t, response.Status, result.Status)
		assert.Nil(t, result.Accrual)
	})

	t.Run("Order not registered", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewAccrualClient(server.URL)
		result, err := client.GetOrderAccrual(ctx, "99999999999")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Rate limit exceeded", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		client := NewAccrualClient(server.URL)
		result, err := client.GetOrderAccrual(ctx, "12345678903")
		assert.Error(t, err)
		assert.Nil(t, result)

		var rateLimitErr *RateLimitError
		assert.ErrorAs(t, err, &rateLimitErr)
	})

	t.Run("Unexpected status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewAccrualClient(server.URL)
		result, err := client.GetOrderAccrual(ctx, "12345678903")
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := NewAccrualClient(server.URL)
		result, err := client.GetOrderAccrual(ctx, "12345678903")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
