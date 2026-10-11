package parsers_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Danikarat/tally/internal/parsers"
)

type mockParser struct {
	provider    string
	canParseFn  func(sender, body string) bool
	parseFn     func(sender, body string) (parsers.TransactionCandidate, error)
}

func (m *mockParser) Provider() string {
	return m.provider
}

func (m *mockParser) CanParse(sender, body string) bool {
	if m.canParseFn != nil {
		return m.canParseFn(sender, body)
	}
	return strings.EqualFold(sender, m.provider)
}

func (m *mockParser) Parse(sender, body string) (parsers.TransactionCandidate, error) {
	if m.parseFn != nil {
		return m.parseFn(sender, body)
	}
	return parsers.TransactionCandidate{
		Provider:         m.provider,
		AmountMinorUnits: 1000,
		Currency:         "GHS",
		Direction:        "OUTBOUND",
		TransactionType:  "EXPENSE",
		EventTimestamp:   time.Now().UTC(),
	}, nil
}

func TestRegistry_RegistrationAndDispatch(t *testing.T) {
	reg := parsers.NewRegistry()

	mtnParser := &mockParser{
		provider: "MTN",
		canParseFn: func(sender, body string) bool {
			return strings.EqualFold(sender, "MobileMoney") || strings.EqualFold(sender, "MTN")
		},
		parseFn: func(sender, body string) (parsers.TransactionCandidate, error) {
			if strings.Contains(body, "Payment of") {
				return parsers.TransactionCandidate{
					Provider:         "MTN",
					SourceType:       "SMS",
					Sender:           sender,
					AmountMinorUnits: 5000,
					Currency:         "GHS",
					Direction:        "OUTBOUND",
					TransactionType:  "EXPENSE",
					EventTimestamp:   time.Now().UTC(),
					RawPayload:       body,
				}, nil
			}
			return parsers.TransactionCandidate{}, parsers.ErrUnrecognizedFormat
		},
	}

	telecelParser := &mockParser{
		provider: "TELECEL",
		canParseFn: func(sender, body string) bool {
			return strings.EqualFold(sender, "TelecelCash")
		},
		parseFn: func(sender, body string) (parsers.TransactionCandidate, error) {
			return parsers.TransactionCandidate{
				Provider:         "TELECEL",
				SourceType:       "SMS",
				Sender:           sender,
				AmountMinorUnits: 7500,
				Currency:         "GHS",
				Direction:        "OUTBOUND",
				TransactionType:  "EXPENSE",
				EventTimestamp:   time.Now().UTC(),
				RawPayload:       body,
			}, nil
		},
	}

	reg.Register(mtnParser)
	reg.Register(telecelParser)

	// 1. Dispatch to MTN parser
	resMTN, err := reg.Parse("MobileMoney", "Payment of GHS 50.00 completed.")
	if err != nil {
		t.Fatalf("expected successful MTN parse, got error: %v", err)
	}
	if resMTN.Provider != "MTN" || resMTN.AmountMinorUnits != 5000 {
		t.Errorf("unexpected MTN candidate: %+v", resMTN)
	}

	// 2. Dispatch to Telecel parser
	resTelecel, err := reg.Parse("TelecelCash", "Cash Out of GHS 75.00 successful.")
	if err != nil {
		t.Fatalf("expected successful Telecel parse, got error: %v", err)
	}
	if resTelecel.Provider != "TELECEL" || resTelecel.AmountMinorUnits != 7500 {
		t.Errorf("unexpected Telecel candidate: %+v", resTelecel)
	}

	// 3. Unsupported sender returns ErrUnsupportedProvider
	_, err = reg.Parse("UnknownSender", "Some random text")
	if !errors.Is(err, parsers.ErrUnsupportedProvider) {
		t.Errorf("expected ErrUnsupportedProvider, got: %v", err)
	}

	// 4. Recognized provider with malformed body returns ErrUnrecognizedFormat
	_, err = reg.Parse("MobileMoney", "Malformed unparseable text without keywords")
	if !errors.Is(err, parsers.ErrUnrecognizedFormat) {
		t.Errorf("expected ErrUnrecognizedFormat, got: %v", err)
	}
}

func TestRegistry_ListProviders(t *testing.T) {
	reg := parsers.NewRegistry()
	reg.Register(&mockParser{provider: "MTN"})
	reg.Register(&mockParser{provider: "TELECEL"})
	reg.Register(&mockParser{provider: "AT_MONEY"})

	providers := reg.ListProviders()
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providers))
	}

	foundMTN, foundTelecel, foundAT := false, false, false
	for _, p := range providers {
		switch p {
		case "MTN":
			foundMTN = true
		case "TELECEL":
			foundTelecel = true
		case "AT_MONEY":
			foundAT = true
		}
	}

	if !foundMTN || !foundTelecel || !foundAT {
		t.Errorf("expected MTN, TELECEL, and AT_MONEY, got %v", providers)
	}
}

func TestRegistry_ConcurrentOperations(t *testing.T) {
	reg := parsers.NewRegistry()

	// Pre-register one parser
	reg.Register(&mockParser{
		provider: "MTN",
		canParseFn: func(sender, body string) bool {
			return sender == "MobileMoney"
		},
		parseFn: func(sender, body string) (parsers.TransactionCandidate, error) {
			return parsers.TransactionCandidate{
				Provider:         "MTN",
				AmountMinorUnits: 2500,
			}, nil
		},
	})

	const goroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			if id%3 == 0 {
				// Concurrently register a new parser
				reg.Register(&mockParser{
					provider: "PROVIDER_" + string(rune('A'+id)),
				})
			} else {
				// Concurrently parse
				cand, err := reg.Parse("MobileMoney", "Payment body")
				if err != nil {
					t.Errorf("unexpected error in goroutine %d: %v", id, err)
				}
				if cand.AmountMinorUnits != 2500 {
					t.Errorf("unexpected amount in goroutine %d: %d", id, cand.AmountMinorUnits)
				}
			}
		}(i)
	}

	wg.Wait()
}
