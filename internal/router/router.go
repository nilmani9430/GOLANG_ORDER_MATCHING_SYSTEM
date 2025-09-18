package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/config"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/handler"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/logger"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/middleware"
	"github.com/gorilla/mux"
)

// Router wraps mux.Router with additional functionality
type Router struct {
	*mux.Router
	config *config.Config
}

// NewRouter creates a new router with all routes and middleware
func NewRouter(
	orderHandler *handler.OrderHandler,
	cfg *config.Config,
	logger *logger.Logger,
) *Router {
	router := mux.NewRouter()

	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.SecurityMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RequestIDMiddleware(logger))
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.TimeoutMiddleware(30 * time.Second))

	router.HandleFunc("/health", healthCheck).Methods("GET")

	api := router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/orders", orderHandler.PlaceOrder).Methods("POST")
	api.HandleFunc("/orders/{id}", orderHandler.CancelOrder).Methods("DELETE")
	api.HandleFunc("/orders/{id}", orderHandler.GetOrderStatus).Methods("GET")
	api.HandleFunc("/orderbook", orderHandler.GetOrderBook).Methods("GET")
	api.HandleFunc("/trades", orderHandler.GetTrades).Methods("GET")

	return &Router{
		Router: router,
		config: cfg,
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`)); err != nil {
		// Log error but don't change response since headers are already sent
		fmt.Printf("Error writing health check response: %v\n", err)
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), r.config.Server.ReadTimeout)
	defer cancel()

	req = req.WithContext(ctx)
	r.Router.ServeHTTP(w, req)
}
