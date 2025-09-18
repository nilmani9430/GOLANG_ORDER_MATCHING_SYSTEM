package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/config"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/database"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/handler"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/logger"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/repository"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/router"
	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using environment variables", "error", err)
	}
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	appLogger := logger.New(cfg.App.LogLevel)
	appLogger.LogServerStart(cfg.Server.Port)

	db, err := database.NewDatabase(&cfg.Database, appLogger.Logger)
	if err != nil {
		appLogger.WithError(err).Error("Failed to initialize database")
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			appLogger.WithError(err).Error("Failed to close database")
		}
	}()

	orderRepo := repository.NewMySQLOrderRepository(db.DB)

	orderService := service.NewOrderService(orderRepo, appLogger)

	orderProcessor := service.NewOrderProcessor(orderService, appLogger, cfg.App.OrderQueueSize, cfg.App.WorkerPoolSize)

	ctx := context.Background()
	if err := orderProcessor.Start(ctx); err != nil {
		appLogger.WithError(err).Error("Failed to start order processor")
		os.Exit(1)
	}
	defer func() {
		if err := orderProcessor.Stop(); err != nil {
			appLogger.WithError(err).Error("Failed to stop order processor")
		}
	}()

	orderHandler := handler.NewOrderHandler(orderService, appLogger)

	appRouter := router.NewRouter(orderHandler, cfg, appLogger)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      appRouter,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		appLogger.Info("Starting HTTP server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Error("Server failed to start")
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.GracefulTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.WithError(err).Error("Server forced to shutdown")
		os.Exit(1)
	}

	appLogger.LogServerStop()
}
