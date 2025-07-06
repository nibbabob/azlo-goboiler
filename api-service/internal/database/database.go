// File: internal/database/database.go
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// DefaultDatabaseConfig returns production-ready database configuration
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		MaxConns:          30,
		MinConns:          5,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   time.Minute * 30,
		HealthCheckPeriod: time.Minute * 5,
	}
}

// ConnectDB creates an optimized database connection pool
func ConnectDB(dsn string) (*pgxpool.Pool, error) {
	return ConnectDBWithConfig(dsn, DefaultDatabaseConfig())
}

// ConnectDBWithConfig creates a database connection pool with custom configuration
func ConnectDBWithConfig(dsn string, dbConfig *DatabaseConfig) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse the DSN
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %v", err)
	}

	// Apply production-ready pool settings
	config.MaxConns = dbConfig.MaxConns
	config.MinConns = dbConfig.MinConns
	config.MaxConnLifetime = dbConfig.MaxConnLifetime
	config.MaxConnIdleTime = dbConfig.MaxConnIdleTime
	config.HealthCheckPeriod = dbConfig.HealthCheckPeriod

	// Set up connection hooks for monitoring and initialization
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Set up any per-connection configuration
		_, err := conn.Exec(ctx, "SET application_name = 'go-api-boilerplate'")
		if err != nil {
			log.Warn().Err(err).Msg("Failed to set application name")
		}

		// Set timezone
		_, err = conn.Exec(ctx, "SET timezone = 'UTC'")
		if err != nil {
			log.Warn().Err(err).Msg("Failed to set timezone")
		}

		log.Debug().Msg("Database connection established")
		return nil
	}

	// Create the connection pool
	dbpool, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	// Test the connection
	if err = dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		return nil, fmt.Errorf("database ping failed: %v", err)
	}

	log.Info().
		Int32("max_conns", config.MaxConns).
		Int32("min_conns", config.MinConns).
		Dur("max_conn_lifetime", config.MaxConnLifetime).
		Dur("max_conn_idle_time", config.MaxConnIdleTime).
		Msg("Database connection pool established")

	return dbpool, nil
}

// InitializeSchema creates the necessary database tables
func InitializeSchema(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		last_login TIMESTAMP WITH TIME ZONE
	);`

	_, err := db.Exec(ctx, createUsersTable)
	if err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);",
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);",
		"CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);",
		"CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);",
	}

	for _, indexSQL := range indexes {
		_, err := db.Exec(ctx, indexSQL)
		if err != nil {
			log.Warn().Err(err).Str("sql", indexSQL).Msg("Failed to create index")
		}
	}

	// Create a trigger to automatically update the updated_at column
	updateTrigger := `
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	DROP TRIGGER IF EXISTS update_users_updated_at ON users;
	CREATE TRIGGER update_users_updated_at
		BEFORE UPDATE ON users
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`

	_, err = db.Exec(ctx, updateTrigger)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create update trigger")
	}

	log.Info().Msg("Database schema initialized successfully")
	return nil
}

// StartConnectionMonitoring starts a goroutine that logs connection pool statistics
func StartConnectionMonitoring(db *pgxpool.Pool) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := db.Stat()

			log.Info().
				Int32("total_conns", stats.TotalConns()).
				Int32("acquired_conns", stats.AcquiredConns()).
				Int32("idle_conns", stats.IdleConns()).
				Int32("max_conns", stats.MaxConns()).
				Dur("acquire_duration", stats.AcquireDuration()).
				Int64("acquire_count", stats.AcquireCount()).
				Int64("canceled_acquire_count", stats.CanceledAcquireCount()).
				Msg("Database connection pool statistics")
		}
	}()
}

// HealthCheck performs a comprehensive database health check
func HealthCheck(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test basic connectivity
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}

	// Test query execution
	var version string
	err := db.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("query test failed: %v", err)
	}

	// Test transaction
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transaction begin failed: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "SELECT 1")
	if err != nil {
		return fmt.Errorf("transaction query failed: %v", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("transaction commit failed: %v", err)
	}

	return nil
}

// GetConnectionStats returns current connection pool statistics
func GetConnectionStats(db *pgxpool.Pool) map[string]interface{} {
	stats := db.Stat()

	return map[string]interface{}{
		"total_connections":          stats.TotalConns(),
		"acquired_connections":       stats.AcquiredConns(),
		"idle_connections":           stats.IdleConns(),
		"max_connections":            stats.MaxConns(),
		"acquire_count":              stats.AcquireCount(),
		"acquire_duration_ms":        stats.AcquireDuration().Milliseconds(),
		"canceled_acquire_count":     stats.CanceledAcquireCount(),
		"constructed_connections":    stats.ConstructingConns(),
		"empty_acquire_count":        stats.EmptyAcquireCount(),
		"max_lifetime_destroy_count": stats.MaxLifetimeDestroyCount(),
		"max_idle_destroy_count":     stats.MaxIdleDestroyCount(),
	}
}
