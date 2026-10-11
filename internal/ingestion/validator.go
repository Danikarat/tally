package ingestion

import (
	"errors"
	"strings"
)

var (
	ErrEmptyBatch     = errors.New("candidates array cannot be empty")
	ErrBatchTooLarge  = errors.New("batch size cannot exceed 100 candidates")
	ErrValidationFail = errors.New("validation failed")
)

var validSourceTypes = map[string]bool{
	SourceTypeSMS:       true,
	SourceTypeStatement: true,
	SourceTypeEmail:     true,
	SourceTypeManual:    true,
}

var validDirections = map[string]bool{
	DirectionInbound:  true,
	DirectionOutbound: true,
}

var validTransactionTypes = map[string]bool{
	TxTypeExpense:        true,
	TxTypeIncome:         true,
	TxTypeTransfer:       true,
	TxTypeRefund:         true,
	TxTypeFee:            true,
	TxTypeCashWithdrawal: true,
	TxTypeUnknown:        true,
}

// ValidateIngestRequest checks the request and all candidates for semantic validity.
func ValidateIngestRequest(req IngestRequest) ([]ValidationErrorDetail, error) {
	if len(req.Candidates) == 0 {
		return nil, ErrEmptyBatch
	}
	if len(req.Candidates) > 100 {
		return nil, ErrBatchTooLarge
	}

	var details []ValidationErrorDetail

	for i, c := range req.Candidates {
		// Amount check (must be strictly positive integer minor units)
		if c.AmountMinorUnits <= 0 {
			details = append(details, ValidationErrorDetail{
				Index:   i,
				Field:   "amount_minor_units",
				Message: "amount_minor_units must be greater than zero",
			})
		}

		// Currency check (ISO 4217, 3 characters)
		trimmedCurrency := strings.TrimSpace(c.Currency)
		if len(trimmedCurrency) != 3 {
			details = append(details, ValidationErrorDetail{
				Index:   i,
				Field:   "currency",
				Message: "currency must be a 3-character ISO code (e.g. GHS)",
			})
		}

		// Direction check
		if !validDirections[strings.ToUpper(strings.TrimSpace(c.Direction))] {
			details = append(details, ValidationErrorDetail{
				Index:   i,
				Field:   "direction",
				Message: "direction must be INBOUND or OUTBOUND",
			})
		}

		// Transaction type check
		if !validTransactionTypes[strings.ToUpper(strings.TrimSpace(c.TransactionType))] {
			details = append(details, ValidationErrorDetail{
				Index:   i,
				Field:   "transaction_type",
				Message: "transaction_type must be one of EXPENSE, INCOME, TRANSFER, REFUND, FEE, CASH_WITHDRAWAL, UNKNOWN",
			})
		}

		// Source type check
		if !validSourceTypes[strings.ToUpper(strings.TrimSpace(c.SourceType))] {
			details = append(details, ValidationErrorDetail{
				Index:   i,
				Field:   "source_type",
				Message: "source_type must be one of SMS, STATEMENT, EMAIL, MANUAL",
			})
		}

		// Provider check
		if strings.TrimSpace(c.Provider) == "" {
			details = append(details, ValidationErrorDetail{
				Index:   i,
				Field:   "provider",
				Message: "provider is required",
			})
		}

		// Event timestamp check
		if c.EventTimestamp.IsZero() {
			details = append(details, ValidationErrorDetail{
				Index:   i,
				Field:   "event_timestamp",
				Message: "event_timestamp must be a valid non-zero timestamp",
			})
		}
	}

	if len(details) > 0 {
		return details, ErrValidationFail
	}

	return nil, nil
}
