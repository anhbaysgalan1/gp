package config

import (
	"fmt"
	"os"
)

type Config struct {
	// Environment
	Environment string

	// Database
	DatabaseURL      string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	PostgresHost     string
	PostgresPort     string

	// Redis
	RedisURL      string
	RedisPassword string

	// Server
	Port string

	// Authentication
	JWTSecret string

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// Formance
	FormanceAPIURL     string
	FormanceAPIKey     string
	FormanceLedgerName string
	FormanceCurrency   string
}

func Load() *Config {
	return &Config{
		// Environment
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),

		// Database
		DatabaseURL:      getEnvOrDefault("DATABASE_URL", ""),
		PostgresDB:       getEnvOrDefault("POSTGRES_DB", "poker_platform"),
		PostgresUser:     getEnvOrDefault("POSTGRES_USER", "poker_user"),
		PostgresPassword: getEnvOrDefault("POSTGRES_PASSWORD", "poker_password"),
		PostgresHost:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnvOrDefault("POSTGRES_PORT", "5432"),

		// Redis
		RedisURL:      getEnvOrDefault("REDIS_URL", "redis://localhost:6379"),
		RedisPassword: getEnvOrDefault("REDIS_PASSWORD", "password"),

		// Server
		Port: getEnvOrDefault("PORT", "8080"),

		// Authentication
		JWTSecret: getEnvOrDefault("JWT_SECRET", "poker-platform-secret-key-change-in-production"),

		// SMTP
		SMTPHost:     getEnvOrDefault("SMTP_HOST", "smtp.resend.com"),
		SMTPPort:     getEnvOrDefault("SMTP_PORT", "587"),
		SMTPUsername: getEnvOrDefault("SMTP_USERNAME", "resend"),
		SMTPPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnvOrDefault("SMTP_FROM", "info@hihi.mn"),

		// Formance
		FormanceAPIURL:     getEnvOrDefault("FORMANCE_API_URL", "http://localhost:3068"),
		FormanceAPIKey:     getEnvOrDefault("FORMANCE_API_KEY", ""),
		FormanceLedgerName: getEnvOrDefault("FORMANCE_LEDGER_NAME", "poker-platform-mnt"),
		FormanceCurrency:   getEnvOrDefault("FORMANCE_CURRENCY", "MNT"),
	}
}

func (c *Config) GetDatabaseURL() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDB,
	)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
