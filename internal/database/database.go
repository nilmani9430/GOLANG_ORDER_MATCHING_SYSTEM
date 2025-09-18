package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/config"
	"github.com/go-sql-driver/mysql"
)

// Database wraps sql.DB with additional functionality
type Database struct {
	*sql.DB
	config *config.DatabaseConfig
	logger *slog.Logger
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.DatabaseConfig, logger *slog.Logger) (*Database, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		DB:     db,
		config: cfg,
		logger: logger,
	}

	// Create tables
	if err := database.createTables(ctx); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	logger.Info("Database connection established successfully")
	return database, nil
}

func (db *Database) Close() error {
	db.logger.Info("Closing database connection")
	return db.DB.Close()
}

func (db *Database) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

func (db *Database) createTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS orders (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL,
			side ENUM('buy', 'sell') NOT NULL,
			type ENUM('limit', 'market') NOT NULL,
			price DECIMAL(18,8),
			quantity INT NOT NULL,
			remaining_quantity INT NOT NULL,
			status ENUM('open', 'filled', 'canceled', 'partial') NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_symbol_side_status (symbol, side, status),
			INDEX idx_symbol_price_time (symbol, price, created_at),
			INDEX idx_status_created (status, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS trades (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL,
			buy_order_id BIGINT NOT NULL,
			sell_order_id BIGINT NOT NULL,
			price DECIMAL(18,8) NOT NULL,
			quantity INT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_symbol_created (symbol, created_at),
			INDEX idx_buy_order (buy_order_id),
			INDEX idx_sell_order (sell_order_id),
			FOREIGN KEY (buy_order_id) REFERENCES orders(id) ON DELETE CASCADE,
			FOREIGN KEY (sell_order_id) REFERENCES orders(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			// Check if it's a duplicate key error (index already exists)
			if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1061 {
				db.logger.Warn("Index already exists, skipping", "error", err)
				continue
			}
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	db.logger.Info("Database tables created successfully")
	return nil
}
func (db *Database) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// GetStats returns database connection statistics
func (db *Database) GetStats() sql.DBStats {
	return db.Stats()
}
