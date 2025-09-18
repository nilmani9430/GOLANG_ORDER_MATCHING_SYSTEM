package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/domain"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/logger"
)

// OrderProcessor implements concurrent order processing
type OrderProcessor struct {
	orderService domain.OrderService
	logger       *logger.Logger
	orderQueue   chan *domain.Order
	workerPool   chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
}

// NewOrderProcessor creates a new order processor
func NewOrderProcessor(orderService domain.OrderService, logger *logger.Logger, queueSize, workerPoolSize int) domain.OrderProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &OrderProcessor{
		orderService: orderService,
		logger:       logger,
		orderQueue:   make(chan *domain.Order, queueSize),
		workerPool:   make(chan struct{}, workerPoolSize),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (p *OrderProcessor) ProcessOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("order processor is not running")
	}
	p.mu.RUnlock()

	select {
	case p.orderQueue <- order:
		// Order queued successfully
		p.logger.InfoContext(ctx, "Order queued for processing",
			"order_id", order.ID,
			"symbol", order.Symbol,
			"queue_size", len(p.orderQueue))

		// Wait for processing to complete
		return p.waitForOrderCompletion(ctx, order.ID)
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("order queue is full")
	}
}

// Start starts the order processor
func (p *OrderProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("order processor is already running")
	}

	p.running = true
	p.logger.Info("Starting order processor")

	// Start workers
	for i := 0; i < cap(p.workerPool); i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	// Start queue monitor
	p.wg.Add(1)
	go p.queueMonitor()

	return nil
}

// Stop stops the order processor
func (p *OrderProcessor) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("order processor is not running")
	}

	p.logger.Info("Stopping order processor")
	p.cancel()
	p.wg.Wait()
	p.running = false

	// Close channels
	close(p.orderQueue)
	close(p.workerPool)

	p.logger.Info("Order processor stopped")
	return nil
}

// worker processes orders from the queue
func (p *OrderProcessor) worker(workerID int) {
	defer p.wg.Done()

	p.logger.Info("Worker started", "worker_id", workerID)

	for {
		select {
		case order := <-p.orderQueue:
			if order == nil {
				p.logger.Info("Worker stopping", "worker_id", workerID)
				return
			}

			// Acquire worker slot
			p.workerPool <- struct{}{}

			// Process order
			start := time.Now()
			processedOrder, err := p.orderService.PlaceOrder(p.ctx, order)
			duration := time.Since(start)

			// Release worker slot
			<-p.workerPool

			if err != nil {
				p.logger.Logger.ErrorContext(p.ctx, "Failed to process order",
					"worker_id", workerID,
					"order_id", order.ID,
					"symbol", order.Symbol,
					"error", err.Error(),
					"duration_ms", duration.Milliseconds())
			} else {
				p.logger.Logger.InfoContext(p.ctx, "Order processed successfully",
					"worker_id", workerID,
					"order_id", processedOrder.ID,
					"symbol", processedOrder.Symbol,
					"status", processedOrder.Status,
					"duration_ms", duration.Milliseconds())
			}

		case <-p.ctx.Done():
			p.logger.Info("Worker stopping due to context cancellation", "worker_id", workerID)
			return
		}
	}
}

// queueMonitor monitors the order queue and logs statistics
func (p *OrderProcessor) queueMonitor() {
	defer p.wg.Done()
	fmt.Println("Inside queueMonitor")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			queueSize := len(p.orderQueue)
			availableWorkers := cap(p.workerPool) - len(p.workerPool)

			p.logger.Info("Queue statistics",
				"queue_size", queueSize,
				"queue_capacity", cap(p.orderQueue),
				"available_workers", availableWorkers,
				"total_workers", cap(p.workerPool))

		case <-p.ctx.Done():
			p.logger.Info("Queue monitor stopping")
			return
		}
	}
}

// waitForOrderCompletion waits for an order to be processed
func (p *OrderProcessor) waitForOrderCompletion(ctx context.Context, orderID int64) (*domain.Order, error) {

	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			order, err := p.orderService.GetOrderStatus(ctx, orderID)
			if err == nil && order.Status != domain.OrderStatusOpen {
				return order, nil
			}
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for order completion")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// GetQueueStats returns current queue statistics
func (p *OrderProcessor) GetQueueStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"queue_size":        len(p.orderQueue),
		"queue_capacity":    cap(p.orderQueue),
		"available_workers": cap(p.workerPool) - len(p.workerPool),
		"total_workers":     cap(p.workerPool),
		"running":           p.running,
	}
}
