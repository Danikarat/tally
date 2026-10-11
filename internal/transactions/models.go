package transactions

import (
	"errors"
	"fmt"
	"time"

	"github.com/Danikarat/tally/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInvalidUUID         = errors.New("invalid uuid format")
)

// TransactionResponse represents the public JSON structure of a transaction.
type TransactionResponse struct {
	ID                     string    `json:"id"`
	AccountID              string    `json:"account_id"`
	SourceEventID          *string   `json:"source_event_id,omitempty"`
	AmountMinorUnits       int64     `json:"amount_minor_units"`
	Currency               string    `json:"currency"`
	Direction              string    `json:"direction"`
	TransactionType        string    `json:"transaction_type"`
	Counterparty           *string   `json:"counterparty,omitempty"`
	CounterpartyIdentifier *string   `json:"counterparty_identifier,omitempty"`
	ProviderReference      *string   `json:"provider_reference,omitempty"`
	Status                 string    `json:"status"`
	UserCategoryID         *string   `json:"user_category_id,omitempty"`
	UserNotes              *string   `json:"user_notes,omitempty"`
	UserModified           bool      `json:"user_modified"`
	EventTimestamp         time.Time `json:"event_timestamp"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// PaginationMeta contains paging information.
type PaginationMeta struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
}

// ListTransactionsResponse represents a paginated list of transactions.
type ListTransactionsResponse struct {
	Data       []TransactionResponse `json:"data"`
	Pagination PaginationMeta        `json:"pagination"`
}

// PatchTransactionRequest holds optional fields for updating user classification and notes.
type PatchTransactionRequest struct {
	UserCategoryID *string `json:"user_category_id"`
	UserNotes      *string `json:"user_notes"`
}

// ListFilter encapsulates query filters for listing transactions.
type ListFilter struct {
	AccountID       *string
	FromDate        *time.Time
	ToDate          *time.Time
	TransactionType *string
	Limit           int
	Offset          int
}

// FromDBModel maps a database Transaction model to public TransactionResponse.
func FromDBModel(m db.Transaction) TransactionResponse {
	resp := TransactionResponse{
		ID:               uuidToString(m.ID),
		AccountID:        uuidToString(m.AccountID),
		AmountMinorUnits: m.AmountMinorUnits,
		Currency:         m.Currency,
		Direction:        m.Direction,
		TransactionType:  m.TransactionType,
		Status:           m.Status,
		UserModified:     m.UserModified,
		EventTimestamp:   m.EventTimestamp.Time.UTC(),
		CreatedAt:        m.CreatedAt.Time.UTC(),
		UpdatedAt:        m.UpdatedAt.Time.UTC(),
	}

	if m.SourceEventID.Valid {
		s := uuidToString(m.SourceEventID)
		resp.SourceEventID = &s
	}
	if m.Counterparty.Valid {
		resp.Counterparty = &m.Counterparty.String
	}
	if m.CounterpartyIdentifier.Valid {
		resp.CounterpartyIdentifier = &m.CounterpartyIdentifier.String
	}
	if m.ProviderReference.Valid {
		resp.ProviderReference = &m.ProviderReference.String
	}
	if m.UserCategoryID.Valid {
		s := uuidToString(m.UserCategoryID)
		resp.UserCategoryID = &s
	}
	if m.UserNotes.Valid {
		resp.UserNotes = &m.UserNotes.String
	}

	return resp
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	src := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		src[0:4], src[4:6], src[6:8], src[8:10], src[10:16])
}
