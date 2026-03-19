package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Postgres struct {
		Host     string
		Port     string
		User     string
		Password string
		DB       string
	}
	Redis struct {
		Host     string
		Port     string
		Password string
		DB       int
	}
	Slack struct {
		AppToken   string
		BotToken   string
		HmacSecret string
	}
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env not found")
	}

	return &Config{
		Postgres: struct {
			Host     string
			Port     string
			User     string
			Password string
			DB       string
		}{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "password"),
			DB:       getEnv("POSTGRES_DB", "attendance_db"),
		},
		Redis: struct {
			Host     string
			Port     string
			Password string
			DB       int
		}{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Slack: struct {
			AppToken   string
			BotToken   string
			HmacSecret string
		}{
			AppToken:   getEnv("SLACK_APP_TOKEN", ""),
			BotToken:   getEnv("SLACK_BOT_TOKEN", ""),
			HmacSecret: getEnv("HMAC_SECRET", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}
