package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration.
// Values are loaded from environment variables with sensible defaults.
type Config struct {
	// Server
	HTTPPort int
	GRPCPort int
	Version  string
	Env      string

	// Database
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// Auth
	JWTSecret     string
	JWTExpiration int // hours

	// Logging
	LogLevel string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:      getEnvInt("APP_PORT", 8080),
		GRPCPort:      getEnvInt("GRPC_PORT", 9090),
		Version:       getEnv("APP_VERSION", "1.0.0"),
		Env:           getEnv("APP_ENV", "development"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnvInt("DB_PORT", 5432),
		DBName:        getEnv("DB_NAME", "microservice"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBSSLMode:     getEnv("DB_SSL_MODE", "disable"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpiration: getEnvInt("JWT_EXPIRATION_HOURS", 24),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// DatabaseURL returns the PostgreSQL connection string.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

// RedisURL returns the Redis connection string.
func (c *Config) RedisURL() string {
	if c.RedisPassword != "" {
		return fmt.Sprintf("redis://:%s@%s:%d/%d", c.RedisPassword, c.RedisHost, c.RedisPort, c.RedisDB)
	}
	return fmt.Sprintf("redis://%s:%d/%d", c.RedisHost, c.RedisPort, c.RedisDB)
}

// validate checks that required configuration is present.
func (c *Config) validate() error {
	if c.Env == "production" && c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required in production")
	}
	if c.JWTSecret == "" {
		c.JWTSecret = "dev-secret-change-in-production"
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
