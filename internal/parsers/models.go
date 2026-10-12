package parsers

import (
	"errors"
	"time"
)

// Provider represents a canonical financial service provider.
type Provider string

const (
	ProviderMTN     Provider = "MTN"
	ProviderTelecel Provider = "TELECEL"
	ProviderATMoney Provider = "AT_MONEY"
	ProviderUnknown Provider = "UNKNOWN"
)

// String returns the string representation of the provider.
func (p Provider) String() string {
	return string(p)
}

// Common source types
const (
	SourceTypeSMS       = "SMS"
	SourceTypeStatement = "STATEMENT"
	SourceTypeEmail     = "EMAIL"
	SourceTypeManual    = "MANUAL"
)

// Common transaction directions
const (
	DirectionInbound  = "INBOUND"
	DirectionOutbound = "OUTBOUND"
)

// Common transaction types
const (
	TxTypeExpense        = "EXPENSE"
	TxTypeIncome         = "INCOME"
	TxTypeTransfer       = "TRANSFER"
	TxTypeRefund         = "REFUND"
	TxTypeFee            = "FEE"
	TxTypeCashWithdrawal = "CASH_WITHDRAWAL"
	TxTypeUnknown        = "UNKNOWN"
)

// Common transaction statuses
const (
	TxStatusCompleted = "COMPLETED"
	TxStatusPending   = "PENDING"
	TxStatusFailed    = "FAILED"
	TxStatusReversed  = "REVERSED"
)

var (
	// ErrUnsupportedProvider is returned when no registered parser recognizes the sender identifier.
	ErrUnsupportedProvider = errors.New("unsupported sender or provider")

	// ErrUnrecognizedFormat is returned when a parser recognizes the provider but the message text does not match expected patterns.
	ErrUnrecognizedFormat = errors.New("unrecognized message format for provider")

	// ErrMalformedMessage is returned when a message contains recognizable keywords but is truncated or missing critical fields.
	ErrMalformedMessage = errors.New("malformed or truncated message")

	// ErrNonFinancialMessage is returned when a message from a financial provider is determined to be non-financial (e.g. promo, OTP, system alert).
	ErrNonFinancialMessage = errors.New("message is non-financial")
)

// TransactionCandidate represents the normalized extraction result from a provider message.
type TransactionCandidate struct {
	Provider               Provider  `json:"provider"`
	SourceType             string    `json:"source_type"`
	Sender                 string    `json:"sender"`
	Status                 string    `json:"status"`
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
