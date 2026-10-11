package parsers

import (
	"errors"
	"time"
)

var (
	// ErrUnsupportedProvider is returned when no registered parser recognizes the sender identifier.
	ErrUnsupportedProvider = errors.New("unsupported sender or provider")

	// ErrUnrecognizedFormat is returned when a parser recognizes the provider but the message text does not match expected patterns.
	ErrUnrecognizedFormat = errors.New("unrecognized message format for provider")

	// ErrNonFinancialMessage is returned when a message from a financial provider is determined to be non-financial (e.g. promo, OTP, system alert).
	ErrNonFinancialMessage = errors.New("message is non-financial")
)

// TransactionCandidate represents the normalized extraction result from a provider message.
type TransactionCandidate struct {
	Provider               string    `json:"provider"`
	SourceType             string    `json:"source_type"`
	Sender                 string    `json:"sender"`
	AmountMinorUnits       int64     `json:"amount_minor_units"`
	Currency               string    `json:"currency"`
	Direction              string    `json:"direction"`
	TransactionType        string    `json:"transaction_type"`
	Counterparty           *string   `json:"counterparty,omitempty"`
	CounterpartyIdentifier *string   `json:"counterparty_identifier,omitempty"`
	ProviderReference      *string   `json:"provider_reference,omitempty"`
	BalanceMinorUnits      *int64    `json:"balance_minor_units,omitempty"`
	FeeMinorUnits          *int64    `json:"fee_minor_units,omitempty"`
	EventTimestamp         time.Time `json:"event_timestamp"`
	RawPayload             string    `json:"raw_payload"`
	PayloadHash            string    `json:"payload_hash"`
}
