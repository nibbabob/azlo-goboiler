// File: internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// Application holds all the application-wide dependencies.
type Application struct {
	Config Config
	Logger zerolog.Logger
	DB     *pgxpool.Pool
	Redis  *redis.Client
}

// Config holds all the configuration variables for the application.
type Config struct {
	Port                 int      `mapstructure:"PORT"`
	App_Env              string   `mapstructure:"APP_ENV"`
	App_Secret           string   `mapstructure:"APP_SECRET"`
	CORS_Allowed_Origins []string `mapstructure:"CORS_ALLOWED_ORIGINS"`
	DatabaseURL          string   `mapstructure:"DATABASE_URL"`
	DbHost               string   `mapstructure:"DB_HOST"`
	DbPort               int      `mapstructure:"DB_PORT"`
	DbUser               string   `mapstructure:"DB_USER"`
	DbPassword           string   `mapstructure:"DB_PASSWORD"`
	DbName               string   `mapstructure:"DB_NAME"`
	DbSslMode            string   `mapstructure:"DB_SSL_MODE"`
	RedisHost            string   `mapstructure:"REDIS_HOST"`
	RedisPort            int      `mapstructure:"REDIS_PORT"`
	RedisPassword        string   `mapstructure:"REDIS_PASSWORD"`
	RateLimit            int      `mapstructure:"RATE_LIMIT"`
	LogLevel             string   `mapstructure:"LOG_LEVEL"`
	RequestTimeout       int      `mapstructure:"REQUEST_TIMEOUT_SECONDS"`
	JWTExpirationHours   int      `mapstructure:"JWT_EXPIRATION_HOURS"`
	DefaultUserUsername  string   `mapstructure:"DEFAULT_USER_USERNAME"`
	DefaultUserPassword  string   `mapstructure:"DEFAULT_USER_PASSWORD"`
}

type ContextKey string

const UserIDKey = ContextKey("userID")

// Helper function to read a secret from a file and set it in Viper.
func setFromSecret(key, secretName string) {
	secretPath := fmt.Sprintf("/run/secrets/%s", secretName)
	if _, err := os.Stat(secretPath); err == nil {
		// File exists, so we read it.
		content, err := os.ReadFile(secretPath)
		if err == nil {
			// Trim whitespace (like trailing newlines) from the secret.
			viper.Set(key, strings.TrimSpace(string(content)))
		}
	}
}

// Load reads configuration from secrets, environment variables, or defaults.
func Load() (config Config, err error) {
	// Environment-specific defaults
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Set defaults based on environment
	if env == "production" {
		// Production defaults
		viper.SetDefault("PORT", 8080)
		viper.SetDefault("RATE_LIMIT", 1000)
		viper.SetDefault("LOG_LEVEL", "info")
		viper.SetDefault("REQUEST_TIMEOUT_SECONDS", 30)
		viper.SetDefault("JWT_EXPIRATION_HOURS", 24)
	} else {
		// Development defaults
		viper.SetDefault("PORT", 8080)
		viper.SetDefault("RATE_LIMIT", 100)
		viper.SetDefault("LOG_LEVEL", "debug")
		viper.SetDefault("REQUEST_TIMEOUT_SECONDS", 60)
		viper.SetDefault("JWT_EXPIRATION_HOURS", 168) // 7 days for dev
		viper.SetDefault("DEFAULT_USER_USERNAME", "admin")
		viper.SetDefault("DEFAULT_USER_PASSWORD", "admin123!")
	}

	// Common defaults
	viper.SetDefault("APP_ENV", env)
	viper.SetDefault("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"})
	viper.SetDefault("DB_HOST", "db")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_SSL_MODE", "require")
	viper.SetDefault("REDIS_HOST", "redis")
	viper.SetDefault("REDIS_PORT", 6379)

	// Environment variables can override defaults
	viper.AutomaticEnv()

	// Secrets will override environment variables and defaults
	setFromSecret("APP_SECRET", "app_secret")
	setFromSecret("DATABASE_URL", "database_url")
	setFromSecret("DB_HOST", "db_host")
	setFromSecret("DB_PORT", "db_port")
	setFromSecret("DB_USER", "db_user")
	setFromSecret("DB_PASSWORD", "db_password")
	setFromSecret("DB_NAME", "db_name")
	setFromSecret("DB_SSL_MODE", "db_ssl_mode")
	setFromSecret("REDIS_HOST", "redis_host")
	setFromSecret("REDIS_PORT", "redis_port")
	setFromSecret("REDIS_PASSWORD", "redis_password")

	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}

	// If DATABASE_URL is not set, construct it from its parts
	if config.DatabaseURL == "" {
		config.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			config.DbUser,
			config.DbPassword,
			config.DbHost,
			config.DbPort,
			config.DbName,
			config.DbSslMode,
		)
	}

	return
}

// Validate performs comprehensive configuration validation
func (c *Config) Validate() error {
	var errors []string

	// Validate APP_SECRET
	if c.App_Secret == "" {
		errors = append(errors, "APP_SECRET is required")
	} else if len(c.App_Secret) < 32 {
		errors = append(errors, "APP_SECRET must be at least 32 characters long")
	}

	// Validate PORT
	if c.Port < 1 || c.Port > 65535 {
		errors = append(errors, "PORT must be between 1 and 65535")
	}

	// Validate RATE_LIMIT
	if c.RateLimit < 1 || c.RateLimit > 100000 {
		errors = append(errors, "RATE_LIMIT must be between 1 and 100000")
	}

	// Validate APP_ENV
	if c.App_Env != "development" && c.App_Env != "production" && c.App_Env != "staging" {
		errors = append(errors, "APP_ENV must be one of: development, production, staging")
	}

	// Validate database configuration
	if c.DbUser == "" {
		errors = append(errors, "DB_USER is required")
	}
	if c.DbPassword == "" {
		errors = append(errors, "DB_PASSWORD is required")
	}
	if c.DbName == "" {
		errors = append(errors, "DB_NAME is required")
	}
	if c.DbHost == "" {
		errors = append(errors, "DB_HOST is required")
	}
	if c.DbPort < 1 || c.DbPort > 65535 {
		errors = append(errors, "DB_PORT must be between 1 and 65535")
	}

	// Validate Redis configuration
	if c.RedisHost == "" {
		errors = append(errors, "REDIS_HOST is required")
	}
	if c.RedisPort < 1 || c.RedisPort > 65535 {
		errors = append(errors, "REDIS_PORT must be between 1 and 65535")
	}

	// Validate timeout settings
	if c.RequestTimeout < 1 || c.RequestTimeout > 300 {
		errors = append(errors, "REQUEST_TIMEOUT_SECONDS must be between 1 and 300")
	}

	// Validate JWT settings
	if c.JWTExpirationHours < 1 || c.JWTExpirationHours > 8760 { // max 1 year
		errors = append(errors, "JWT_EXPIRATION_HOURS must be between 1 and 8760")
	}

	// Validate CORS origins
	if len(c.CORS_Allowed_Origins) == 0 {
		errors = append(errors, "At least one CORS_ALLOWED_ORIGIN must be specified")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[c.LogLevel] {
		errors = append(errors, "LOG_LEVEL must be one of: debug, info, warn, error, fatal")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// IsDevelopment returns true if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App_Env == "development"
}

// IsProduction returns true if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.App_Env == "production"
}

// GetJWTExpiration returns the JWT expiration duration
func (c *Config) GetJWTExpiration() time.Duration {
	return time.Duration(c.JWTExpirationHours) * time.Hour
}

// GetRequestTimeout returns the request timeout duration
func (c *Config) GetRequestTimeout() time.Duration {
	return time.Duration(c.RequestTimeout) * time.Second
}
