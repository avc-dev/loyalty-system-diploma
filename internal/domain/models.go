package domain

import "time"

// OrderStatus представляет статус заказа
type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

// TransactionType представляет тип транзакции
type TransactionType string

const (
	TransactionTypeAccrual    TransactionType = "accrual"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
)

// User представляет пользователя системы
type User struct {
	ID           int64     `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"` // Не отправляем хеш в JSON
	CreatedAt    time.Time `json:"created_at"`
}

// Order представляет заказ пользователя
type Order struct {
	ID         int64       `json:"-"`
	UserID     int64       `json:"-"`
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual,omitempty"` // Может быть null
	UploadedAt time.Time   `json:"uploaded_at"`
}

// Transaction представляет операцию на счете
type Transaction struct {
	ID          int64           `json:"-"`
	UserID      int64           `json:"-"`
	OrderNumber string          `json:"order"`
	Amount      float64         `json:"sum"`
	Type        TransactionType `json:"-"`
	ProcessedAt time.Time       `json:"processed_at"`
}

// Balance представляет баланс пользователя
type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// AccrualResponse представляет ответ от системы начислений
type AccrualResponse struct {
	Order   string      `json:"order"`
	Status  OrderStatus `json:"status"`
	Accrual *float64    `json:"accrual,omitempty"`
}
