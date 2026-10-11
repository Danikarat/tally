package ingestion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Danikarat/tally/internal/auth"
	"github.com/Danikarat/tally/internal/db"
	"github.com/Danikarat/tally/internal/ingestion"
	"github.com/go-chi/chi/v5"
)

func TestIdempotency_SequentialDuplicates(t *testing.T) {
	pool, jwtSecret, cleanup := setupTestEnvironment(t)
	defer cleanup()

	queries := db.New(pool)
	ctx := context.Background()

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        "seq-duplicate@tally.local",
		PasswordHash: "passhash",
		FullName:     db.Text("Sequential Test User"),
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	userIDStr := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		user.ID.Bytes[0:4], user.ID.Bytes[4:6], user.ID.Bytes[6:8], user.ID.Bytes[8:10], user.ID.Bytes[10:16])

	token, _, err := auth.GenerateToken(userIDStr, user.Email, jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	svc := ingestion.NewService(pool)
	handler := ingestion.NewHandler(svc, jwtSecret)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	now := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	batch := ingestion.IngestRequest{
		Candidates: []ingestion.TransactionCandidate{
			{
				SourceType:        "SMS",
				Provider:          "MTN",
				SenderIdentifier:  stringPtr("MobileMoney"),
				ExternalReference: stringPtr("REF-SEQ-100"),
				RawPayload:        stringPtr("Payment of GHS 80.00 made to MTN MoMo merchant. Ref: REF-SEQ-100."),
				AmountMinorUnits:  8000, // 80.00 GHS
				Currency:          "GHS",
				Direction:         "OUTBOUND",
				TransactionType:   "EXPENSE",
				Counterparty:      stringPtr("Merchant"),
				ProviderReference: stringPtr("REF-SEQ-100"),
				EventTimestamp:    now,
			},
		},
	}
	body, _ := json.Marshal(batch)

	// First submission
	req1 := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", bytes.NewReader(body))
	req1.Header.Set("Authorization", "Bearer "+token)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected status 200 on first submission, got %d (body: %s)", rec1.Code, rec1.Body.String())
	}

	var resp1 ingestion.IngestResponse
	if err := json.NewDecoder(rec1.Body).Decode(&resp1); err != nil {
		t.Fatalf("failed to decode resp1: %v", err)
	}
	if resp1.IngestedCount != 1 || resp1.DuplicateCount != 0 {
		t.Errorf("expected 1 ingested, 0 duplicates, got %d ingested, %d duplicates", resp1.IngestedCount, resp1.DuplicateCount)
	}
	if resp1.Items[0].Status != ingestion.StatusIngested {
		t.Errorf("expected status INGESTED, got %s", resp1.Items[0].Status)
	}
	firstTxID := *resp1.Items[0].TransactionID
	firstSrcID := *resp1.Items[0].SourceEventID

	// Second submission (identical payload)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200 on second submission, got %d (body: %s)", rec2.Code, rec2.Body.String())
	}

	var resp2 ingestion.IngestResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode resp2: %v", err)
	}
	if resp2.IngestedCount != 0 || resp2.DuplicateCount != 1 {
		t.Errorf("expected 0 ingested, 1 duplicate, got %d ingested, %d duplicates", resp2.IngestedCount, resp2.DuplicateCount)
	}
	if resp2.Items[0].Status != ingestion.StatusDuplicateIgnored {
		t.Errorf("expected status DUPLICATE_IGNORED, got %s", resp2.Items[0].Status)
	}
	if *resp2.Items[0].SourceEventID != firstSrcID {
		t.Errorf("expected same source_event_id %s, got %s", firstSrcID, *resp2.Items[0].SourceEventID)
	}
	if resp2.Items[0].TransactionID == nil || *resp2.Items[0].TransactionID != firstTxID {
		t.Errorf("expected same transaction_id %s, got %v", firstTxID, resp2.Items[0].TransactionID)
	}

	// Verify database rows
	txList, err := queries.ListTransactionsByUser(ctx, db.ListTransactionsByUserParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("failed to query transactions: %v", err)
	}
	if len(txList) != 1 {
		t.Fatalf("expected exactly 1 transaction row in database, got %d", len(txList))
	}
}

func TestIdempotency_ConcurrentSubmissions(t *testing.T) {
	pool, jwtSecret, cleanup := setupTestEnvironment(t)
	defer cleanup()

	queries := db.New(pool)
	ctx := context.Background()

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        "concurrent-dup@tally.local",
		PasswordHash: "passhash",
		FullName:     db.Text("Concurrent Test User"),
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	userIDStr := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		user.ID.Bytes[0:4], user.ID.Bytes[4:6], user.ID.Bytes[6:8], user.ID.Bytes[8:10], user.ID.Bytes[10:16])

	token, _, err := auth.GenerateToken(userIDStr, user.Email, jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	svc := ingestion.NewService(pool)
	handler := ingestion.NewHandler(svc, jwtSecret)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	now := time.Date(2026, 3, 22, 12, 30, 0, 0, time.UTC)
	batch := ingestion.IngestRequest{
		Candidates: []ingestion.TransactionCandidate{
			{
				SourceType:        "SMS",
				Provider:          "MTN",
				SenderIdentifier:  stringPtr("MobileMoney"),
				ExternalReference: stringPtr("REF-CONCURRENT-999"),
				RawPayload:        stringPtr("Payment of GHS 150.00 to Shoprite Ghana. Ref: REF-CONCURRENT-999."),
				AmountMinorUnits:  15000, // 150.00 GHS
				Currency:          "GHS",
				Direction:         "OUTBOUND",
				TransactionType:   "EXPENSE",
				Counterparty:      stringPtr("Shoprite"),
				ProviderReference: stringPtr("REF-CONCURRENT-999"),
				EventTimestamp:    now,
			},
		},
	}
	body, _ := json.Marshal(batch)

	const concurrency = 10
	var wg sync.WaitGroup
	var totalIngested atomic.Int32
	var totalDuplicates atomic.Int32
	var totalSuccess atomic.Int32

	startGate := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startGate // Synchronize start to maximize contention

			req := httptest.NewRequest(http.MethodPost, "/v1/transactions/ingest", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				totalSuccess.Add(1)
				var resp ingestion.IngestResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err == nil {
					totalIngested.Add(int32(resp.IngestedCount))
					totalDuplicates.Add(int32(resp.DuplicateCount))
				}
			}
		}()
	}

	// Release all goroutines concurrently
	close(startGate)
	wg.Wait()

	if int(totalSuccess.Load()) != concurrency {
		t.Errorf("expected all %d requests to succeed with 200, got %d", concurrency, totalSuccess.Load())
	}
	if totalIngested.Load() != 1 {
		t.Errorf("expected exactly 1 INGESTED across all concurrent requests, got %d", totalIngested.Load())
	}
	if totalDuplicates.Load() != concurrency-1 {
		t.Errorf("expected exactly %d DUPLICATE_IGNORED across concurrent requests, got %d", concurrency-1, totalDuplicates.Load())
	}

	// Verify database state: exactly 1 transaction and 1 source event created
	txList, err := queries.ListTransactionsByUser(ctx, db.ListTransactionsByUserParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("failed to query transactions: %v", err)
	}
	if len(txList) != 1 {
		t.Fatalf("expected exactly 1 transaction in database, got %d", len(txList))
	}

	eventsList, err := queries.ListSourceEventsByUser(ctx, db.ListSourceEventsByUserParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("failed to query source events: %v", err)
	}
	if len(eventsList) != 1 {
		t.Fatalf("expected exactly 1 source event in database, got %d", len(eventsList))
	}
}
