-- Откат миграции
DROP INDEX IF EXISTS idx_unique_accrual_per_order;
DROP INDEX IF EXISTS idx_transactions_order_number;
