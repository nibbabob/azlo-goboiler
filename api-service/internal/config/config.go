package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Application holds all the application-wide dependencies.
type Application struct {
	Config         Config
	Logger         zerolog.Logger
	DB             *pgxpool.Pool
	Redis          *redis.Client
	TracerProvider *trace.TracerProvider
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
	// Notification Configuration
	SMTPHost     string `mapstructure:"SMTP_HOST"`
	SMTPPort     int    `mapstructure:"SMTP_PORT"`
	SMTPUser     string `mapstructure:"SMTP_USER"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom     string `mapstructure:"SMTP_FROM"`
}

type ContextKey string

const (
	UserIDKey    = ContextKey("userID")
	RequestIDKey = ContextKey("request_id")
)

// Load reads configuration from secrets, environment variables, or defaults.
func Load() (config Config, err error) {
	// 1. Determine Environment First
	// We check OS Env directly first to decide how to load the rest
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	viper.Set("APP_ENV", env)

	// 2. Set Defaults based on Environment
	if env == "production" {
		viper.SetDefault("PORT", 8080)
		viper.SetDefault("RATE_LIMIT", 1000)
		viper.SetDefault("LOG_LEVEL", "info")
		viper.SetDefault("REQUEST_TIMEOUT_SECONDS", 30)
		viper.SetDefault("JWT_EXPIRATION_HOURS", 24)
	} else {
		viper.SetDefault("PORT", 8080)
		viper.SetDefault("RATE_LIMIT", 100)
		viper.SetDefault("LOG_LEVEL", "debug")
		viper.SetDefault("REQUEST_TIMEOUT_SECONDS", 60)
		viper.SetDefault("JWT_EXPIRATION_HOURS", 168)
		viper.SetDefault("DEFAULT_USER_USERNAME", "admin")
		viper.SetDefault("DEFAULT_USER_PASSWORD", "admin123!")
	}

	// Universal Defaults
	viper.SetDefault("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"})
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_SSL_MODE", "disable")
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", 6379)
	viper.SetDefault("SMTP_PORT", 587)

	// 3. Conditional Loading Logic
	if env == "development" {
		// --- DEVELOPMENT: Load from .env file ---
		// We try loading from current and parent directory
		_ = loadEnvFile(".env")
		_ = loadEnvFile("../.env")
	} else {
		// --- PRODUCTION: Load from Docker Secrets ---
		loadSecret("APP_SECRET", "app_secret")
		loadSecret("DATABASE_URL", "database_url")
		loadSecret("DB_HOST", "db_host")
		loadSecret("DB_PORT", "db_port")
		loadSecret("DB_USER", "db_user")
		loadSecret("DB_PASSWORD", "db_password")
		loadSecret("DB_NAME", "db_name")
		loadSecret("DB_SSL_MODE", "db_ssl_mode")
		loadSecret("REDIS_HOST", "redis_host")
		loadSecret("REDIS_PORT", "redis_port")
		loadSecret("REDIS_PASSWORD", "redis_password")
		loadSecret("SMTP_PASSWORD", "smtp_password")
	}

	// 4. AutomaticEnv (System Env Vars override everything loaded so far)
	viper.AutomaticEnv()

	// 5. Explicit Overrides (for specific manual bindings)
	bindExplicitEnvs()

	// 6. Unmarshal
	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}

	// 7. Post-Load Logic
	if config.DatabaseURL == "" {
		config.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			config.DbUser, config.DbPassword, config.DbHost, config.DbPort, config.DbName, config.DbSslMode,
		)
	}

	return
}

// loadSecret reads a file from /run/secrets and sets it in Viper
func loadSecret(key, name string) {
	candidates := []string{name, strings.ToUpper(name), strings.ToLower(name)}
	for _, filename := range candidates {
		path := fmt.Sprintf("/run/secrets/%s", filename)
		if _, err := os.Stat(path); err == nil {
			content, _ := os.ReadFile(path)
			if len(content) > 0 {
				viper.Set(key, strings.TrimSpace(string(content)))
				return
			}
		}
	}
}

// loadEnvFile parses a .env file and sets values into Viper AND os.Env
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes
		if len(value) > 1 && (value[0] == '"' || value[0] == '\'') && value[0] == value[len(value)-1] {
			value = value[1 : len(value)-1]
		}

		// LOGIC: Set in Viper (so Unmarshal works)
		// Only set if not already set by system env (precedence)
		if os.Getenv(key) == "" {
			viper.Set(key, value)
			os.Setenv(key, value) // Keep this if other libs rely on os.Getenv
		}
	}

	return scanner.Err()
}

func bindExplicitEnvs() {
	if host := os.Getenv("SMTP_HOST"); host != "" {
		viper.Set("SMTP_HOST", host)
	}
	if port := os.Getenv("SMTP_PORT"); port != "" {
		viper.Set("SMTP_PORT", port)
	}
	if from := os.Getenv("SMTP_FROM"); from != "" {
		viper.Set("SMTP_FROM", from)
	}
	if user := os.Getenv("ALERT_SMTP_USER"); user != "" {
		viper.Set("SMTP_USER", user)
	} else if user := os.Getenv("SMTP_USER"); user != "" {
		viper.Set("SMTP_USER", user)
	}
}

// Validate performs comprehensive configuration validation
func (c *Config) Validate() error {
	var errors []string

	if c.App_Secret == "" {
		errors = append(errors, "APP_SECRET is required")
	} else if len(c.App_Secret) < 32 {
		errors = append(errors, "APP_SECRET must be at least 32 characters long")
	}

	if c.DbUser == "" {
		errors = append(errors, "DB_USER is required")
	}
	if c.DbPassword == "" {
		errors = append(errors, "DB_PASSWORD is required")
	}
	if c.DbName == "" {
		errors = append(errors, "DB_NAME is required")
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
