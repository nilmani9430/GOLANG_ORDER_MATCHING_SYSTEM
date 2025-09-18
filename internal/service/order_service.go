package service

import (
	"context"
	"fmt"
	"time"

	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/domain"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/logger"
)

// OrderService implements the order business logic
type OrderService struct {
	repo   domain.OrderRepository
	logger *logger.Logger
}

// NewOrderService creates a new order service
func NewOrderService(repo domain.OrderRepository, logger *logger.Logger) domain.OrderService {
	return &OrderService{
		repo:   repo,
		logger: logger,
	}
}

// PlaceOrder places a new order and processes matching
func (s *OrderService) PlaceOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	// Validate order
	if err := s.validateOrder(order); err != nil {
		return nil, err
	}

	// Set order defaults
	order.RemainingQty = order.Quantity
	order.Status = domain.OrderStatusOpen
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	// Create order in database
	if err := s.repo.Create(ctx, order); err != nil {
		s.logger.LogDatabaseError(ctx, "create_order", err)
		return nil, domain.ErrDatabaseError
	}

	s.logger.LogOrderPlaced(ctx, order.ID, order.Symbol, string(order.Side), order.Quantity, order.Price)

	// Process order matching
	fmt.Println("calling processorder matching")
	if err := s.processOrderMatching(ctx, order); err != nil {
		s.logger.Logger.ErrorContext(ctx, "Failed to process order matching", "error", err.Error())
		return nil, domain.ErrOrderProcessingFailed
	}
	fmt.Println("returned from processorder matching")

	return order, nil
}

// CancelOrder cancels an existing order
func (s *OrderService) CancelOrder(ctx context.Context, id int64) error {
	// Get order to check if it exists and is open
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if order.Status != domain.OrderStatusOpen {
		return domain.ErrOrderNotOpen
	}

	// Update order status to canceled
	if err := s.repo.UpdateStatus(ctx, id, domain.OrderStatusCanceled, order.RemainingQty); err != nil {
		s.logger.LogDatabaseError(ctx, "cancel_order", err)
		return domain.ErrDatabaseError
	}

	s.logger.LogOrderCanceled(ctx, id, order.Symbol)
	return nil
}

// GetOrderStatus retrieves the status of an order
func (s *OrderService) GetOrderStatus(ctx context.Context, id int64) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// GetOrderBook retrieves the order book for a symbol
func (s *OrderService) GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBook, error) {
	if symbol == "" {
		return nil, domain.ErrInvalidSymbol
	}

	orderBook, err := s.repo.GetOrderBook(ctx, symbol)
	if err != nil {
		s.logger.LogDatabaseError(ctx, "get_order_book", err)
		return nil, domain.ErrDatabaseError
	}

	return orderBook, nil
}

// GetTrades retrieves trades for a symbol
func (s *OrderService) GetTrades(ctx context.Context, symbol string) ([]domain.Trade, error) {
	if symbol == "" {
		return nil, domain.ErrInvalidSymbol
	}

	trades, err := s.repo.GetTrades(ctx, symbol)
	if err != nil {
		s.logger.LogDatabaseError(ctx, "get_trades", err)
		return nil, domain.ErrDatabaseError
	}

	return trades, nil
}

// validateOrder validates order data
func (s *OrderService) validateOrder(order *domain.Order) error {
	if order.Symbol == "" {
		return domain.ErrInvalidSymbol
	}

	if order.Quantity <= 0 {
		return domain.ErrInvalidQuantity
	}

	if order.Type == domain.LimitOrder && order.Price <= 0 {
		return domain.ErrInvalidPrice
	}

	if order.Side != domain.BuySide && order.Side != domain.SellSide {
		return domain.ErrInvalidOrder
	}

	if order.Type != domain.LimitOrder && order.Type != domain.MarketOrder {
		return domain.ErrInvalidOrder
	}

	return nil
}

// processOrderMatching handles order matching logic
func (s *OrderService) processOrderMatching(ctx context.Context, order *domain.Order) error {
	// Get matching orders
	fmt.Println("calling getmatchingorders")
	matchingOrders, err := s.repo.GetMatchingOrders(ctx, order)
	if err != nil {
		return err
	}
	fmt.Println("returned from getmatchingorders")

	//if len(matchingOrders) == 0 {
	//	return nil // No matches found
	//}

	// Process matches concurrently
	return s.processMatches(ctx, order, matchingOrders)
}

// processMatches processes order matches with batch database operations
func (s *OrderService) processMatches(ctx context.Context, order *domain.Order, matchingOrders []domain.Order) error {
	var orderUpdates []domain.OrderUpdate
	var trades []domain.Trade
	var processingErrors []error

	fmt.Println("Coming inside processMatches")

	// Process matches sequentially to avoid race conditions
	for _, match := range matchingOrders {
		if order.RemainingQty <= 0 {
			break
		}
		fmt.Println("match = ", match)

		// Calculate match quantity
		matchQty := min(order.RemainingQty, match.RemainingQty)
		fmt.Println("order.RemainingQty = ", order.RemainingQty)
		fmt.Println("match.RemainingQty = ", match.RemainingQty)

		if matchQty <= 0 {
			continue
		}

		// Update remaining quantities
		order.RemainingQty -= matchQty
		match.RemainingQty -= matchQty

		// Create trade (don't save to DB yet)
		trade := &domain.Trade{
			BuyOrderID:  s.getBuyOrderID(order.Side, order.ID, match.ID),
			SellOrderID: s.getSellOrderID(order.Side, order.ID, match.ID),
			Symbol:      order.Symbol,
			Price:       match.Price,
			Quantity:    matchQty,
			CreatedAt:   time.Now(),
		}

		// Add trade to batch
		trades = append(trades, *trade)

		// Determine final status for matching order
		var matchingOrderStatus domain.OrderStatus
		if match.RemainingQty == 0 {
			matchingOrderStatus = domain.OrderStatusFilled
		} else {
			matchingOrderStatus = domain.OrderStatusPartial
		}

		// Add order update to batch
		orderUpdates = append(orderUpdates, domain.OrderUpdate{
			ID:           match.ID,
			Status:       matchingOrderStatus,
			RemainingQty: match.RemainingQty,
		})

		// Log trade
		s.logger.LogOrderMatched(ctx, trade.BuyOrderID, trade.SellOrderID, trade.Symbol, trade.Price, trade.Quantity)
	}

	// Batch create all trades in a single database call
	if len(trades) > 0 {
		if err := s.repo.CreateTradesBatch(ctx, trades); err != nil {
			processingErrors = append(processingErrors, err)
		}
	}

	// Check for processing errors
	if len(processingErrors) > 0 {
		return fmt.Errorf("failed to process %d matches", len(processingErrors))
	}

	// Update all orders in batch
	if len(orderUpdates) > 0 {
		if err := s.repo.UpdateMultipleOrders(ctx, orderUpdates); err != nil {
			return err
		}
	}

	fmt.Println("My order type is ", order.Type)

	// Update the original order status
	var orderStatus domain.OrderStatus
	if order.RemainingQty == 0 {
		orderStatus = domain.OrderStatusFilled
	} else if order.Type == domain.MarketOrder {
		orderStatus = domain.OrderStatusCanceled
	} else {
		orderStatus = domain.OrderStatusOpen
	}

	if err := s.repo.UpdateStatus(ctx, order.ID, orderStatus, order.RemainingQty); err != nil {
		return err
	}

	order.Status = orderStatus
	return nil
}

// getBuyOrderID determines which order is the buy order
func (s *OrderService) getBuyOrderID(side domain.OrderSide, incomingID, matchID int64) int64 {
	if side == domain.BuySide {
		return incomingID
	}
	return matchID
}

// getSellOrderID determines which order is the sell order
func (s *OrderService) getSellOrderID(side domain.OrderSide, incomingID, matchID int64) int64 {
	if side == domain.SellSide {
		return incomingID
	}
	return matchID
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
