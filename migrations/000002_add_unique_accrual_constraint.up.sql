-- Добавление уникального индекса для предотвращения дублирования начислений
-- Один заказ может иметь только одну транзакцию типа accrual
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_accrual_per_order 
    ON transactions(order_number) WHERE type = 'accrual';

-- Добавление индекса для быстрого поиска по order_number
CREATE INDEX IF NOT EXISTS idx_transactions_order_number 
    ON transactions(order_number);
