package config

import (
	"os"
	"testing"
	"time"
)

func TestConfig_Defaults(t *testing.T) {
	// Clear any environment variables that might interfere
	os.Unsetenv("ENV")
	os.Unsetenv("PORT")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("JWT_SECRET")

	cfg, err := LoadWithOptions(LoadOptions{IgnoreDotEnv: true})
	if err != nil {
		t.Fatalf("expected no error loading defaults, got %v", err)
	}

	if cfg.Environment != "development" {
		t.Errorf("expected Environment 'development', got %q", cfg.Environment)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", cfg.Port)
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("expected DB.Host 'localhost', got %q", cfg.DB.Host)
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("expected DB.Port 5432, got %d", cfg.DB.Port)
	}
	if cfg.DB.Name != "tally_dev" {
		t.Errorf("expected DB.Name 'tally_dev', got %q", cfg.DB.Name)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("expected ReadTimeout 5s, got %v", cfg.ReadTimeout)
	}

	expectedDSN := "postgres://tally:tally_secret@localhost:5432/tally_dev?sslmode=disable"
	if cfg.DB.DSN() != expectedDSN {
		t.Errorf("expected DSN %q, got %q", expectedDSN, cfg.DB.DSN())
	}
}

func TestConfig_EnvOverrides(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("PORT", "9090")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "prod_user")
	t.Setenv("DB_PASSWORD", "prod_pass")
	t.Setenv("DB_NAME", "prod_tally")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("JWT_SECRET", "super-secret-production-token")
	t.Setenv("READ_TIMEOUT", "15s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	if cfg.Environment != "production" {
		t.Errorf("expected Environment 'production', got %q", cfg.Environment)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected Port 9090, got %d", cfg.Port)
	}
	if cfg.DB.Host != "db.internal" {
		t.Errorf("expected DB.Host 'db.internal', got %q", cfg.DB.Host)
	}
	if cfg.DB.Port != 5433 {
		t.Errorf("expected DB.Port 5433, got %d", cfg.DB.Port)
	}
	if cfg.DB.User != "prod_user" {
		t.Errorf("expected DB.User 'prod_user', got %q", cfg.DB.User)
	}
	if cfg.DB.SSLMode != "require" {
		t.Errorf("expected DB.SSLMode 'require', got %q", cfg.DB.SSLMode)
	}
	if cfg.Auth.JWTSecret != "super-secret-production-token" {
		t.Errorf("expected JWT_SECRET to match override")
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("expected ReadTimeout 15s, got %v", cfg.ReadTimeout)
	}
}
