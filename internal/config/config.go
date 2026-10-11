package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime application configuration.
type Config struct {
	Environment  string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	DB           DatabaseConfig
	Auth         AuthConfig
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

// AuthConfig holds JWT and security settings.
type AuthConfig struct {
	JWTSecret     string
	TokenDuration time.Duration
}

// LoadOptions configures configuration loading behavior.
type LoadOptions struct {
	IgnoreDotEnv bool
}

// Load reads configuration from environment variables and an optional .env file.
func Load() (*Config, error) {
	return LoadWithOptions(LoadOptions{})
}

// LoadWithOptions reads configuration with customizable options.
func LoadWithOptions(opts LoadOptions) (*Config, error) {
	if !opts.IgnoreDotEnv {
		for _, path := range []string{".env", "../.env", "../../.env"} {
			if err := godotenv.Load(path); err == nil {
				break
			}
		}
	}

	cfg := &Config{
		Environment:  getEnv("ENV", "development"),
		Port:         getEnvInt("PORT", 8080),
		ReadTimeout:  getEnvDuration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  getEnvDuration("IDLE_TIMEOUT", 120*time.Second),
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "tally"),
			Password: getEnv("DB_PASSWORD", "tally_secret"),
			Name:     getEnv("DB_NAME", "tally_dev"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Auth: AuthConfig{
			JWTSecret:     getEnv("JWT_SECRET", "dev-insecure-secret-change-in-production-min-32-chars"),
			TokenDuration: getEnvDuration("JWT_DURATION", 24*time.Hour),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if valStr, ok := os.LookupEnv(key); ok && valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if valStr, ok := os.LookupEnv(key); ok && valStr != "" {
		if val, err := time.ParseDuration(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}
