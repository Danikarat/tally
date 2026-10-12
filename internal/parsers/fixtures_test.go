package parsers_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type FixtureExpected struct {
	Provider               string  `json:"provider"`
	AmountMinorUnits       int64   `json:"amount_minor_units"`
	Currency               string  `json:"currency"`
	Direction              string  `json:"direction"`
	TransactionType        string  `json:"transaction_type"`
	Counterparty           *string `json:"counterparty"`
	CounterpartyIdentifier *string `json:"counterparty_identifier"`
	ProviderReference      *string `json:"provider_reference"`
	BalanceMinorUnits      *int64  `json:"balance_minor_units"`
	FeeMinorUnits          *int64  `json:"fee_minor_units"`
}

type Fixture struct {
	ID       string          `json:"id"`
	Sender   string          `json:"sender"`
	RawBody  string          `json:"raw_body"`
	Expected FixtureExpected `json:"expected"`
}

func TestMTNFixtures_Validation(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "testdata", "fixtures", "mtn")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("failed to read mtn fixture directory: %v", err)
	}

	validDirections := map[string]bool{
		"INBOUND":  true,
		"OUTBOUND": true,
	}

	validTransactionTypes := map[string]bool{
		"EXPENSE":         true,
		"INCOME":          true,
		"TRANSFER":        true,
		"REFUND":          true,
		"FEE":             true,
		"CASH_WITHDRAWAL": true,
		"UNKNOWN":         true,
	}

	fixtureCount := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		fixtureCount++
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(fixtureDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read fixture file %s: %v", entry.Name(), err)
			}

			var f Fixture
			if err := json.Unmarshal(data, &f); err != nil {
				t.Fatalf("failed to parse fixture json %s: %v", entry.Name(), err)
			}

			if f.ID == "" {
				t.Errorf("fixture %s: id must not be empty", entry.Name())
			}
			if f.Sender == "" {
				t.Errorf("fixture %s: sender must not be empty", entry.Name())
			}
			if f.RawBody == "" {
				t.Errorf("fixture %s: raw_body must not be empty", entry.Name())
			}

			if f.Expected.Provider != "MTN" {
				t.Errorf("fixture %s: expected provider 'MTN', got '%s'", entry.Name(), f.Expected.Provider)
			}
			if f.Expected.Currency != "GHS" {
				t.Errorf("fixture %s: expected currency 'GHS', got '%s'", entry.Name(), f.Expected.Currency)
			}
			if f.Expected.AmountMinorUnits <= 0 {
				t.Errorf("fixture %s: amount_minor_units must be > 0, got %d", entry.Name(), f.Expected.AmountMinorUnits)
			}
			if !validDirections[f.Expected.Direction] {
				t.Errorf("fixture %s: invalid direction '%s'", entry.Name(), f.Expected.Direction)
			}
			if !validTransactionTypes[f.Expected.TransactionType] {
				t.Errorf("fixture %s: invalid transaction_type '%s'", entry.Name(), f.Expected.TransactionType)
			}
			if f.Expected.ProviderReference == nil || *f.Expected.ProviderReference == "" {
				t.Errorf("fixture %s: provider_reference must be non-empty", entry.Name())
			}
			if f.Expected.BalanceMinorUnits == nil {
				t.Errorf("fixture %s: balance_minor_units must not be nil", entry.Name())
			}
			if f.Expected.FeeMinorUnits == nil {
				t.Errorf("fixture %s: fee_minor_units must not be nil", entry.Name())
			}
			if f.Expected.Counterparty == nil || *f.Expected.Counterparty == "" {
				t.Errorf("fixture %s: counterparty must not be empty", entry.Name())
			}
		})
	}

	if fixtureCount < 5 {
		t.Errorf("expected at least 5 MTN fixtures covering core types, found %d", fixtureCount)
	}
}
