// File: cmd/api/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/database"
	"azlo-goboiler/internal/router"
	"azlo-goboiler/internal/telemetry"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Version information (set during build)
	version   = "1.0.1"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// @title           Azlo Go Boilerplate API
// @version         1.0.1
// @description     Production-ready SaaS starter kit API.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost
// @BasePath  /
// @schemes   https

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
func main() {
	// Initialize logger first
	logger := initLogger()

	// Log startup information
	logger.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Str("git_commit", gitCommit).
		Str("go_version", runtime.Version()).
		Str("os", runtime.GOOS).
		Str("arch", runtime.GOARCH).
		Msg("Starting API server")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("Configuration validation failed")
	}

	// Production readiness checks
	if cfg.App_Env == "production" {
		if cfg.App_Secret == "" || len(cfg.App_Secret) < 32 {
			logger.Fatal().Msg("Refusing to start in production with an insecure APP_SECRET")
		}
	}

	// Set log level based on environment
	if cfg.App_Env == "development" {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Database Connection with retry logic
	var db *pgxpool.Pool
	for attempts := 0; attempts < 5; attempts++ {
		var dsn string
		if cfg.DatabaseURL != "" {
			dsn = cfg.DatabaseURL
			logger.Info().Msg("Connecting to database using DATABASE_URL")
		} else {
			logger.Info().Msg("Constructing database DSN from individual environment variables")
			dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
				cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPassword, cfg.DbName, cfg.DbSslMode)
		}

		dbConfig := &database.DatabaseConfig{
			MaxConns:          getEnvInt("DB_MAX_CONNS", 30),
			MinConns:          getEnvInt("DB_MIN_CONNS", 5),
			MaxConnLifetime:   time.Duration(getEnvInt("DB_MAX_CONN_LIFETIME_MINUTES", 60)) * time.Minute,
			MaxConnIdleTime:   time.Duration(getEnvInt("DB_MAX_CONN_IDLE_MINUTES", 30)) * time.Minute,
			HealthCheckPeriod: time.Duration(getEnvInt("DB_HEALTH_CHECK_MINUTES", 5)) * time.Minute,
		}

		db, err = database.ConnectDBWithConfig(dsn, dbConfig)
		if err != nil {
			logger.Warn().
				Err(err).
				Int("attempt", attempts+1).
				Msg("Database connection failed, retrying...")

			if attempts < 4 {
				time.Sleep(time.Duration(attempts+1) * 2 * time.Second)
				continue
			}
			logger.Fatal().Err(err).Msg("Database connection failed after all retries")
		}
		break
	}
	defer db.Close()

	// Initialize OpenTelemetry Tracer
	tp, err := telemetry.InitTracerProvider()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize TracerProvider")
	}

	// Application Context
	app := &config.Application{
		Config:         cfg,
		Logger:         logger,
		DB:             db,
		TracerProvider: tp,
	}

	// Initialize database schema
	if err := database.InitializeSchema(db); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database schema")
	}

	// Seed default user in development
	database.SeedDefaultUser(app)

	// Start database connection monitoring
	database.StartConnectionMonitoring(db)

	// Redis Connection with retry logic
	var redisClient *redis.Client
	for attempts := 0; attempts < 5; attempts++ {
		redisAddr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)
		redisClient = redis.NewClient(&redis.Options{
			Addr:         redisAddr,
			Password:     cfg.RedisPassword,
			DB:           0,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
			MinIdleConns: 5,
		})
		redisClient.AddHook(redisotel.NewTracingHook())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := redisClient.Ping(ctx).Result()
		cancel()

		if err != nil {
			logger.Warn().
				Err(err).
				Int("attempt", attempts+1).
				Msg("Redis connection failed, retrying...")

			if attempts < 8 {
				time.Sleep(time.Duration(attempts+1) * 2 * time.Second)
				continue
			}
			logger.Fatal().Err(err).Msg("Redis connection failed after all retries")
		}
		break
	}
	defer redisClient.Close()
	logger.Info().Msg("Redis client initialized")

	// Update Application Context with Redis client
	app.Redis = redisClient

	// Server Setup with production-ready timeouts
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router.Setup(app),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		// Add additional security headers
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info().
			Int("port", cfg.Port).
			Str("env", cfg.App_Env).
			Msg("Starting HTTP server")

		serverErrors <- srv.ListenAndServe()
	}()

	// Enhanced Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("Server failed to start")
		}
	case sig := <-quit:
		logger.Info().
			Str("signal", sig.String()).
			Msg("Received shutdown signal, starting graceful shutdown...")

		gracefulShutdown(srv, app, logger)
	}

	logger.Info().Msg("Server stopped gracefully")
}

// initLogger initializes the global logger
func initLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.With().
		Timestamp().
		Caller().
		Logger()

	return logger
}

// gracefulShutdown handles the graceful shutdown process
func gracefulShutdown(srv *http.Server, app *config.Application, logger zerolog.Logger) {
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Disable keep-alives to force existing connections to close
	srv.SetKeepAlivesEnabled(false)

	// Shutdown OpenTelemetry TracerProvider
	logger.Info().Msg("Shutting down OpenTelemetry TracerProvider...")
	if err := app.TracerProvider.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("TracerProvider shutdown error")
	} else {
		logger.Info().Msg("TracerProvider shutdown complete")
	}

	// Close database connections
	logger.Info().Msg("Closing database connections...")
	app.DB.Close()
	logger.Info().Msg("Database connections closed")

	// Close Redis connections
	logger.Info().Msg("Closing Redis connections...")
	if err := app.Redis.Close(); err != nil {
		logger.Error().Err(err).Msg("Redis shutdown error")
	} else {
		logger.Info().Msg("Redis connections closed")
	}

	logger.Info().Msg("Shutting down HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	} else {
		logger.Info().Msg("HTTP server shutdown complete")
	}

	logger.Info().Msg("Graceful shutdown completed")
}

// getEnvInt gets an environment variable as int with default fallback
func getEnvInt(key string, defaultValue int) int32 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return int32(intValue)
		}
	}
	return int32(defaultValue)
}
