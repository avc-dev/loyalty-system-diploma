package domain

import "context"

// UserRepository определяет методы для работы с пользователями
type UserRepository interface {
	CreateUser(ctx context.Context, login, passwordHash string) (*User, error)
	GetUserByLogin(ctx context.Context, login string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
}

// OrderRepository определяет методы для работы с заказами
type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, number string) (*Order, error)
	GetOrderByNumber(ctx context.Context, number string) (*Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64) ([]*Order, error)
	UpdateOrderStatus(ctx context.Context, number string, status OrderStatus, accrual *float64) error
	GetPendingOrders(ctx context.Context) ([]*Order, error)
}

// TransactionRepository определяет методы для работы с транзакциями
type TransactionRepository interface {
	CreateTransaction(ctx context.Context, userID int64, orderNumber string, amount float64, txType TransactionType) error
	GetBalance(ctx context.Context, userID int64) (*Balance, error)
	GetWithdrawals(ctx context.Context, userID int64) ([]*Transaction, error)
	WithdrawWithLock(ctx context.Context, userID int64, orderNumber string, amount float64) error
}

// AuthService определяет методы аутентификации
type AuthService interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
}

// OrderService определяет методы работы с заказами
type OrderService interface {
	SubmitOrder(ctx context.Context, userID int64, orderNumber string) error
	GetOrders(ctx context.Context, userID int64) ([]*Order, error)
}

// BalanceService определяет методы работы с балансом
type BalanceService interface {
	GetBalance(ctx context.Context, userID int64) (*Balance, error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error
	GetWithdrawals(ctx context.Context, userID int64) ([]*Transaction, error)
}

// AccrualClient определяет методы взаимодействия с системой начислений
type AccrualClient interface {
	GetOrderAccrual(ctx context.Context, orderNumber string) (*AccrualResponse, error)
}
