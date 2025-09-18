package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/domain"
)

// MySQLOrderRepository implements OrderRepository interface for MySQL
type MySQLOrderRepository struct {
	db *sql.DB
}

// NewMySQLOrderRepository creates a new MySQL order repository
func NewMySQLOrderRepository(db *sql.DB) domain.OrderRepository {
	return &MySQLOrderRepository{db: db}
}

// Create inserts a new order into the database
func (r *MySQLOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	query := `
		INSERT INTO orders (symbol, side, type, price, quantity, remaining_quantity, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query,
		order.Symbol, order.Side, order.Type, order.Price,
		order.Quantity, order.RemainingQty, order.Status,
		order.CreatedAt, order.UpdatedAt)

	if err != nil {
		return domain.ErrDatabaseError
	}

	id, err := result.LastInsertId()
	if err != nil {
		return domain.ErrDatabaseError
	}

	order.ID = id
	return nil
}

// GetByID retrieves an order by its ID
func (r *MySQLOrderRepository) GetByID(ctx context.Context, id int64) (*domain.Order, error) {
	query := `
		SELECT id, symbol, side, type, price, quantity, remaining_quantity, status, created_at, updated_at
		FROM orders WHERE id = ?`

	order := &domain.Order{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID, &order.Symbol, &order.Side, &order.Type, &order.Price,
		&order.Quantity, &order.RemainingQty, &order.Status,
		&order.CreatedAt, &order.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, domain.ErrDatabaseError
	}

	return order, nil
}

// GetOrderBook retrieves the order book for a given symbol
func (r *MySQLOrderRepository) GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	orderBook := &domain.OrderBook{
		Symbol: symbol,
		Bids:   []domain.Order{},
		Asks:   []domain.Order{},
	}

	// Get buy orders (bids) - ordered by price DESC, then by time ASC
	bidsQuery := `
		SELECT id, symbol, side, type, price, quantity, remaining_quantity, status, created_at, updated_at
		FROM orders 
		WHERE symbol = ? AND side = 'buy' AND status = 'open'
		ORDER BY price DESC, created_at ASC`

	bids, err := r.queryOrders(ctx, bidsQuery, symbol)
	if err != nil {
		return nil, err
	}
	orderBook.Bids = bids

	// Get sell orders (asks) - ordered by price ASC, then by time ASC
	asksQuery := `
		SELECT id, symbol, side, type, price, quantity, remaining_quantity, status, created_at, updated_at
		FROM orders 
		WHERE symbol = ? AND side = 'sell' AND status = 'open'
		ORDER BY price ASC, created_at ASC`

	asks, err := r.queryOrders(ctx, asksQuery, symbol)
	if err != nil {
		return nil, err
	}
	orderBook.Asks = asks

	return orderBook, nil
}

// GetTrades retrieves all trades for a given symbol
func (r *MySQLOrderRepository) GetTrades(ctx context.Context, symbol string) ([]domain.Trade, error) {
	query := `
		SELECT id, buy_order_id, sell_order_id, symbol, price, quantity, created_at
		FROM trades 
		WHERE symbol = ?
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, symbol)
	if err != nil {
		return nil, domain.ErrDatabaseError
	}
	defer rows.Close()

	var trades []domain.Trade
	for rows.Next() {
		var trade domain.Trade
		err := rows.Scan(
			&trade.ID, &trade.BuyOrderID, &trade.SellOrderID,
			&trade.Symbol, &trade.Price, &trade.Quantity, &trade.CreatedAt)
		if err != nil {
			return nil, domain.ErrDatabaseError
		}
		trades = append(trades, trade)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.ErrDatabaseError
	}

	return trades, nil
}

// UpdateStatus updates the status and remaining quantity of an order
func (r *MySQLOrderRepository) UpdateStatus(ctx context.Context, id int64, status domain.OrderStatus, remainingQty int) error {
	query := `
		UPDATE orders 
		SET status = ?, remaining_quantity = ?, updated_at = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, remainingQty, time.Now(), id)
	if err != nil {
		return domain.ErrDatabaseError
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError
	}

	if rowsAffected == 0 {
		return domain.ErrOrderNotFound
	}

	return nil
}

// CreateTradesBatch creates multiple trades in a single database transaction
func (r *MySQLOrderRepository) CreateTradesBatch(ctx context.Context, trades []domain.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ErrDatabaseError
	}
	defer tx.Rollback()

	// Prepare batch insert query
	query := `
		INSERT INTO trades (buy_order_id, sell_order_id, symbol, price, quantity, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return domain.ErrDatabaseError
	}
	defer stmt.Close()

	// Insert all trades
	for _, trade := range trades {
		_, err := stmt.ExecContext(ctx,
			trade.BuyOrderID, trade.SellOrderID, trade.Symbol,
			trade.Price, trade.Quantity, trade.CreatedAt)
		if err != nil {
			return domain.ErrDatabaseError
		}
	}

	// Commit transaction
	return tx.Commit()
}

// GetMatchingOrders retrieves orders that can be matched with the given order
func (r *MySQLOrderRepository) GetMatchingOrders(ctx context.Context, order *domain.Order) ([]domain.Order, error) {
	var query string
	var args []interface{}

	if order.Side == domain.BuySide {
		// For limit buy orders, get sell orders with price <= order price
		query = `
				SELECT id, symbol, side, type, price, quantity, remaining_quantity, status, created_at, updated_at
				FROM orders
				WHERE symbol = ? AND side = 'sell' AND status IN ('open', 'partial') AND price <= ?
				ORDER BY price ASC, created_at ASC`
		args = []interface{}{order.Symbol, order.Price}
	} else {
		// For limit sell orders, get buy orders with price >= order price
		query = `
				SELECT id, symbol, side, type, price, quantity, remaining_quantity, status, created_at, updated_at
				FROM orders
				WHERE symbol = ? AND side = 'buy' AND status IN ('open', 'partial') AND price >= ?
				ORDER BY price DESC, created_at ASC`
		args = []interface{}{order.Symbol, order.Price}
	}

	return r.queryOrders(ctx, query, args...)
}

// UpdateOrderRemainingQty updates the remaining quantity of an order
func (r *MySQLOrderRepository) UpdateOrderRemainingQty(ctx context.Context, id int64, remainingQty int) error {
	query := `
		UPDATE orders 
		SET remaining_quantity = ?, updated_at = ?
		WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, remainingQty, time.Now(), id)
	if err != nil {
		return domain.ErrDatabaseError
	}

	return nil
}

// CreateTrade inserts a new trade into the database
func (r *MySQLOrderRepository) CreateTrade(ctx context.Context, trade *domain.Trade) error {
	query := `
		INSERT INTO trades (buy_order_id, sell_order_id, symbol, price, quantity, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query,
		trade.BuyOrderID, trade.SellOrderID, trade.Symbol,
		trade.Price, trade.Quantity, trade.CreatedAt)

	if err != nil {
		return domain.ErrDatabaseError
	}

	id, err := result.LastInsertId()
	if err != nil {
		return domain.ErrDatabaseError
	}

	trade.ID = id
	return nil
}

// UpdateMultipleOrders updates multiple orders in a single transaction
func (r *MySQLOrderRepository) UpdateMultipleOrders(ctx context.Context, updates []domain.OrderUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ErrDatabaseError
	}
	defer tx.Rollback()

	now := time.Now()
	for _, update := range updates {
		query := `
			UPDATE orders 
			SET status = ?, remaining_quantity = ?, updated_at = ?
			WHERE id = ?`

		_, err := tx.ExecContext(ctx, query, update.Status, update.RemainingQty, now, update.ID)
		if err != nil {
			return domain.ErrDatabaseError
		}
	}

	return tx.Commit()
}

// queryOrders is a helper method to execute order queries
func (r *MySQLOrderRepository) queryOrders(ctx context.Context, query string, args ...interface{}) ([]domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, domain.ErrDatabaseError
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		err := rows.Scan(
			&order.ID, &order.Symbol, &order.Side, &order.Type, &order.Price,
			&order.Quantity, &order.RemainingQty, &order.Status,
			&order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, domain.ErrDatabaseError
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.ErrDatabaseError
	}

	return orders, nil
}
