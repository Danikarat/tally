package database

import (
	"context"
	"testing"
	"time"

	"github.com/Danikarat/tally/internal/config"
)

func TestMigrations_LifecycleAndConstraints(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	dsn := cfg.DB.DSN()

	// Ensure database connection can be established
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Skipf("skipping test: database not reachable at %s (%v)", dsn, err)
		return
	}
	defer pool.Close()

	// 1. Rollback any existing migrations to start clean
	_ = MigrateDown(dsn)

	// Verify version is 0 after rollback
	ver, dirty, err := MigrateVersion(dsn)
	if err != nil {
		t.Fatalf("unexpected error checking version after rollback: %v", err)
	}
	if ver != 0 || dirty {
		t.Fatalf("expected version 0 and clean, got version %d, dirty=%v", ver, dirty)
	}

	// 2. Apply migrations Up
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("failed to migrate up: %v", err)
	}

	ver, dirty, err = MigrateVersion(dsn)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if ver != 2 || dirty {
		t.Fatalf("expected version 2 and not dirty, got version %d, dirty=%v", ver, dirty)
	}

	// 3. Test Inserting a valid user
	var userID string
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "test@example.com", "argon2id_hash_sample", "Test User").Scan(&userID)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	// 4. Test Inserting a valid financial account
	var accountID string
	err = pool.QueryRow(ctx, `
		INSERT INTO financial_accounts (user_id, account_type, provider_name, account_identifier_masked, currency, balance_minor_units)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, userID, "MOMO", "MTN MoMo", "***1234", "GHS", 15000).Scan(&accountID)
	if err != nil {
		t.Fatalf("failed to insert valid financial account: %v", err)
	}

	// 5. Constraint Test: Invalid account_type (not in MOMO, BANK, CASH, OTHER)
	_, err = pool.Exec(ctx, `
		INSERT INTO financial_accounts (user_id, account_type, provider_name, currency)
		VALUES ($1, $2, $3, $4)
	`, userID, "INVALID_TYPE", "Crypto", "GHS")
	if err == nil {
		t.Error("expected check constraint error for invalid account_type, got nil")
	}

	// 6. Constraint Test: Invalid currency length (must be 3 chars)
	_, err = pool.Exec(ctx, `
		INSERT INTO financial_accounts (user_id, account_type, provider_name, currency)
		VALUES ($1, $2, $3, $4)
	`, userID, "BANK", "Stanbic", "TOOLONG")
	if err == nil {
		t.Error("expected check constraint error for invalid currency length, got nil")
	}

	// 7. Constraint Test: Non-existent user_id (foreign key violation)
	_, err = pool.Exec(ctx, `
		INSERT INTO financial_accounts (user_id, account_type, provider_name, currency)
		VALUES ('00000000-0000-0000-0000-000000000000', 'MOMO', 'MTN MoMo', 'GHS')
	`)
	if err == nil {
		t.Error("expected foreign key violation for non-existent user_id, got nil")
	}

	// 8. Rollback Down to verify teardown
	if err := MigrateDown(dsn); err != nil {
		t.Fatalf("failed to migrate down: %v", err)
	}

	// 9. Re-apply Up so dev database remains ready for subsequent use
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("failed to re-apply migrations up: %v", err)
	}
}
