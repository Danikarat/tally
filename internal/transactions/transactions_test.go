package transactions_test

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
	"github.com/Danikarat/tally/internal/transactions"
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

	testDBName := fmt.Sprintf("tally_test_tx_%d", time.Now().UnixNano()%100000)
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

func uuidToString(src [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		src[0:4], src[4:6], src[6:8], src[8:10], src[10:16])
}

func TestTransactions_ListWithFiltersAndPagination(t *testing.T) {
	pool, jwtSecret, cleanup := setupTestEnvironment(t)
	defer cleanup()

	queries := db.New(pool)
	ctx := context.Background()

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        "kofi.list@tally.local",
		PasswordHash: "passhash123",
		FullName:     db.Text("Kofi List"),
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	userIDStr := uuidToString(user.ID.Bytes)
	token, _, err := auth.GenerateToken(userIDStr, user.Email, jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	account1, err := queries.CreateFinancialAccount(ctx, db.CreateFinancialAccountParams{
		UserID:                  user.ID,
		AccountType:             "MOMO",
		ProviderName:            "MTN MoMo",
		AccountIdentifierMasked: db.Text("024****111"),
		Currency:                "GHS",
		BalanceMinorUnits:       10000,
		IsDefault:               true,
		IsActive:                true,
	})
	if err != nil {
		t.Fatalf("failed to create account 1: %v", err)
	}

	account2, err := queries.CreateFinancialAccount(ctx, db.CreateFinancialAccountParams{
		UserID:                  user.ID,
		AccountType:             "MOMO",
		ProviderName:            "Telecel Cash",
		AccountIdentifierMasked: db.Text("020****222"),
		Currency:                "GHS",
		BalanceMinorUnits:       5000,
		IsDefault:               false,
		IsActive:                true,
	})
	if err != nil {
		t.Fatalf("failed to create account 2: %v", err)
	}

	// Insert 5 test transactions with different dates and types
	baseDate := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	types := []string{"EXPENSE", "EXPENSE", "INCOME", "TRANSFER", "EXPENSE"}
	accounts := []db.FinancialAccount{account1, account1, account2, account1, account2}

	for i := 0; i < 5; i++ {
		_, err := queries.InsertTransaction(ctx, db.InsertTransactionParams{
			UserID:                 user.ID,
			AccountID:              accounts[i].ID,
			AmountMinorUnits:       int64((i + 1) * 2000), // 20.00, 40.00, 60.00... GHS
			Currency:               "GHS",
			Direction:              "OUTBOUND",
			TransactionType:        types[i],
			Counterparty:           db.Text(fmt.Sprintf("Merchant %d", i+1)),
			CounterpartyIdentifier: db.NullText(),
			ProviderReference:      db.Text(fmt.Sprintf("REF-%d", i+1)),
			Status:                 "COMPLETED",
			UserCategoryID:         db.NullUUID(),
			UserNotes:              db.NullText(),
			UserModified:           false,
			EventTimestamp:         db.Timestamptz(baseDate.Add(time.Duration(i*24) * time.Hour)),
		})
		if err != nil {
			t.Fatalf("failed to insert transaction %d: %v", i, err)
		}
	}

	svc := transactions.NewService(pool)
	handler := transactions.NewHandler(svc, jwtSecret)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	// 1. Unfiltered list (all 5 items)
	{
		req := httptest.NewRequest(http.MethodGet, "/v1/transactions", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp transactions.ListTransactionsResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Pagination.Total != 5 {
			t.Errorf("expected total=5, got %d", resp.Pagination.Total)
		}
		if len(resp.Data) != 5 {
			t.Errorf("expected 5 transactions, got %d", len(resp.Data))
		}
	}

	// 2. Pagination: limit=2, offset=1
	{
		req := httptest.NewRequest(http.MethodGet, "/v1/transactions?limit=2&offset=1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp transactions.ListTransactionsResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Pagination.Total != 5 {
			t.Errorf("expected total=5, got %d", resp.Pagination.Total)
		}
		if len(resp.Data) != 2 {
			t.Errorf("expected 2 items, got %d", len(resp.Data))
		}
		if resp.Pagination.Offset != 1 || resp.Pagination.Limit != 2 {
			t.Errorf("unexpected pagination params: %+v", resp.Pagination)
		}
	}

	// 3. Filter by account_id
	{
		acc1Str := uuidToString(account1.ID.Bytes)
		req := httptest.NewRequest(http.MethodGet, "/v1/transactions?account_id="+acc1Str, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp transactions.ListTransactionsResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Pagination.Total != 3 {
			t.Errorf("expected total=3 for account 1, got %d", resp.Pagination.Total)
		}
		for _, tx := range resp.Data {
			if tx.AccountID != acc1Str {
				t.Errorf("expected account_id %s, got %s", acc1Str, tx.AccountID)
			}
		}
	}

	// 4. Filter by transaction_type
	{
		req := httptest.NewRequest(http.MethodGet, "/v1/transactions?type=EXPENSE", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp transactions.ListTransactionsResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Pagination.Total != 3 {
			t.Errorf("expected total=3 EXPENSE items, got %d", resp.Pagination.Total)
		}
		for _, tx := range resp.Data {
			if tx.TransactionType != "EXPENSE" {
				t.Errorf("expected EXPENSE, got %s", tx.TransactionType)
			}
		}
	}

	// 5. Filter by Date Range (from_date and to_date)
	{
		fromDate := "2026-03-11T00:00:00Z"
		toDate := "2026-03-13T23:59:59Z"
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/transactions?from_date=%s&to_date=%s", fromDate, toDate), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp transactions.ListTransactionsResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		// Indices 1, 2, 3 correspond to March 11, 12, 13
		if resp.Pagination.Total != 3 {
			t.Errorf("expected 3 items in date range, got %d", resp.Pagination.Total)
		}
	}

	// 6. Invalid date format returns 400 Bad Request
	{
		req := httptest.NewRequest(http.MethodGet, "/v1/transactions?from_date=not-a-date", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid date, got %d", rec.Code)
		}
	}
}

func TestTransactions_GetAndPatch_TenantIsolation(t *testing.T) {
	pool, jwtSecret, cleanup := setupTestEnvironment(t)
	defer cleanup()

	queries := db.New(pool)
	ctx := context.Background()

	// Create User 1
	u1, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        "u1@tally.local",
		PasswordHash: "secret123",
		FullName:     db.Text("User One"),
	})
	if err != nil {
		t.Fatalf("failed to create user 1: %v", err)
	}

	// Create User 2
	u2, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        "u2@tally.local",
		PasswordHash: "secret123",
		FullName:     db.Text("User Two"),
	})
	if err != nil {
		t.Fatalf("failed to create user 2: %v", err)
	}

	u1IDStr := uuidToString(u1.ID.Bytes)
	u2IDStr := uuidToString(u2.ID.Bytes)

	token1, _, _ := auth.GenerateToken(u1IDStr, u1.Email, jwtSecret, 1*time.Hour)
	token2, _, _ := auth.GenerateToken(u2IDStr, u2.Email, jwtSecret, 1*time.Hour)

	acc1, err := queries.CreateFinancialAccount(ctx, db.CreateFinancialAccountParams{
		UserID:                  u1.ID,
		AccountType:             "MOMO",
		ProviderName:            "MTN MoMo",
		AccountIdentifierMasked: db.Text("024****123"),
		Currency:                "GHS",
		BalanceMinorUnits:       5000,
		IsDefault:               true,
		IsActive:                true,
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	tx1, err := queries.InsertTransaction(ctx, db.InsertTransactionParams{
		UserID:                 u1.ID,
		AccountID:              acc1.ID,
		AmountMinorUnits:       9500, // 95.00 GHS
		Currency:               "GHS",
		Direction:              "OUTBOUND",
		TransactionType:        "EXPENSE",
		Counterparty:           db.Text("Supermarket"),
		CounterpartyIdentifier: db.NullText(),
		ProviderReference:      db.Text("REF-9500"),
		Status:                 "COMPLETED",
		UserCategoryID:         db.NullUUID(),
		UserNotes:              db.NullText(),
		UserModified:           false,
		EventTimestamp:         db.Timestamptz(time.Now().UTC()),
	})
	if err != nil {
		t.Fatalf("failed to insert transaction: %v", err)
	}

	tx1IDStr := uuidToString(tx1.ID.Bytes)

	svc := transactions.NewService(pool)
	handler := transactions.NewHandler(svc, jwtSecret)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	// 1. User 1 gets own transaction -> 200 OK
	{
		req := httptest.NewRequest(http.MethodGet, "/v1/transactions/"+tx1IDStr, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp struct {
			Transaction transactions.TransactionResponse `json:"transaction"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode get response: %v", err)
		}
		if resp.Transaction.ID != tx1IDStr {
			t.Errorf("expected tx id %s, got %s", tx1IDStr, resp.Transaction.ID)
		}
		if resp.Transaction.AmountMinorUnits != 9500 {
			t.Errorf("expected amount 9500, got %d", resp.Transaction.AmountMinorUnits)
		}
	}

	// 2. User 2 attempts to get User 1's transaction -> 404 Not Found (tenant isolation)
	{
		req := httptest.NewRequest(http.MethodGet, "/v1/transactions/"+tx1IDStr, nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for cross-tenant access, got %d", rec.Code)
		}
	}

	// 3. User 1 patches transaction with notes and category -> 200 OK
	categoryIDStr := "11111111-2222-3333-4444-555555555555"
	{
		patchReq := transactions.PatchTransactionRequest{
			UserCategoryID: &categoryIDStr,
			UserNotes:      stringPtr("Weekly grocery shopping"),
		}
		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest(http.MethodPatch, "/v1/transactions/"+tx1IDStr, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on patch, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var resp struct {
			Transaction transactions.TransactionResponse `json:"transaction"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode patch response: %v", err)
		}
		if !resp.Transaction.UserModified {
			t.Errorf("expected user_modified to be true")
		}
		if resp.Transaction.UserNotes == nil || *resp.Transaction.UserNotes != "Weekly grocery shopping" {
			t.Errorf("expected updated user_notes, got %v", resp.Transaction.UserNotes)
		}
		if resp.Transaction.UserCategoryID == nil || *resp.Transaction.UserCategoryID != categoryIDStr {
			t.Errorf("expected updated user_category_id, got %v", resp.Transaction.UserCategoryID)
		}
	}

	// 4. User 2 attempts to patch User 1's transaction -> 404 Not Found (tenant isolation)
	{
		patchReq := transactions.PatchTransactionRequest{
			UserNotes: stringPtr("Malicious note overwrite"),
		}
		body, _ := json.Marshal(patchReq)
		req := httptest.NewRequest(http.MethodPatch, "/v1/transactions/"+tx1IDStr, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token2)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for cross-tenant patch, got %d", rec.Code)
		}
	}

	// 5. Unauthenticated request -> 401 Unauthorized
	{
		req := httptest.NewRequest(http.MethodGet, "/v1/transactions/"+tx1IDStr, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	}
}
