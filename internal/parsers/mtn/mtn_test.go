package mtn_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Danikarat/tally/internal/parsers"
	"github.com/Danikarat/tally/internal/parsers/mtn"
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

func TestMtnMomoParser_Fixtures(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "testdata", "fixtures", "mtn")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("failed to read mtn fixture directory: %v", err)
	}

	parser := mtn.New()
	if parser.Provider() != "MTN" {
		t.Fatalf("expected provider 'MTN', got %q", parser.Provider())
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

			if !parser.CanParse(f.Sender, f.RawBody) {
				t.Errorf("parser.CanParse returned false for fixture %s", entry.Name())
			}

			cand, err := parser.Parse(f.Sender, f.RawBody)
			if err != nil {
				t.Fatalf("parser.Parse failed for fixture %s: %v", entry.Name(), err)
			}

			if cand.Provider != f.Expected.Provider {
				t.Errorf("Provider: got %q, want %q", cand.Provider, f.Expected.Provider)
			}
			if cand.Currency != f.Expected.Currency {
				t.Errorf("Currency: got %q, want %q", cand.Currency, f.Expected.Currency)
			}
			if cand.AmountMinorUnits != f.Expected.AmountMinorUnits {
				t.Errorf("AmountMinorUnits: got %d, want %d", cand.AmountMinorUnits, f.Expected.AmountMinorUnits)
			}
			if cand.Direction != f.Expected.Direction {
				t.Errorf("Direction: got %q, want %q", cand.Direction, f.Expected.Direction)
			}
			if cand.TransactionType != f.Expected.TransactionType {
				t.Errorf("TransactionType: got %q, want %q", cand.TransactionType, f.Expected.TransactionType)
			}

			// Provider reference
			if f.Expected.ProviderReference != nil {
				if cand.ProviderReference == nil {
					t.Errorf("ProviderReference: got nil, want %q", *f.Expected.ProviderReference)
				} else if *cand.ProviderReference != *f.Expected.ProviderReference {
					t.Errorf("ProviderReference: got %q, want %q", *cand.ProviderReference, *f.Expected.ProviderReference)
				}
			}

			// Counterparty
			if f.Expected.Counterparty != nil {
				if cand.Counterparty == nil {
					t.Errorf("Counterparty: got nil, want %q", *f.Expected.Counterparty)
				} else if *cand.Counterparty != *f.Expected.Counterparty {
					t.Errorf("Counterparty: got %q, want %q", *cand.Counterparty, *f.Expected.Counterparty)
				}
			}

			// Counterparty Identifier
			if f.Expected.CounterpartyIdentifier != nil {
				if cand.CounterpartyIdentifier == nil {
					t.Errorf("CounterpartyIdentifier: got nil, want %q", *f.Expected.CounterpartyIdentifier)
				} else if *cand.CounterpartyIdentifier != *f.Expected.CounterpartyIdentifier {
					t.Errorf("CounterpartyIdentifier: got %q, want %q", *cand.CounterpartyIdentifier, *f.Expected.CounterpartyIdentifier)
				}
			}

			// Balance
			if f.Expected.BalanceMinorUnits != nil {
				if cand.BalanceMinorUnits == nil {
					t.Errorf("BalanceMinorUnits: got nil, want %d", *f.Expected.BalanceMinorUnits)
				} else if *cand.BalanceMinorUnits != *f.Expected.BalanceMinorUnits {
					t.Errorf("BalanceMinorUnits: got %d, want %d", *cand.BalanceMinorUnits, *f.Expected.BalanceMinorUnits)
				}
			}

			// Fee
			if f.Expected.FeeMinorUnits != nil {
				if cand.FeeMinorUnits == nil {
					t.Errorf("FeeMinorUnits: got nil, want %d", *f.Expected.FeeMinorUnits)
				} else if *cand.FeeMinorUnits != *f.Expected.FeeMinorUnits {
					t.Errorf("FeeMinorUnits: got %d, want %d", *cand.FeeMinorUnits, *f.Expected.FeeMinorUnits)
				}
			}

			if cand.PayloadHash == "" {
				t.Errorf("PayloadHash must not be empty")
			}
			if cand.RawPayload != f.RawBody {
				t.Errorf("RawPayload: got %q, want %q", cand.RawPayload, f.RawBody)
			}
		})
	}

	if fixtureCount < 6 {
		t.Errorf("expected at least 6 fixtures tested, got %d", fixtureCount)
	}
}

func TestMtnMomoParser_CanParse(t *testing.T) {
	p := mtn.New()

	validCases := []struct {
		sender string
		body   string
	}{
		{"MobileMoney", "Any text"},
		{"mobilemoney", "Any text"},
		{"MTN", "Any text"},
		{"mtn", "Any text"},
		{"MTNMoMo", "Any text"},
		{"momo", "Any text"},
		{"Unknown", "Visit mtnmymomo.onelink.me for info"},
		{"Alert", "MTN Mobile Money transaction alert"},
	}

	for _, tc := range validCases {
		if !p.CanParse(tc.sender, tc.body) {
			t.Errorf("expected CanParse=true for sender=%q body=%q", tc.sender, tc.body)
		}
	}

	invalidCases := []struct {
		sender string
		body   string
	}{
		{"TelecelCash", "Vodafone cash notification"},
		{"ATMoney", "AirtelTigo Money notification"},
		{"BankAlert", "Your account was debited"},
		{"Google", "Your security code is 123456"},
	}

	for _, tc := range invalidCases {
		if p.CanParse(tc.sender, tc.body) {
			t.Errorf("expected CanParse=false for sender=%q body=%q", tc.sender, tc.body)
		}
	}
}

func TestMtnMomoParser_NonFinancialMessages(t *testing.T) {
	p := mtn.New()

	nonFinancialMessages := []string{
		"Your one-time password is 123456. Do not share with anyone.",
		"Your OTP is 987654. Valid for 5 minutes.",
		"Enter PIN to approve transaction.",
		"Win with MTN MoMo! Dial *170# now.",
		"Y'ello! Enjoy 50% extra bonus on your next recharge.",
	}

	for _, msg := range nonFinancialMessages {
		_, err := p.Parse("MobileMoney", msg)
		if !errors.Is(err, parsers.ErrNonFinancialMessage) {
			t.Errorf("expected ErrNonFinancialMessage for body %q, got: %v", msg, err)
		}
	}
}

func TestMtnMomoParser_UnrecognizedFormat(t *testing.T) {
	p := mtn.New()

	malformedMessages := []string{
		"",
		"   ",
		"Random text message with no financial data or keywords whatsoever",
		"Payment completed but without any amounts or IDs",
		"Payment for GHS to Bills.INV. Current Balance: GHS. Transaction Id: 123.",
	}

	for _, msg := range malformedMessages {
		_, err := p.Parse("MobileMoney", msg)
		if !errors.Is(err, parsers.ErrUnrecognizedFormat) {
			t.Errorf("expected ErrUnrecognizedFormat for body %q, got: %v", msg, err)
		}
	}
}

func TestMtnMomoParser_RegistryIntegration(t *testing.T) {
	// The init() hook automatically registers mtn.New() with parsers.DefaultRegistry()
	cand, err := parsers.ParseDefault("MobileMoney", "Payment made for GHS 35.00 to KOFI MENSAH Current Balance: GHS 22.91 . Available Balance: GHS 22.91. Reference: Lunch. Transaction ID: 91383000001. Fee charged: GHS0.50 Tax charged: 0. Download the MoMo App for a Faster & Easier Experience. Click here: https://mtnmymomo.onelink.me/XJOt/MoMo")
	if err != nil {
		t.Fatalf("expected ParseDefault to succeed, got: %v", err)
	}

	if cand.Provider != "MTN" {
		t.Errorf("expected Provider 'MTN', got %q", cand.Provider)
	}
	if cand.AmountMinorUnits != 3500 {
		t.Errorf("expected AmountMinorUnits 3500, got %d", cand.AmountMinorUnits)
	}
}
