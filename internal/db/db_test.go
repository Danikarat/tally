package db_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Danikarat/tally/internal/config"
	"github.com/Danikarat/tally/internal/database"
	"github.com/Danikarat/tally/internal/db"
)

func TestSQLC_CRUD_AndConstraints(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	adminPool, err := database.NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		t.Skipf("skipping test: database not reachable at %s (%v)", cfg.DB.DSN(), err)
		return
	}
	defer adminPool.Close()

	testDBName := "tally_test_sqlc_queries"
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

	testPool, err := database.NewPool(ctx, testDSN)
	if err != nil {
		t.Fatalf("failed to connect to isolated test database: %v", err)
	}
	defer testPool.Close()

	// Apply migrations
	if err := database.MigrateUp(testDSN); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	queries := db.New(testPool)

	// 1. User Queries
	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        "kofi.mensah@example.com",
		PasswordHash: "hashed_argon2id_secret",
		FullName:     db.Text("Kofi Mensah"),
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Email != "kofi.mensah@example.com" {
		t.Errorf("expected email kofi.mensah@example.com, got %s", user.Email)
	}
	if !user.IsActive {
		t.Errorf("expected user to be active by default")
	}

	byEmail, err := queries.GetUserByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if byEmail.ID != user.ID {
		t.Errorf("expected user ID %v, got %v", user.ID, byEmail.ID)
	}

	byID, err := queries.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if byID.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, byID.Email)
	}

	updatedUser, err := queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:       user.ID,
		FullName: db.Text("Kofi Osei Mensah"),
	})
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if updatedUser.FullName.String != "Kofi Osei Mensah" {
		t.Errorf("expected updated full name, got %s", updatedUser.FullName.String)
	}

	// 2. Financial Account Queries
	account, err := queries.CreateFinancialAccount(ctx, db.CreateFinancialAccountParams{
		UserID:                  user.ID,
		AccountType:             "MOMO",
		ProviderName:            "MTN MoMo",
		AccountIdentifierMasked: db.Text("024****123"),
		Currency:                "GHS",
		BalanceMinorUnits:       25000, // 250.00 GHS in pesewas
		IsDefault:               true,
		IsActive:                true,
	})
	if err != nil {
		t.Fatalf("CreateFinancialAccount failed: %v", err)
	}
	if account.BalanceMinorUnits != 25000 {
		t.Errorf("expected balance 25000, got %d", account.BalanceMinorUnits)
	}
	if !account.IsDefault {
		t.Errorf("expected account to be default")
	}

	accByID, err := queries.GetFinancialAccountByID(ctx, db.GetFinancialAccountByIDParams{
		ID:     account.ID,
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("GetFinancialAccountByID failed: %v", err)
	}
	if accByID.ProviderName != "MTN MoMo" {
		t.Errorf("expected provider MTN MoMo, got %s", accByID.ProviderName)
	}

	defAcc, err := queries.GetDefaultFinancialAccount(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetDefaultFinancialAccount failed: %v", err)
	}
	if defAcc.ID != account.ID {
		t.Errorf("expected default account ID %v, got %v", account.ID, defAcc.ID)
	}

	accounts, err := queries.ListFinancialAccountsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListFinancialAccountsByUser failed: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	accByProvider, err := queries.FindAccountByProviderAndMaskedIdentifier(ctx, db.FindAccountByProviderAndMaskedIdentifierParams{
		UserID:                  user.ID,
		ProviderName:            "MTN MoMo",
		AccountIdentifierMasked: db.Text("024****123"),
	})
	if err != nil {
		t.Fatalf("FindAccountByProviderAndMaskedIdentifier failed: %v", err)
	}
	if accByProvider.ID != account.ID {
		t.Errorf("expected found account ID %v, got %v", account.ID, accByProvider.ID)
	}

	updatedAcc, err := queries.UpdateAccountBalance(ctx, db.UpdateAccountBalanceParams{
		ID:                account.ID,
		UserID:            user.ID,
		BalanceMinorUnits: 30000,
	})
	if err != nil {
		t.Fatalf("UpdateAccountBalance failed: %v", err)
	}
	if updatedAcc.BalanceMinorUnits != 30000 {
		t.Errorf("expected updated balance 30000, got %d", updatedAcc.BalanceMinorUnits)
	}

	// 3. Source Events Queries
	receivedTime := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)
	srcEvent, err := queries.InsertSourceEvent(ctx, db.InsertSourceEventParams{
		UserID:            user.ID,
		SourceType:        "SMS",
		Provider:          "MTN",
		SenderIdentifier:  db.Text("MobileMoney"),
		ExternalReference: db.Text("1234567890"),
		PayloadHash:       "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		RawPayload:        db.Text("Payment of GHS 50.00 to Melcom completed. Current balance: GHS 250.00."),
		ParseStatus:       "PENDING",
		ReceivedAt:        db.Timestamptz(receivedTime),
	})
	if err != nil {
		t.Fatalf("InsertSourceEvent failed: %v", err)
	}
	if srcEvent.ParseStatus != "PENDING" {
		t.Errorf("expected parse_status PENDING, got %s", srcEvent.ParseStatus)
	}

	eventByHash, err := queries.FindSourceEventByHash(ctx, db.FindSourceEventByHashParams{
		UserID:      user.ID,
		PayloadHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	})
	if err != nil {
		t.Fatalf("FindSourceEventByHash failed: %v", err)
	}
	if eventByHash.ID != srcEvent.ID {
		t.Errorf("expected event ID %v, got %v", srcEvent.ID, eventByHash.ID)
	}

	eventByID, err := queries.GetSourceEventByID(ctx, db.GetSourceEventByIDParams{
		ID:     srcEvent.ID,
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("GetSourceEventByID failed: %v", err)
	}
	if eventByID.ID != srcEvent.ID {
		t.Errorf("expected event ID %v, got %v", srcEvent.ID, eventByID.ID)
	}

	updatedEvent, err := queries.UpdateSourceEventStatus(ctx, db.UpdateSourceEventStatusParams{
		ID:          srcEvent.ID,
		UserID:      user.ID,
		ParseStatus: "PARSED",
	})
	if err != nil {
		t.Fatalf("UpdateSourceEventStatus failed: %v", err)
	}
	if updatedEvent.ParseStatus != "PARSED" {
		t.Errorf("expected parse_status PARSED, got %s", updatedEvent.ParseStatus)
	}

	eventList, err := queries.ListSourceEventsByUser(ctx, db.ListSourceEventsByUserParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListSourceEventsByUser failed: %v", err)
	}
	if len(eventList) != 1 {
		t.Errorf("expected 1 source event in list, got %d", len(eventList))
	}

	// Verify duplicate payload_hash constraint violation
	_, err = queries.InsertSourceEvent(ctx, db.InsertSourceEventParams{
		UserID:            user.ID,
		SourceType:        "SMS",
		Provider:          "MTN",
		SenderIdentifier:  db.Text("MobileMoney"),
		ExternalReference: db.Text("1234567890"),
		PayloadHash:       "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		RawPayload:        db.Text("Duplicate SMS body"),
		ParseStatus:       "PENDING",
		ReceivedAt:        db.Timestamptz(receivedTime),
	})
	if err == nil {
		t.Fatalf("expected duplicate payload_hash insert to fail, but it succeeded")
	}

	// 4. Transaction Queries
	eventTimestamp := time.Date(2026, 3, 15, 10, 29, 30, 0, time.UTC)
	txRecord, err := queries.InsertTransaction(ctx, db.InsertTransactionParams{
		UserID:                 user.ID,
		AccountID:              account.ID,
		SourceEventID:          srcEvent.ID,
		AmountMinorUnits:       5000, // 50.00 GHS in pesewas
		Currency:               "GHS",
		Direction:              "OUTBOUND",
		TransactionType:        "EXPENSE",
		Counterparty:           db.Text("Melcom"),
		CounterpartyIdentifier: db.Text("MELCOM-ACCRA"),
		ProviderReference:      db.Text("REF-MOMO-987654"),
		Status:                 "COMPLETED",
		UserCategoryID:         db.NullUUID(),
		UserNotes:              db.Text("Groceries shopping"),
		UserModified:           false,
		EventTimestamp:         db.Timestamptz(eventTimestamp),
	})
	if err != nil {
		t.Fatalf("InsertTransaction failed: %v", err)
	}
	if txRecord.AmountMinorUnits != 5000 {
		t.Errorf("expected amount 5000, got %d", txRecord.AmountMinorUnits)
	}
	if txRecord.Currency != "GHS" {
		t.Errorf("expected currency GHS, got %s", txRecord.Currency)
	}

	txByRef, err := queries.FindTransactionByReference(ctx, db.FindTransactionByReferenceParams{
		UserID:            user.ID,
		ProviderReference: db.Text("REF-MOMO-987654"),
	})
	if err != nil {
		t.Fatalf("FindTransactionByReference failed: %v", err)
	}
	if txByRef.ID != txRecord.ID {
		t.Errorf("expected tx ID %v, got %v", txRecord.ID, txByRef.ID)
	}

	txByID, err := queries.GetTransactionByID(ctx, db.GetTransactionByIDParams{
		ID:     txRecord.ID,
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("GetTransactionByID failed: %v", err)
	}
	if txByID.ID != txRecord.ID {
		t.Errorf("expected tx ID %v, got %v", txRecord.ID, txByID.ID)
	}

	txList, err := queries.ListTransactionsByUser(ctx, db.ListTransactionsByUserParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListTransactionsByUser failed: %v", err)
	}
	if len(txList) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txList))
	}
	if txList[0].AmountMinorUnits != 5000 {
		t.Errorf("expected amount 5000, got %d", txList[0].AmountMinorUnits)
	}

	txByAcc, err := queries.ListTransactionsByAccount(ctx, db.ListTransactionsByAccountParams{
		UserID:    user.ID,
		AccountID: account.ID,
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("ListTransactionsByAccount failed: %v", err)
	}
	if len(txByAcc) != 1 {
		t.Errorf("expected 1 transaction for account, got %d", len(txByAcc))
	}

	updatedTx, err := queries.UpdateTransactionStatus(ctx, db.UpdateTransactionStatusParams{
		ID:     txRecord.ID,
		UserID: user.ID,
		Status: "REVERSED",
	})
	if err != nil {
		t.Fatalf("UpdateTransactionStatus failed: %v", err)
	}
	if updatedTx.Status != "REVERSED" {
		t.Errorf("expected status REVERSED, got %s", updatedTx.Status)
	}

	// Verify check constraint on amount_minor_units >= 0
	_, err = queries.InsertTransaction(ctx, db.InsertTransactionParams{
		UserID:                 user.ID,
		AccountID:              account.ID,
		SourceEventID:          srcEvent.ID,
		AmountMinorUnits:       -100, // Invalid negative amount
		Currency:               "GHS",
		Direction:              "OUTBOUND",
		TransactionType:        "EXPENSE",
		Counterparty:           db.Text("Bad Merchant"),
		CounterpartyIdentifier: db.NullText(),
		ProviderReference:      db.Text("REF-NEGATIVE-TEST"),
		Status:                 "COMPLETED",
		UserCategoryID:         db.NullUUID(),
		UserNotes:              db.NullText(),
		UserModified:           false,
		EventTimestamp:         db.Timestamptz(eventTimestamp),
	})
	if err == nil {
		t.Fatalf("expected negative amount transaction insert to fail constraint, but it succeeded")
	}

	// Verify check constraint on direction
	_, err = queries.InsertTransaction(ctx, db.InsertTransactionParams{
		UserID:                 user.ID,
		AccountID:              account.ID,
		SourceEventID:          srcEvent.ID,
		AmountMinorUnits:       1000,
		Currency:               "GHS",
		Direction:              "SIDEWAYS", // Invalid direction
		TransactionType:        "EXPENSE",
		Counterparty:           db.Text("Merchant"),
		CounterpartyIdentifier: db.NullText(),
		ProviderReference:      db.Text("REF-DIRECTION-TEST"),
		Status:                 "COMPLETED",
		UserCategoryID:         db.NullUUID(),
		UserNotes:              db.NullText(),
		UserModified:           false,
		EventTimestamp:         db.Timestamptz(eventTimestamp),
	})
	if err == nil {
		t.Fatalf("expected invalid direction insert to fail constraint, but it succeeded")
	}
}
