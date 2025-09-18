package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/domain"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/logger"
	"github.com/gorilla/mux"
)

type OrderHandler struct {
	orderService domain.OrderService
	logger       *logger.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService domain.OrderService, logger *logger.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		logger:       logger,
	}
}

// PlaceOrder handles POST /orders
func (h *OrderHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	var order domain.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		h.logger.Logger.ErrorContext(ctx, "Failed to decode request body", "error", err.Error())
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validateOrderRequest(&order); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Process order
	fmt.Println("calling placeOrder")
	processedOrder, err := h.orderService.PlaceOrder(ctx, &order)
	fmt.Println("returned from placeOrder")
	if err != nil {
		if domainErr, ok := domain.IsDomainError(err); ok {
			h.logger.Logger.ErrorContext(ctx, "Domain error in place order", "error", err.Error())
			h.writeErrorResponse(w, domainErr.Status, domainErr.Message)
			return
		}
		h.logger.Logger.ErrorContext(ctx, "Failed to place order", "error", err.Error())
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to process order")
		return
	}

	duration := time.Since(start)
	h.logger.Logger.InfoContext(ctx, "Order placed successfully",
		"order_id", processedOrder.ID,
		"symbol", processedOrder.Symbol,
		"duration_ms", duration.Milliseconds())

	h.writeJSONResponse(w, http.StatusCreated, processedOrder)
}

// CancelOrder handles DELETE /orders/{id}
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	id, err := h.parseOrderID(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	if err := h.orderService.CancelOrder(ctx, id); err != nil {
		if domainErr, ok := domain.IsDomainError(err); ok {
			h.logger.Logger.ErrorContext(ctx, "Domain error in cancel order", "error", err.Error())
			h.writeErrorResponse(w, domainErr.Status, domainErr.Message)
			return
		}
		h.logger.Logger.ErrorContext(ctx, "Failed to cancel order", "error", err.Error())
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to cancel order")
		return
	}

	duration := time.Since(start)
	h.logger.Logger.InfoContext(ctx, "Order canceled successfully", "order_id", id, "duration_ms", duration.Milliseconds())

	h.writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Order canceled"})
}

// GetOrderStatus handles GET /orders/{id}
func (h *OrderHandler) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	id, err := h.parseOrderID(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	order, err := h.orderService.GetOrderStatus(ctx, id)
	if err != nil {
		if domainErr, ok := domain.IsDomainError(err); ok {
			h.logger.Logger.ErrorContext(ctx, "Domain error in get order status", "error", err.Error())
			h.writeErrorResponse(w, domainErr.Status, domainErr.Message)
			return
		}
		h.logger.Logger.ErrorContext(ctx, "Failed to get order status", "error", err.Error())
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get order status")
		return
	}

	duration := time.Since(start)
	h.logger.Logger.InfoContext(ctx, "Order status retrieved", "order_id", id, "duration_ms", duration.Milliseconds())

	h.writeJSONResponse(w, http.StatusOK, order)
}

// GetOrderBook handles GET /orderbook?symbol={symbol}
func (h *OrderHandler) GetOrderBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Symbol parameter is required")
		return
	}

	orderBook, err := h.orderService.GetOrderBook(ctx, symbol)
	if err != nil {
		if domainErr, ok := domain.IsDomainError(err); ok {
			h.logger.Logger.ErrorContext(ctx, "Domain error in get order book", "error", err.Error())
			h.writeErrorResponse(w, domainErr.Status, domainErr.Message)
			return
		}
		h.logger.Logger.ErrorContext(ctx, "Failed to get order book", "error", err.Error())
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get order book")
		return
	}

	duration := time.Since(start)
	h.logger.Logger.InfoContext(ctx, "Order book retrieved", "symbol", symbol, "duration_ms", duration.Milliseconds())

	h.writeJSONResponse(w, http.StatusOK, orderBook)
}

// GetTrades handles GET /trades?symbol={symbol}
func (h *OrderHandler) GetTrades(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Symbol parameter is required")
		return
	}

	trades, err := h.orderService.GetTrades(ctx, symbol)
	if err != nil {
		if domainErr, ok := domain.IsDomainError(err); ok {
			h.logger.Logger.ErrorContext(ctx, "Domain error in get trades", "error", err.Error())
			h.writeErrorResponse(w, domainErr.Status, domainErr.Message)
			return
		}
		h.logger.Logger.ErrorContext(ctx, "Failed to get trades", "error", err.Error())
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get trades")
		return
	}

	duration := time.Since(start)
	h.logger.Logger.InfoContext(ctx, "Trades retrieved", "symbol", symbol, "count", len(trades), "duration_ms", duration.Milliseconds())

	h.writeJSONResponse(w, http.StatusOK, trades)
}

// validateOrderRequest validates the order request
func (h *OrderHandler) validateOrderRequest(order *domain.Order) error {
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

// parseOrderID extracts and parses order ID from request
func (h *OrderHandler) parseOrderID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		return 0, domain.ErrInvalidOrder
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, domain.ErrInvalidOrder
	}

	return id, nil
}

func (h *OrderHandler) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Logger.Error("Failed to encode JSON response", "error", err.Error())
	}
}
func (h *OrderHandler) writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResponse := map[string]string{"error": message}
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		h.logger.Logger.Error("Failed to encode error response", "error", err.Error())
	}
}
