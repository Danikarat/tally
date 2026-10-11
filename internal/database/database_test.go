package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Danikarat/tally/internal/config"
)

func TestMigrations_LifecycleAndConstraints(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	adminPool, err := NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		t.Skipf("skipping test: database not reachable at %s (%v)", cfg.DB.DSN(), err)
		return
	}
	defer adminPool.Close()

	testDBName := "tally_test_migration_lifecycle"
	_, _ = adminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", testDBName))
	_, err = adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		t.Fatalf("failed to create isolated test database: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", testDBName))
	}()

	testCfg := *cfg
	testCfg.DB.Name = testDBName
	testDSN := testCfg.DB.DSN()

	testPool, err := NewPool(ctx, testDSN)
	if err != nil {
		t.Fatalf("failed to connect to isolated test database: %v", err)
	}
	defer testPool.Close()

	// 1. Check initial version is 0
	ver, dirty, err := MigrateVersion(testDSN)
	if err != nil {
		t.Fatalf("unexpected error checking version on fresh database: %v", err)
	}
	if ver != 0 || dirty {
		t.Fatalf("expected version 0 and clean, got version %d, dirty=%v", ver, dirty)
	}

	// 2. Apply migrations Up
	if err := MigrateUp(testDSN); err != nil {
		t.Fatalf("failed to migrate up: %v", err)
	}

	ver, dirty, err = MigrateVersion(testDSN)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if ver != 4 || dirty {
		t.Fatalf("expected version 4 and not dirty, got version %d, dirty=%v", ver, dirty)
	}

	// 3. Test Inserting a valid user
	var userID string
	err = testPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "test@example.com", "argon2id_hash_sample", "Test User").Scan(&userID)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	// 4. Test Inserting a valid financial account
	var accountID string
	err = testPool.QueryRow(ctx, `
		INSERT INTO financial_accounts (user_id, account_type, provider_name, account_identifier_masked, currency, balance_minor_units)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, userID, "MOMO", "MTN MoMo", "***1234", "GHS", 15000).Scan(&accountID)
	if err != nil {
		t.Fatalf("failed to insert valid financial account: %v", err)
	}

	// 5. Account Constraint Tests:
	// 5a. Invalid account_type
	_, err = testPool.Exec(ctx, `
		INSERT INTO financial_accounts (user_id, account_type, provider_name, currency)
		VALUES ($1, $2, $3, $4)
	`, userID, "INVALID_TYPE", "Crypto", "GHS")
	if err == nil {
		t.Error("expected check constraint error for invalid account_type, got nil")
	}

	// 5b. Invalid currency length
	_, err = testPool.Exec(ctx, `
		INSERT INTO financial_accounts (user_id, account_type, provider_name, currency)
		VALUES ($1, $2, $3, $4)
	`, userID, "BANK", "Stanbic", "TOOLONG")
	if err == nil {
		t.Error("expected check constraint error for invalid currency length, got nil")
	}

	// 6. Test Inserting a valid source_event
	var sourceEventID string
	payloadHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	err = testPool.QueryRow(ctx, `
		INSERT INTO source_events (user_id, source_type, provider, sender_identifier, external_reference, payload_hash, parse_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, userID, "SMS", "MTN", "MobileMoney", "REF12345", payloadHash, "PARSED").Scan(&sourceEventID)
	if err != nil {
		t.Fatalf("failed to insert valid source_event: %v", err)
	}

	// 7. Source Event Constraint Tests:
	// 7a. Duplicate (user_id, payload_hash) must violate unique constraint
	_, err = testPool.Exec(ctx, `
		INSERT INTO source_events (user_id, source_type, provider, payload_hash)
		VALUES ($1, $2, $3, $4)
	`, userID, "SMS", "MTN", payloadHash)
	if err == nil {
		t.Error("expected unique constraint violation for duplicate payload_hash on same user, got nil")
	}

	// 7b. Invalid source_type
	_, err = testPool.Exec(ctx, `
		INSERT INTO source_events (user_id, source_type, provider, payload_hash)
		VALUES ($1, $2, $3, $4)
	`, userID, "TELEGRAM", "MTN", "unique_hash_123")
	if err == nil {
		t.Error("expected check constraint violation for invalid source_type, got nil")
	}

	// 8. Test Inserting a valid normalized transaction
	var transactionID string
	eventTime := time.Date(2026, 10, 11, 14, 30, 0, 0, time.UTC)
	err = testPool.QueryRow(ctx, `
		INSERT INTO transactions (
			user_id, account_id, source_event_id, amount_minor_units, currency,
			direction, transaction_type, counterparty, provider_reference, status, event_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, userID, accountID, sourceEventID, 5000, "GHS", "OUTBOUND", "EXPENSE", "KFC Ghana", "REF12345", "COMPLETED", eventTime).Scan(&transactionID)
	if err != nil {
		t.Fatalf("failed to insert valid transaction: %v", err)
	}

	// 9. Transaction Constraint Tests:
	// 9a. Negative monetary amount
	_, err = testPool.Exec(ctx, `
		INSERT INTO transactions (user_id, account_id, amount_minor_units, currency, direction, transaction_type, event_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, accountID, -500, "GHS", "OUTBOUND", "EXPENSE", eventTime)
	if err == nil {
		t.Error("expected check constraint violation for negative amount_minor_units, got nil")
	}

	// 9b. Invalid direction
	_, err = testPool.Exec(ctx, `
		INSERT INTO transactions (user_id, account_id, amount_minor_units, currency, direction, transaction_type, event_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, accountID, 1000, "GHS", "SIDEWAYS", "EXPENSE", eventTime)
	if err == nil {
		t.Error("expected check constraint violation for invalid direction, got nil")
	}

	// 9c. Invalid transaction_type
	_, err = testPool.Exec(ctx, `
		INSERT INTO transactions (user_id, account_id, amount_minor_units, currency, direction, transaction_type, event_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, accountID, 1000, "GHS", "OUTBOUND", "INVALID_TYPE", eventTime)
	if err == nil {
		t.Error("expected check constraint violation for invalid transaction_type, got nil")
	}

	// 10. Rollback Down to verify clean teardown of all 4 migrations
	if err := MigrateDown(testDSN); err != nil {
		t.Fatalf("failed to migrate down: %v", err)
	}

	ver, _, err = MigrateVersion(testDSN)
	if err != nil {
		t.Fatalf("unexpected error checking version after rollback: %v", err)
	}
	if ver != 0 {
		t.Fatalf("expected version 0 after rollback, got %d", ver)
	}
}
