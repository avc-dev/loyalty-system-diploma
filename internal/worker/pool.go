package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/avc/loyalty-system-diploma/internal/domain"
	"go.uber.org/zap"
)

// PoolConfig содержит конфигурацию worker pool
type PoolConfig struct {
	Workers      int           // Количество воркеров
	QueueSize    int           // Размер очереди заказов
	ScanInterval time.Duration // Интервал сканирования pending заказов
}

// DefaultPoolConfig возвращает конфигурацию по умолчанию
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		Workers:      3,
		QueueSize:    100,
		ScanInterval: 10 * time.Second,
	}
}

// Pool представляет пул воркеров для обработки заказов
type Pool struct {
	config          PoolConfig
	queue           chan string
	retryQueue      chan retryItem
	orderRepo       domain.OrderRepository
	transactionRepo domain.TransactionRepository
	accrualClient   domain.AccrualClient
	logger          *zap.Logger
	wg              sync.WaitGroup
}

// retryItem представляет заказ для повторной обработки
type retryItem struct {
	orderNumber string
	retryAfter  time.Time
}

// NewPool создает новый worker pool
func NewPool(
	config PoolConfig,
	orderRepo domain.OrderRepository,
	transactionRepo domain.TransactionRepository,
	accrualClient domain.AccrualClient,
	logger *zap.Logger,
) *Pool {
	return &Pool{
		config:          config,
		queue:           make(chan string, config.QueueSize),
		retryQueue:      make(chan retryItem, config.QueueSize),
		orderRepo:       orderRepo,
		transactionRepo: transactionRepo,
		accrualClient:   accrualClient,
		logger:          logger,
	}
}

// Start запускает worker pool
func (p *Pool) Start(ctx context.Context) {
	// Запускаем воркеры
	for i := 0; i < p.config.Workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Запускаем сканер pending заказов
	p.wg.Add(1)
	go p.scanner(ctx)

	// Запускаем обработчик retry очереди
	p.wg.Add(1)
	go p.retryProcessor(ctx)
}

// Stop останавливает worker pool
func (p *Pool) Stop() {
	close(p.queue)
	close(p.retryQueue)
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

	ticker := time.NewTicker(p.config.ScanInterval)
	defer ticker.Stop()

	// Сканируем сразу при старте
	p.scanPendingOrders(ctx)

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

// retryProcessor обрабатывает заказы для повторной попытки
func (p *Pool) retryProcessor(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("retry processor stopping")
			return
		case item, ok := <-p.retryQueue:
			if !ok {
				return
			}

			// Ждем до времени retry
			waitDuration := time.Until(item.retryAfter)
			if waitDuration > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(waitDuration):
				}
			}

			// Пытаемся добавить в основную очередь
			select {
			case p.queue <- item.orderNumber:
				p.logger.Debug("order re-queued after rate limit",
					zap.String("order", item.orderNumber))
			case <-ctx.Done():
				return
			default:
				// Очередь полна, пробуем снова через некоторое время
				p.logger.Warn("queue full during retry, will try again",
					zap.String("order", item.orderNumber))
			}
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
		// Обработка rate limiting - неблокирующий retry
		var rateLimitErr *domain.RateLimitError
		if errors.As(err, &rateLimitErr) {
			p.logger.Warn("rate limit exceeded, scheduling retry",
				zap.String("order", orderNumber),
				zap.Duration("retry_after", rateLimitErr.RetryAfter),
			)
			// Добавляем в retry очередь без блокировки
			select {
			case p.retryQueue <- retryItem{
				orderNumber: orderNumber,
				retryAfter:  time.Now().Add(rateLimitErr.RetryAfter),
			}:
			case <-ctx.Done():
			default:
				p.logger.Warn("retry queue full, order will be picked up by scanner",
					zap.String("order", orderNumber))
			}
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

		// Создаем транзакцию начисления (с защитой от дублирования через БД constraint)
		if err := p.transactionRepo.CreateTransaction(ctx, order.UserID, orderNumber, *accrualResp.Accrual, domain.TransactionTypeAccrual); err != nil {
			// Игнорируем ошибку дубликата - заказ уже был обработан
			if errors.Is(err, domain.ErrDuplicateAccrual) {
				p.logger.Debug("accrual already exists for order",
					zap.String("order", orderNumber))
				return
			}
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
