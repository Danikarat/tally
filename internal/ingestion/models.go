package ingestion

import "time"

// Source types
const (
	SourceTypeSMS       = "SMS"
	SourceTypeStatement = "STATEMENT"
	SourceTypeEmail     = "EMAIL"
	SourceTypeManual    = "MANUAL"
)

// Transaction directions
const (
	DirectionInbound  = "INBOUND"
	DirectionOutbound = "OUTBOUND"
)

// Transaction types
const (
	TxTypeExpense        = "EXPENSE"
	TxTypeIncome         = "INCOME"
	TxTypeTransfer       = "TRANSFER"
	TxTypeRefund         = "REFUND"
	TxTypeFee            = "FEE"
	TxTypeCashWithdrawal = "CASH_WITHDRAWAL"
	TxTypeUnknown        = "UNKNOWN"
)

// Item ingestion statuses
const (
	StatusIngested         = "INGESTED"
	StatusDuplicateIgnored = "DUPLICATE_IGNORED"
	StatusFailed           = "FAILED"
)

// TransactionCandidate represents a parsed transaction candidate submitted by the client.
type TransactionCandidate struct {
	SourceType             string    `json:"source_type"`
	Provider               string    `json:"provider"`
	SenderIdentifier       *string   `json:"sender_identifier,omitempty"`
	ExternalReference      *string   `json:"external_reference,omitempty"`
	PayloadHash            *string   `json:"payload_hash,omitempty"`
	RawPayload             *string   `json:"raw_payload,omitempty"`
	AccountID              *string   `json:"account_id,omitempty"`
	AmountMinorUnits       int64     `json:"amount_minor_units"`
	Currency               string    `json:"currency"`
	Direction              string    `json:"direction"`
	TransactionType        string    `json:"transaction_type"`
	Counterparty           *string   `json:"counterparty,omitempty"`
	CounterpartyIdentifier *string   `json:"counterparty_identifier,omitempty"`
	ProviderReference      *string   `json:"provider_reference,omitempty"`
	EventTimestamp         time.Time `json:"event_timestamp"`
}

// IngestRequest represents the payload for POST /v1/transactions/ingest.
type IngestRequest struct {
	Candidates []TransactionCandidate `json:"candidates"`
}

// IngestItemResult reports the ingestion outcome for an individual candidate in a batch.
type IngestItemResult struct {
	Index         int     `json:"index"`
	Status        string  `json:"status"` // INGESTED, DUPLICATE_IGNORED, FAILED
	TransactionID *string `json:"transaction_id,omitempty"`
	SourceEventID *string `json:"source_event_id,omitempty"`
	Error         *string `json:"error,omitempty"`
}

// IngestResponse represents the structured outcome of POST /v1/transactions/ingest.
type IngestResponse struct {
	TotalReceived  int                `json:"total_received"`
	IngestedCount  int                `json:"ingested_count"`
	DuplicateCount int                `json:"duplicate_count"`
	FailedCount    int                `json:"failed_count"`
	Items          []IngestItemResult `json:"items"`
}

// ValidationErrorDetail describes a specific field-level validation issue.
type ValidationErrorDetail struct {
	Index   int    `json:"index"`
	Field   string `json:"field"`
	Message string `json:"message"`
}
