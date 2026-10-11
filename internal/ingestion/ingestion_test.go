package ingestion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Danikarat/tally/internal/auth"
	"github.com/Danikarat/tally/internal/config"
	"github.com/Danikarat/tally/internal/database"
	"github.com/Danikarat/tally/internal/db"
	"github.com/Danikarat/tally/internal/ingestion"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestEnvironment(t *testing.T) (*pgxpool.Pool, string, func()) {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	adminPool, err := database.NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		t.Skipf("skipping test: database not reachable at %s (%v)", cfg.DB.DSN(), err)
		return nil, "", func() {}
	}

	testDBName := fmt.Sprintf("tally_test_ingestion_%d", time.Now().UnixNano()%100000)
	_, _ = adminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", testDBName))
	_, err = adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		adminPool.Close()
		t.Fatalf("failed to create isolated test database: %v", err)
	}

	testCfg := *cfg
	testCfg.DB.Name = testDBName
	testDSN := testCfg.DB.DSN()

	testPool, err := database.NewPool(ctx, testDSN)
	if err != nil {
		adminPool.Close()
		t.Fatalf("failed to connect to isolated test database: %v", err)
	}

	if err := database.MigrateUp(testDSN); err != nil {
		testPool.Close()
		adminPool.Close()
		t.Fatalf("failed to apply migrations: %v", err)
	}

	cleanup := func() {
		testPool.Close()
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", testDBName))
		adminPool.Close()
	}

	return testPool, cfg.Auth.JWTSecret, cleanup
}

func stringPtr(s string) *string {
	return &s
}

func TestIngest_ValidationErrors(t *testing.T) {
	pool, jwtSecret, cleanup := setupTestEnvironment(t)
	defer cleanup()

	svc := ingestion.NewService(pool)
	handler := ingestion.NewHandler(svc, jwtSecret)

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	token, _, err := auth.GenerateToken("00000000-0000-0000-0000-000000000001", "test@example.com", jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// 1. Unauthenticated request -> 401
	{
		body := bytes.NewReader([]byte(`{"candidates":[]}`))
		req := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", body)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 for unauthenticated request, got %d", rec.Code)
		}
	}

	// 2. Empty batch -> 400
	{
		body := bytes.NewReader([]byte(`{"candidates":[]}`))
		req := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", body)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty batch, got %d", rec.Code)
		}
	}

	// 3. Malformed JSON -> 400
	{
		body := bytes.NewReader([]byte(`{"candidates": invalid_json}`))
		req := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", body)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for malformed json, got %d", rec.Code)
		}
	}

	// 4. Invalid candidate fields (non-positive amount, invalid direction, invalid currency) -> 422
	{
		ingestReq := ingestion.IngestRequest{
			Candidates: []ingestion.TransactionCandidate{
				{
					SourceType:       "SMS",
					Provider:         "MTN",
					AmountMinorUnits: 0, // Invalid: must be > 0
					Currency:         "GH", // Invalid: must be 3 characters
					Direction:        "INVALID_DIR", // Invalid
					TransactionType:  "EXPENSE",
					EventTimestamp:   time.Now().UTC(),
				},
			},
		}
		data, _ := json.Marshal(ingestReq)
		req := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422 for field validation errors, got %d", rec.Code)
		}

		var errResp struct {
			Error   string                            `json:"error"`
			Details []ingestion.ValidationErrorDetail `json:"details"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode validation error response: %v", err)
		}
		if len(errResp.Details) < 3 {
			t.Errorf("expected at least 3 validation error details, got %d", len(errResp.Details))
		}
	}
}

func TestIngest_ValidBatchAndTenantIsolation(t *testing.T) {
	pool, jwtSecret, cleanup := setupTestEnvironment(t)
	defer cleanup()

	queries := db.New(pool)
	ctx := context.Background()

	// Create User 1
	u1, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        "user1@tally.local",
		PasswordHash: "secret123",
		FullName:     db.Text("User One"),
	})
	if err != nil {
		t.Fatalf("failed to create user 1: %v", err)
	}

	// Create User 2
	u2, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        "user2@tally.local",
		PasswordHash: "secret123",
		FullName:     db.Text("User Two"),
	})
	if err != nil {
		t.Fatalf("failed to create user 2: %v", err)
	}

	u1IDStr := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u1.ID.Bytes[0:4], u1.ID.Bytes[4:6], u1.ID.Bytes[6:8], u1.ID.Bytes[8:10], u1.ID.Bytes[10:16])

	token1, _, err := auth.GenerateToken(u1IDStr, u1.Email, jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token for user 1: %v", err)
	}

	svc := ingestion.NewService(pool)
	handler := ingestion.NewHandler(svc, jwtSecret)

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	now := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	batch := ingestion.IngestRequest{
		Candidates: []ingestion.TransactionCandidate{
			{
				SourceType:             "SMS",
				Provider:               "MTN",
				SenderIdentifier:       stringPtr("MobileMoney"),
				ExternalReference:      stringPtr("EXT-101"),
				RawPayload:             stringPtr("Payment of GHS 45.00 made to Fuel Station. Ref: 101."),
				AmountMinorUnits:       4500, // 45.00 GHS
				Currency:               "GHS",
				Direction:              "OUTBOUND",
				TransactionType:        "EXPENSE",
				Counterparty:           stringPtr("Fuel Station"),
				ProviderReference:      stringPtr("TX-MTN-101"),
				EventTimestamp:         now,
			},
			{
				SourceType:             "SMS",
				Provider:               "MTN",
				SenderIdentifier:       stringPtr("MobileMoney"),
				ExternalReference:      stringPtr("EXT-102"),
				RawPayload:             stringPtr("Received GHS 120.00 from Ama Mensah. Ref: 102."),
				AmountMinorUnits:       12000, // 120.00 GHS
				Currency:               "GHS",
				Direction:              "INBOUND",
				TransactionType:        "INCOME",
				Counterparty:           stringPtr("Ama Mensah"),
				ProviderReference:      stringPtr("TX-MTN-102"),
				EventTimestamp:         now.Add(1 * time.Hour),
			},
		},
	}

	payloadBytes, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", bytes.NewReader(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+token1)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp ingestion.IngestResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalReceived != 2 {
		t.Errorf("expected TotalReceived=2, got %d", resp.TotalReceived)
	}
	if resp.IngestedCount != 2 {
		t.Errorf("expected IngestedCount=2, got %d", resp.IngestedCount)
	}
	if resp.DuplicateCount != 0 {
		t.Errorf("expected DuplicateCount=0, got %d", resp.DuplicateCount)
	}
	for i, item := range resp.Items {
		if item.Status != ingestion.StatusIngested {
			t.Errorf("item %d expected status INGESTED, got %s", i, item.Status)
		}
		if item.TransactionID == nil || *item.TransactionID == "" {
			t.Errorf("item %d expected non-empty transaction_id", i)
		}
		if item.SourceEventID == nil || *item.SourceEventID == "" {
			t.Errorf("item %d expected non-empty source_event_id", i)
		}
	}

	// Verify Tenant Isolation: User 1 has 2 transactions, User 2 has 0 transactions
	u1Tx, err := queries.ListTransactionsByUser(ctx, db.ListTransactionsByUserParams{
		UserID: u1.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("failed to list user 1 transactions: %v", err)
	}
	if len(u1Tx) != 2 {
		t.Errorf("expected user 1 to have 2 transactions, got %d", len(u1Tx))
	}

	u2Tx, err := queries.ListTransactionsByUser(ctx, db.ListTransactionsByUserParams{
		UserID: u2.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("failed to list user 2 transactions: %v", err)
	}
	if len(u2Tx) != 0 {
		t.Errorf("expected user 2 to have 0 transactions, got %d", len(u2Tx))
	}

	// Idempotency check: Re-submitting the exact same batch by User 1 returns DUPLICATE_IGNORED
	req2 := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", bytes.NewReader(payloadBytes))
	req2.Header.Set("Authorization", "Bearer "+token1)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on duplicate submission, got %d", rec2.Code)
	}

	var resp2 ingestion.IngestResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode duplicate response: %v", err)
	}

	if resp2.DuplicateCount != 2 {
		t.Errorf("expected DuplicateCount=2 on resubmission, got %d", resp2.DuplicateCount)
	}
	if resp2.IngestedCount != 0 {
		t.Errorf("expected IngestedCount=0 on resubmission, got %d", resp2.IngestedCount)
	}

	// Verify row counts did not increase
	u1TxAfter, err := queries.ListTransactionsByUser(ctx, db.ListTransactionsByUserParams{
		UserID: u1.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("failed to list user 1 transactions after resubmission: %v", err)
	}
	if len(u1TxAfter) != 2 {
		t.Errorf("expected user 1 transactions count to remain 2, got %d", len(u1TxAfter))
	}
}
