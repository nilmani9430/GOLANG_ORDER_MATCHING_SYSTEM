package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

type Logger struct {
	*slog.Logger
}

func New(level string) *Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{Logger: logger}
}

// WithRequestID adds request ID to the logger context
func (l *Logger) WithRequestID(ctx context.Context, requestID string) *slog.Logger {
	return l.Logger.With("request_id", requestID)
}

//  adds order information to the logger context
func (l *Logger) WithOrder(orderID int64, symbol string) *slog.Logger {
	return l.Logger.With("order_id", orderID, "symbol", symbol)
}

//  adds trade information to the logger context
func (l *Logger) WithTrade(tradeID int64, symbol string, price float64, quantity int) *slog.Logger {
	return l.Logger.With("trade_id", tradeID, "symbol", symbol, "price", price, "quantity", quantity)
}

//  adds duration to the logger context
func (l *Logger) WithDuration(duration time.Duration) *slog.Logger {
	return l.Logger.With("duration_ms", duration.Milliseconds())
}

//  adds error to the logger context
func (l *Logger) WithError(err error) *slog.Logger {
	return l.Logger.With("error", err.Error())
}

//  logs when an order is placed
func (l *Logger) LogOrderPlaced(ctx context.Context, orderID int64, symbol string, side string, quantity int, price float64) {
	l.WithOrder(orderID, symbol).InfoContext(ctx, "Order placed",
		"side", side,
		"quantity", quantity,
		"price", price,
	)
}

//  logs when orders are matched
func (l *Logger) LogOrderMatched(ctx context.Context, buyOrderID, sellOrderID int64, symbol string, price float64, quantity int) {
	l.Logger.InfoContext(ctx, "Orders matched",
		"buy_order_id", buyOrderID,
		"sell_order_id", sellOrderID,
		"symbol", symbol,
		"price", price,
		"quantity", quantity,
	)
}

// logs when an order is canceled
func (l *Logger) LogOrderCanceled(ctx context.Context, orderID int64, symbol string) {
	l.WithOrder(orderID, symbol).InfoContext(ctx, "Order canceled")
}

//  logs database errors
func (l *Logger) LogDatabaseError(ctx context.Context, operation string, err error) {
	l.Logger.ErrorContext(ctx, "Database error", "operation", operation, "error", err.Error())
}

// LogServerStart logs server startup
func (l *Logger) LogServerStart(port string) {
	l.Logger.Info("Server starting", "port", port)
}

func (l *Logger) LogServerStop() {
	l.Logger.Info("Server stopped")
}
