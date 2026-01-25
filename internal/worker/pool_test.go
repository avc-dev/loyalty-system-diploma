package worker

import (
	"context"
	"testing"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	domainmocks "github.com/avc/loyalty-system-diploma/internal/domain/mocks"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestPool_ProcessOrder(t *testing.T) {
	mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
	mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
	mockAccrualClient := domainmocks.NewAccrualClientMock(t)
	logger, _ := zap.NewDevelopment()

	pool := NewPool(1, 10, mockOrderRepo, mockTxRepo, mockAccrualClient, logger)

	ctx := context.Background()
	orderNumber := "12345678903"
	accrual := 100.0

	// Mock responses
	accrualResp := &domain.AccrualResponse{
		Order:   orderNumber,
		Status:  domain.OrderStatusProcessed,
		Accrual: &accrual,
	}
	order := &domain.Order{ID: 1, UserID: 1, Number: orderNumber, Status: domain.OrderStatusNew}

	mockAccrualClient.EXPECT().GetOrderAccrual(mock.Anything, orderNumber).Return(accrualResp, nil).Once()
	mockOrderRepo.EXPECT().UpdateOrderStatus(mock.Anything, orderNumber, domain.OrderStatusProcessed, &accrual).Return(nil).Once()
	mockOrderRepo.EXPECT().GetOrderByNumber(mock.Anything, orderNumber).Return(order, nil).Once()
	mockTxRepo.EXPECT().CreateTransaction(mock.Anything, int64(1), orderNumber, accrual, domain.TransactionTypeAccrual).Return(nil).Once()

	pool.processOrder(ctx, orderNumber)

	// Give some time for async operations
	time.Sleep(100 * time.Millisecond)
}

func TestPool_ProcessOrder_NoAccrual(t *testing.T) {
	mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
	mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
	mockAccrualClient := domainmocks.NewAccrualClientMock(t)
	logger, _ := zap.NewDevelopment()

	pool := NewPool(1, 10, mockOrderRepo, mockTxRepo, mockAccrualClient, logger)

	ctx := context.Background()
	orderNumber := "12345678903"

	// Заказ не зарегистрирован в системе начислений
	mockAccrualClient.EXPECT().GetOrderAccrual(mock.Anything, orderNumber).Return(nil, nil).Once()
	mockOrderRepo.EXPECT().UpdateOrderStatus(mock.Anything, orderNumber, domain.OrderStatusProcessing, (*float64)(nil)).Return(nil).Once()

	pool.processOrder(ctx, orderNumber)

	time.Sleep(100 * time.Millisecond)
}

func TestPool_ProcessOrder_InvalidStatus(t *testing.T) {
	mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
	mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
	mockAccrualClient := domainmocks.NewAccrualClientMock(t)
	logger, _ := zap.NewDevelopment()

	pool := NewPool(1, 10, mockOrderRepo, mockTxRepo, mockAccrualClient, logger)

	ctx := context.Background()
	orderNumber := "12345678903"

	// Заказ отклонен системой начислений
	accrualResp := &domain.AccrualResponse{
		Order:  orderNumber,
		Status: domain.OrderStatusInvalid,
	}

	mockAccrualClient.EXPECT().GetOrderAccrual(mock.Anything, orderNumber).Return(accrualResp, nil).Once()
	mockOrderRepo.EXPECT().UpdateOrderStatus(mock.Anything, orderNumber, domain.OrderStatusInvalid, (*float64)(nil)).Return(nil).Once()

	pool.processOrder(ctx, orderNumber)

	time.Sleep(100 * time.Millisecond)
}

func TestPool_ScanPendingOrders(t *testing.T) {
	mockOrderRepo := domainmocks.NewOrderRepositoryMock(t)
	mockTxRepo := domainmocks.NewTransactionRepositoryMock(t)
	mockAccrualClient := domainmocks.NewAccrualClientMock(t)
	logger, _ := zap.NewDevelopment()

	pool := NewPool(1, 10, mockOrderRepo, mockTxRepo, mockAccrualClient, logger)

	ctx := context.Background()

	pendingOrders := []*domain.Order{
		{ID: 1, Number: "111", Status: domain.OrderStatusNew},
		{ID: 2, Number: "222", Status: domain.OrderStatusProcessing},
	}

	mockOrderRepo.EXPECT().GetPendingOrders(mock.Anything).Return(pendingOrders, nil).Once()

	pool.scanPendingOrders(ctx)

	// Проверяем, что заказы добавлены в очередь
	select {
	case num := <-pool.queue:
		if num != "111" && num != "222" {
			t.Errorf("unexpected order number in queue: %s", num)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected order in queue, got timeout")
	}
}
