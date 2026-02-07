package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
)

// AccrualClient определяет методы взаимодействия с системой начислений.
type AccrualClient interface {
	GetOrderAccrual(ctx context.Context, orderNumber string) (*domain.AccrualResponse, error)
}

// HTTPAccrualClient реализует AccrualClient.
type HTTPAccrualClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAccrualClient создает новый AccrualClient
func NewAccrualClient(baseURL string) AccrualClient {
	return &HTTPAccrualClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetOrderAccrual получает информацию о начислении для заказа
func (c *HTTPAccrualClient) GetOrderAccrual(ctx context.Context, orderNumber string) (*domain.AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("accrual client: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("accrual client: failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var accrualResp domain.AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
			return nil, fmt.Errorf("accrual client: failed to decode response: %w", err)
		}
		return &accrualResp, nil

	case http.StatusNoContent:
		// Заказ не зарегистрирован в системе расчета
		return nil, nil

	case http.StatusTooManyRequests:
		// Слишком много запросов, нужно повторить позже
		retryAfter := resp.Header.Get("Retry-After")
		seconds, _ := strconv.Atoi(retryAfter)
		return nil, NewRateLimitError(time.Duration(seconds) * time.Second)

	default:
		return nil, fmt.Errorf("accrual client: unexpected status code: %d", resp.StatusCode)
	}
}
