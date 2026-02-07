-- Откат схемы базы данных
DROP INDEX IF EXISTS idx_unique_accrual_per_order;
DROP INDEX IF EXISTS idx_transactions_order_number;
DROP INDEX IF EXISTS idx_transactions_processed_at;
DROP INDEX IF EXISTS idx_transactions_type;
DROP INDEX IF EXISTS idx_transactions_user_id;
DROP TABLE IF EXISTS transactions;

DROP INDEX IF EXISTS idx_orders_number;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_user_id;
DROP TABLE IF EXISTS orders;

DROP INDEX IF EXISTS idx_users_login;
DROP TABLE IF EXISTS users;
