package domain

import (
	"context"
	"time"
)

type Order struct {
	ID           int64       `json:"id"`
	Symbol       string      `json:"symbol"`
	Side         OrderSide   `json:"side"`
	Type         OrderType   `json:"type"`
	Price        float64     `json:"price,omitempty"`
	Quantity     int         `json:"quantity"`
	RemainingQty int         `json:"remaining_quantity"`
	Status       OrderStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type Trade struct {
	ID          int64     `json:"id"`
	BuyOrderID  int64     `json:"buy_order_id"`
	SellOrderID int64     `json:"sell_order_id"`
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrderBook struct {
	Symbol string  `json:"symbol"`
	Bids   []Order `json:"bids"`
	Asks   []Order `json:"asks"`
}

type OrderType string

const (
	LimitOrder  OrderType = "limit"
	MarketOrder OrderType = "market"
)

type OrderSide string

const (
	BuySide  OrderSide = "buy"
	SellSide OrderSide = "sell"
)

type OrderStatus string

const (
	OrderStatusOpen     OrderStatus = "open"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusCanceled OrderStatus = "canceled"
	OrderStatusPartial  OrderStatus = "partial"
)

// OrderRepository defines the interface for order data operations
type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id int64) (*Order, error)
	GetOrderBook(ctx context.Context, symbol string) (*OrderBook, error)
	GetTrades(ctx context.Context, symbol string) ([]Trade, error)
	UpdateStatus(ctx context.Context, id int64, status OrderStatus, remainingQty int) error
	GetMatchingOrders(ctx context.Context, order *Order) ([]Order, error)
	UpdateOrderRemainingQty(ctx context.Context, id int64, remainingQty int) error
	CreateTrade(ctx context.Context, trade *Trade) error
	CreateTradesBatch(ctx context.Context, trades []Trade) error
	UpdateMultipleOrders(ctx context.Context, updates []OrderUpdate) error
}

// OrderUpdate represents a batch update for orders
type OrderUpdate struct {
	ID           int64
	Status       OrderStatus
	RemainingQty int
}

// OrderService defines the interface for order business logic
type OrderService interface {
	PlaceOrder(ctx context.Context, order *Order) (*Order, error)
	CancelOrder(ctx context.Context, id int64) error
	GetOrderStatus(ctx context.Context, id int64) (*Order, error)
	GetOrderBook(ctx context.Context, symbol string) (*OrderBook, error)
	GetTrades(ctx context.Context, symbol string) ([]Trade, error)
}

// OrderProcessor handles concurrent order processing
type OrderProcessor interface {
	ProcessOrder(ctx context.Context, order *Order) (*Order, error)
	Start(ctx context.Context) error
	Stop() error
}
