// Package config provides application configuration management.
package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Slack    SlackConfig
	JWT      JWTConfig
}

// AppConfig holds general application settings.
type AppConfig struct {
	Name        string
	Version     string
	Env         string
	Port        string
	BaseURL     string
	FrontendURL string
	LogLevel    string
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DB              string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	PingTimeout     time.Duration
	URL             string
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	URL      string
}

// SlackConfig holds Slack application credentials.
// Only Client ID/Secret and Signing Secret are needed — no Bot/App tokens.
type SlackConfig struct {
	ClientID      string
	ClientSecret  string
	SigningSecret string
	RedirectURL   string
}

// JWTConfig holds JWT signing configuration.
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// Load reads configuration from environment variables and .env file.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("[config] .env file not found, using environment variables")
	}

	return &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "ZenReply"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Env:         getEnv("APP_ENV", "development"),
			Port:        getEnv("APP_PORT", "8080"),
			BaseURL:     getEnv("APP_BASE_URL", "http://localhost:8080"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:4200"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnv("POSTGRES_PORT", "5432"),
			User:            getEnv("POSTGRES_USER", "admin"),
			Password:        getEnv("POSTGRES_PASSWORD", "zenreply"),
			DB:              getEnv("POSTGRES_DB", "zenreply"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxConns:        int32(getEnvAsInt("POSTGRES_MAX_CONNS", 25)),
			MinConns:        int32(getEnvAsInt("POSTGRES_MIN_CONNS", 2)),
			MaxConnLifetime: getEnvAsDuration("POSTGRES_MAX_CONN_LIFETIME", 5*time.Minute),
			MaxConnIdleTime: getEnvAsDuration("POSTGRES_MAX_CONN_IDLE_TIME", 30*time.Minute),
			PingTimeout:     getEnvAsDuration("POSTGRES_PING_TIMEOUT", 15*time.Second),
			URL:             getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", "zenreply"),
			DB:       getEnvAsInt("REDIS_DB", 0),
			URL:      getEnv("REDIS_URL", ""),
		},
		Slack: SlackConfig{
			ClientID:      getEnv("SLACK_CLIENT_ID", ""),
			ClientSecret:  getEnv("SLACK_CLIENT_SECRET", ""),
			SigningSecret: getEnv("SLACK_SIGNING_SECRET", ""),
			RedirectURL:   getEnv("SLACK_REDIRECT_URL", "http://localhost:8080/api/v1/slack/callback"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "change-me-in-production"),
			Expiration: getEnvAsDuration("JWT_EXPIRATION", 24*time.Hour),
		},
	}
}

// DSN returns the PostgreSQL Data Source Name.
func (c *PostgresConfig) DSN() string {
	if c.URL != "" {
		return c.URL
	}
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + c.Port + "/" + c.DB + "?sslmode=" + c.SSLMode
}

// Addr returns the Redis address string.
func (c *RedisConfig) Addr() string {
	return c.Host + ":" + c.Port
}

// IsProduction returns true if the app is running in production mode.
func (c *AppConfig) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if valueStr := getEnv(key, ""); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if valueStr := getEnv(key, ""); valueStr != "" {
		if value, err := time.ParseDuration(valueStr); err == nil {
			return value
		}
	}
	return fallback
}
