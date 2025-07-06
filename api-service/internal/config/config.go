// File: internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strings" // NEW: Import for trimming whitespace

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
}

type ContextKey string

const UserIDKey = ContextKey("userID")

// NEW: Helper function to read a secret from a file and set it in Viper.
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
	// Defaults are the lowest priority
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("RATE_LIMIT", 100)
	viper.SetDefault("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"})
	viper.SetDefault("DB_HOST", "db")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_SSL_MODE", "require")
	viper.SetDefault("REDIS_HOST", "redis")
	viper.SetDefault("REDIS_PORT", 6379)

	// Environment variables can override defaults
	viper.AutomaticEnv()

	// MODIFIED: Secrets will override environment variables and defaults
	setFromSecret("APP_SECRET", "app_secret")
	setFromSecret("DATABASE_URL", "database_url")
	setFromSecret("DB_HOST", "db_host")
	setFromSecret("DB_PORT", "db_port")
	setFromSecret("DB_USER", "db_user")
	setFromSecret("DB_PASSWORD", "db_password")
	setFromSecret("DB_NAME", "db_name")
	setFromSecret("REDIS_HOST", "redis_host")
	setFromSecret("REDIS_PORT", "redis_port")
	setFromSecret("REDIS_PASSWORD", "redis_password")

	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}

	// MODIFIED: If DATABASE_URL is not set, construct it from its parts.
	// This makes the config robust for different environments.
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
