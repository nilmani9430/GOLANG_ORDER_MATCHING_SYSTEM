# Go Order Matching System

A high-performance, concurrent order matching system built with Go, featuring clean architecture, proper error handling, and scalable design patterns.

## 🚀 Features

- **Concurrent Order Processing**: Multi-worker order processing with goroutines and channels
- **Clean Architecture**: Domain-driven design with proper separation of concerns
- **Robust Error Handling**: Custom error types with proper HTTP status codes
- **Structured Logging**: JSON logging with request tracing and performance metrics
- **Graceful Shutdown**: Proper signal handling and resource cleanup
- **Database Optimization**: Connection pooling, prepared statements, and proper indexing
- **Security**: CORS, security headers, and input validation
- **Health Checks**: Built-in health monitoring endpoints
- **Docker Support**: Multi-stage Docker build with security best practices

## 🏗️ Architecture

The application follows clean architecture principles with the following layers:

```
cmd/server/          # Application entry point
internal/
├── config/          # Configuration management
├── database/        # Database connection and setup
├── domain/          # Business entities and interfaces
├── handler/         # HTTP request handlers
├── middleware/      # HTTP middleware (logging, CORS, etc.)
├── repository/      # Data access layer
├── router/          # HTTP routing
└── service/         # Business logic layer
```

## 🔧 API Endpoints

### Orders
- `POST /api/v1/orders` - Place a new order
- `DELETE /api/v1/orders/{id}` - Cancel an order
- `GET /api/v1/orders/{id}` - Get order status

### Market Data
- `GET /api/v1/orderbook?symbol={symbol}` - Get order book
- `GET /api/v1/trades?symbol={symbol}` - Get trade history

### System
- `GET /health` - Health check endpoint

## 📋 Order Types

### Order Sides
- `buy` - Buy order
- `sell` - Sell order

### Order Types
- `limit` - Limit order with specific price
- `market` - Market order (executes at best available price)

### Order Status
- `open` - Order is active and waiting for matches
- `filled` - Order is completely filled
- `canceled` - Order has been canceled
- `partial` - Order is partially filled

## 🚀 Quick Start

### Prerequisites
- Go 1.24.3+
- MySQL 8.0+
- Docker (optional)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/nilmani9430/GOLANG_ORDER_MATCHING_SYSTEM.git
   cd GOLANG_ORDER_MATCHING_SYSTEM
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your database credentials
   ```

4. **Run the application**
   ```bash
   go run cmd/server/main.go
   ```

<!--     -->

## ⚙️ Configuration

The application uses environment variables for configuration:

### Database Configuration
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 3306)
- `DB_USER` - Database username (default: root)
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name (default: order_matching)
- `DB_MAX_OPEN_CONNS` - Max open connections (default: 25)
- `DB_MAX_IDLE_CONNS` - Max idle connections (default: 5)
- `DB_CONN_MAX_LIFETIME` - Connection max lifetime (default: 5m)

### Server Configuration
- `PORT` - Server port (default: 8080)
- `SERVER_READ_TIMEOUT` - Read timeout (default: 30s)
- `SERVER_WRITE_TIMEOUT` - Write timeout (default: 30s)
- `SERVER_IDLE_TIMEOUT` - Idle timeout (default: 120s)

### Application Configuration
- `ENVIRONMENT` - Environment (development/production)
- `LOG_LEVEL` - Log level (debug/info/warn/error)
- `ORDER_QUEUE_SIZE` - Order queue size (default: 1000)
- `WORKER_POOL_SIZE` - Worker pool size (default: 10)
- `GRACEFUL_TIMEOUT` - Graceful shutdown timeout (default: 30s)

## 🔄 Concurrency Features

### Order Processing
- **Worker Pool**: Configurable number of workers processing orders concurrently
- **Order Queue**: Buffered channel for queuing orders
- **Concurrent Matching**: Multiple orders can be matched simultaneously
- **Batch Updates**: Database updates are batched for better performance

### Performance Optimizations
- **Connection Pooling**: Efficient database connection management
- **Prepared Statements**: Reused SQL statements for better performance
- **Indexing**: Optimized database indexes for fast queries
- **Context Support**: Request cancellation and timeout handling

## 📊 Monitoring

### Health Checks
- Built-in health check endpoint at `/health`
- Database connection monitoring
- Order processor statistics

### Logging
- Structured JSON logging
- Request tracing with unique IDs
- Performance metrics (duration, queue size, etc.)
- Error tracking with stack traces

## 🛡️ Security Features

- **CORS Support**: Configurable cross-origin resource sharing
- **Security Headers**: XSS protection, content type options, etc.
- **Input Validation**: Comprehensive request validation
- **Non-root Container**: Docker container runs as non-root user
- **Request Timeouts**: Prevents resource exhaustion


## 📈 Performance

The system is designed for high throughput:
- Concurrent order processing with worker pools
- Efficient database operations with connection pooling
- Optimized order matching algorithms
- Minimal memory allocations

## 🔧 Development

### Code Structure
- **Domain Layer**: Business entities and interfaces
- **Service Layer**: Business logic and orchestration
- **Repository Layer**: Data access abstraction
- **Handler Layer**: HTTP request handling
- **Middleware**: Cross-cutting concerns

### Best Practices Implemented
- Dependency injection
- Interface-based design
- Error handling with custom types
- Context propagation
- Graceful shutdown
- Resource cleanup

## 📝 API Examples

### Place a Limit Buy Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTCUSD",
    "side": "buy",
    "type": "limit",
    "price": 50000.0,
    "quantity": 10
  }'
```

### Place a Market Sell Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTCUSD",
    "side": "sell",
    "type": "market",
    "quantity": 5
  }'
```

### Get Order Book
```bash
curl "http://localhost:8080/api/v1/orderbook?symbol=BTCUSD"
```

### Cancel Order
```bash
curl -X DELETE http://localhost:8080/api/v1/orders/123
```
