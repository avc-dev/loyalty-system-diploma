package worker

import (
	"context"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	domainmocks "github.com/avc/loyalty-system-diploma/internal/domain/mocks"
	"github.com/avc/loyalty-system-diploma/internal/repository/postgres"
	"github.com/avc/loyalty-system-diploma/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func newTestPool(t *testing.T) (*Pool, *domainmocks.OrderRepositoryMock, *domainmocks.TransactionRepositoryMock, *domainmocks.AccrualClientMock) {
	mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
	mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
	mockAccrualClient := domainmocks.NewAccrualClientMock(t)
	logger, _ := zap.NewDevelopment()

	config := PoolConfig{
		Workers:      1,
		QueueSize:    10,
		ScanInterval: time.Second,
	}
	pool := NewPool(config, mockOrderRepo, mockTxRepo, mockAccrualClient, logger)

	return pool, mockOrderRepo, mockTxRepo, mockAccrualClient
}

func TestPool_ProcessOrder(t *testing.T) {
	tests := []struct {
		name        string
		orderNumber string
		setupMocks  func(*domainmocks.OrderRepositoryMock, *domainmocks.TransactionRepositoryMock, *domainmocks.AccrualClientMock)
	}{
		{
			name:        "Success with accrual",
			orderNumber: "12345678903",
			setupMocks: func(orderRepo *domainmocks.OrderRepositoryMock, txRepo *domainmocks.TransactionRepositoryMock, accrualClient *domainmocks.AccrualClientMock) {
				accrual := 100.0
				accrualResp := &domain.AccrualResponse{
					Order:   "12345678903",
					Status:  domain.OrderStatusProcessed,
					Accrual: &accrual,
				}
				order := &domain.Order{ID: 1, UserID: 1, Number: "12345678903", Status: domain.OrderStatusNew}

				accrualClient.EXPECT().GetOrderAccrual(mock.Anything, "12345678903").Return(accrualResp, nil).Once()
				orderRepo.EXPECT().UpdateOrderStatus(mock.Anything, "12345678903", domain.OrderStatusProcessed, &accrual).Return(nil).Once()
				orderRepo.EXPECT().GetOrderByNumber(mock.Anything, "12345678903").Return(order, nil).Once()
				txRepo.EXPECT().CreateTransaction(mock.Anything, int64(1), "12345678903", accrual, domain.TransactionTypeAccrual).Return(nil).Once()
			},
		},
		{
			name:        "Order not registered in accrual system",
			orderNumber: "12345678903",
			setupMocks: func(orderRepo *domainmocks.OrderRepositoryMock, txRepo *domainmocks.TransactionRepositoryMock, accrualClient *domainmocks.AccrualClientMock) {
				accrualClient.EXPECT().GetOrderAccrual(mock.Anything, "12345678903").Return(nil, nil).Once()
				orderRepo.EXPECT().UpdateOrderStatus(mock.Anything, "12345678903", domain.OrderStatusProcessing, (*float64)(nil)).Return(nil).Once()
			},
		},
		{
			name:        "Order rejected by accrual system",
			orderNumber: "12345678903",
			setupMocks: func(orderRepo *domainmocks.OrderRepositoryMock, txRepo *domainmocks.TransactionRepositoryMock, accrualClient *domainmocks.AccrualClientMock) {
				accrualResp := &domain.AccrualResponse{
					Order:  "12345678903",
					Status: domain.OrderStatusInvalid,
				}
				accrualClient.EXPECT().GetOrderAccrual(mock.Anything, "12345678903").Return(accrualResp, nil).Once()
				orderRepo.EXPECT().UpdateOrderStatus(mock.Anything, "12345678903", domain.OrderStatusInvalid, (*float64)(nil)).Return(nil).Once()
			},
		},
		{
			name:        "Duplicate accrual - already processed",
			orderNumber: "12345678903",
			setupMocks: func(orderRepo *domainmocks.OrderRepositoryMock, txRepo *domainmocks.TransactionRepositoryMock, accrualClient *domainmocks.AccrualClientMock) {
				accrual := 100.0
				accrualResp := &domain.AccrualResponse{
					Order:   "12345678903",
					Status:  domain.OrderStatusProcessed,
					Accrual: &accrual,
				}
				order := &domain.Order{ID: 1, UserID: 1, Number: "12345678903", Status: domain.OrderStatusNew}

				accrualClient.EXPECT().GetOrderAccrual(mock.Anything, "12345678903").Return(accrualResp, nil).Once()
				orderRepo.EXPECT().UpdateOrderStatus(mock.Anything, "12345678903", domain.OrderStatusProcessed, &accrual).Return(nil).Once()
				orderRepo.EXPECT().GetOrderByNumber(mock.Anything, "12345678903").Return(order, nil).Once()
				txRepo.EXPECT().CreateTransaction(mock.Anything, int64(1), "12345678903", accrual, domain.TransactionTypeAccrual).Return(postgres.ErrDuplicateAccrual).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, orderRepo, txRepo, accrualClient := newTestPool(t)
			tt.setupMocks(orderRepo, txRepo, accrualClient)

			ctx := context.Background()
			pool.processOrder(ctx, tt.orderNumber)
		})
	}
}

func TestPool_ProcessOrder_RateLimit(t *testing.T) {
	pool, _, _, accrualClient := newTestPool(t)
	ctx := context.Background()
	orderNumber := "12345678903"

	// Симулируем rate limit
	accrualClient.EXPECT().GetOrderAccrual(mock.Anything, orderNumber).
		Return(nil, service.NewRateLimitError(100*time.Millisecond)).Once()

	pool.processOrder(ctx, orderNumber)

	// Проверяем, что заказ добавлен в retry очередь
	select {
	case item := <-pool.retryQueue:
		assert.Equal(t, orderNumber, item.orderNumber)
		assert.True(t, item.retryAfter.After(time.Now()))
	case <-time.After(100 * time.Millisecond):
		t.Error("expected order in retry queue, got timeout")
	}
}

func TestPool_ScanPendingOrders(t *testing.T) {
	pool, orderRepo, _, _ := newTestPool(t)
	ctx := context.Background()

	pendingOrders := []*domain.Order{
		{ID: 1, Number: "111", Status: domain.OrderStatusNew},
		{ID: 2, Number: "222", Status: domain.OrderStatusProcessing},
	}

	orderRepo.EXPECT().GetPendingOrders(mock.Anything).Return(pendingOrders, nil).Once()

	pool.scanPendingOrders(ctx)

	// Проверяем, что заказы добавлены в очередь
	assert.Equal(t, 2, len(pool.queue), "expected 2 orders in queue")

	received := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case num := <-pool.queue:
			received = append(received, num)
		default:
			t.Fatal("expected item in queue")
		}
	}

	assert.Contains(t, received, "111")
	assert.Contains(t, received, "222")
}
