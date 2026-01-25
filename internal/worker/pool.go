package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"github.com/avc/loyalty-system-diploma/internal/service"
	"go.uber.org/zap"
)

// Pool представляет пул воркеров для обработки заказов
type Pool struct {
	workers         int
	queue           chan string
	orderRepo       domain.OrderRepository
	transactionRepo domain.TransactionRepository
	accrualClient   domain.AccrualClient
	logger          *zap.Logger
	wg              sync.WaitGroup
	scanInterval    time.Duration
}

// NewPool создает новый worker pool
func NewPool(
	workers int,
	queueSize int,
	orderRepo domain.OrderRepository,
	transactionRepo domain.TransactionRepository,
	accrualClient domain.AccrualClient,
	logger *zap.Logger,
) *Pool {
	return &Pool{
		workers:         workers,
		queue:           make(chan string, queueSize),
		orderRepo:       orderRepo,
		transactionRepo: transactionRepo,
		accrualClient:   accrualClient,
		logger:          logger,
		scanInterval:    10 * time.Second,
	}
}

// Start запускает worker pool
func (p *Pool) Start(ctx context.Context) {
	// Запускаем воркеры
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Запускаем сканер pending заказов
	p.wg.Add(1)
	go p.scanner(ctx)
}

// Stop останавливает worker pool
func (p *Pool) Stop() {
	close(p.queue)
	p.wg.Wait()
}

// worker обрабатывает заказы из очереди
func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	p.logger.Info("worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("worker stopping", zap.Int("worker_id", id))
			return
		case orderNumber, ok := <-p.queue:
			if !ok {
				return
			}
			p.processOrder(ctx, orderNumber)
		}
	}
}

// scanner периодически сканирует pending заказы
func (p *Pool) scanner(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("scanner stopping")
			return
		case <-ticker.C:
			p.scanPendingOrders(ctx)
		}
	}
}

// scanPendingOrders сканирует и отправляет pending заказы в очередь
func (p *Pool) scanPendingOrders(ctx context.Context) {
	orders, err := p.orderRepo.GetPendingOrders(ctx)
	if err != nil {
		p.logger.Error("failed to get pending orders", zap.Error(err))
		return
	}

	for _, order := range orders {
		select {
		case p.queue <- order.Number:
			// Успешно добавлено в очередь
		case <-ctx.Done():
			return
		default:
			// Очередь заполнена, пропускаем
			p.logger.Warn("queue is full, skipping order", zap.String("order", order.Number))
		}
	}
}

// processOrder обрабатывает один заказ
func (p *Pool) processOrder(ctx context.Context, orderNumber string) {
	p.logger.Debug("processing order", zap.String("order", orderNumber))

	// Получаем информацию от accrual системы
	accrualResp, err := p.accrualClient.GetOrderAccrual(ctx, orderNumber)
	if err != nil {
		// Обработка rate limiting
		var rateLimitErr *service.RateLimitError
		if errors.As(err, &rateLimitErr) {
			p.logger.Warn("rate limit exceeded",
				zap.String("order", orderNumber),
				zap.Duration("retry_after", rateLimitErr.RetryAfter),
			)
			time.Sleep(rateLimitErr.RetryAfter)
			return
		}

		p.logger.Error("failed to get accrual",
			zap.String("order", orderNumber),
			zap.Error(err),
		)
		return
	}

	// Если заказ не найден в системе начислений, обновляем статус на PROCESSING
	if accrualResp == nil {
		if err := p.orderRepo.UpdateOrderStatus(ctx, orderNumber, domain.OrderStatusProcessing, nil); err != nil {
			p.logger.Error("failed to update order status to PROCESSING",
				zap.String("order", orderNumber),
				zap.Error(err),
			)
		}
		return
	}

	// Обновляем статус заказа
	if err := p.orderRepo.UpdateOrderStatus(ctx, orderNumber, accrualResp.Status, accrualResp.Accrual); err != nil {
		p.logger.Error("failed to update order status",
			zap.String("order", orderNumber),
			zap.Error(err),
		)
		return
	}

	// Если есть начисление и статус PROCESSED, создаем транзакцию
	if accrualResp.Status == domain.OrderStatusProcessed && accrualResp.Accrual != nil && *accrualResp.Accrual > 0 {
		// Получаем информацию о заказе для user_id
		order, err := p.orderRepo.GetOrderByNumber(ctx, orderNumber)
		if err != nil {
			p.logger.Error("failed to get order info",
				zap.String("order", orderNumber),
				zap.Error(err),
			)
			return
		}

		// Создаем транзакцию начисления
		if err := p.transactionRepo.CreateTransaction(ctx, order.UserID, orderNumber, *accrualResp.Accrual, domain.TransactionTypeAccrual); err != nil {
			p.logger.Error("failed to create accrual transaction",
				zap.String("order", orderNumber),
				zap.Float64("accrual", *accrualResp.Accrual),
				zap.Error(err),
			)
			return
		}

		p.logger.Info("order processed successfully",
			zap.String("order", orderNumber),
			zap.Float64("accrual", *accrualResp.Accrual),
		)
	}
}
