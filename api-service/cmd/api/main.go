// File: cmd/api/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/database"
	"azlo-goboiler/internal/router"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// --- Logger ---
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.With().Timestamp().Logger()

	// --- Configuration ---
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if cfg.App_Env == "development" {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// --- Production Readiness Checks ---
	if cfg.App_Env == "production" && (cfg.App_Secret == "" || len(cfg.App_Secret) < 32) {
		logger.Fatal().Msg("Refusing to start in production with an insecure APP_SECRET.")
	}

	// --- Database Connection ---
	var dsn string
	if cfg.DatabaseURL != "" {
		dsn = cfg.DatabaseURL
		logger.Info().Msg("Connecting to database using DATABASE_URL")
	} else {
		logger.Info().Msg("Constructing database DSN from individual environment variables")
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPassword, cfg.DbName, cfg.DbSslMode)
	}

	db, err := database.ConnectDB(dsn)
	if err != nil {
		logger.Fatal().Err(err).Msg("Database connection failed")
	}
	defer db.Close()

	// --- Redis Connection ---
	redisAddr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.RedisPassword,
	})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		logger.Fatal().Err(err).Msg("Redis connection failed")
	}
	defer redisClient.Close()
	logger.Info().Msg("Redis client initialized")

	// --- Application Context ---
	app := &config.Application{
		Config: cfg,
		Logger: logger,
		DB:     db,
		Redis:  redisClient,
	}

	// --- Server Setup ---
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router.Setup(app),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- Graceful Shutdown ---
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info().Msg("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatal().Err(err).Msg("Server forced to shutdown")
		}
	}()

	// --- Start Server ---
	logger.Info().Int("port", cfg.Port).Str("env", cfg.App_Env).Msg("Starting server")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}

	logger.Info().Msg("Server stopped")
}
