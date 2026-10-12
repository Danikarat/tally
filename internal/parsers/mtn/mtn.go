package mtn

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Danikarat/tally/internal/parsers"
)

// FlowType defines the specific transaction flow identified in an MTN MoMo SMS.
type FlowType int

const (
	FlowUnknown FlowType = iota
	FlowTransferSent
	FlowPaymentReceived
	FlowCashInDeposit
	FlowPaymentExpense
	FlowCashOutWithdrawal
	FlowFailedTransaction
	FlowNonFinancial
	FlowMalformed
)

// MtnMomoParser implements parsers.ProviderAdapter for MTN MoMo Ghana.
type MtnMomoParser struct{}

func init() {
	parsers.RegisterDefault(New())
}

// New creates a new instance of MtnMomoParser.
func New() *MtnMomoParser {
	return &MtnMomoParser{}
}

// Provider returns the canonical identifier for MTN.
func (p *MtnMomoParser) Provider() parsers.Provider {
	return parsers.ProviderMTN
}

// CanParse checks if the sender and/or body indicate an MTN MoMo message.
func (p *MtnMomoParser) CanParse(sender, body string) bool {
	return parsers.DetectProvider(sender, body) == parsers.ProviderMTN
}

var (
	// Common regex patterns
	reTxID          = regexp.MustCompile(`(?i)(?:Financial\s+)?Transaction\s+Id:\s*([A-Za-z0-9]+)`)
	reBalance       = regexp.MustCompile(`(?i)Current\s+Balance:?\s*(?:GHS\s*)?([\d,]+\.?\d*)`)
	reFee           = regexp.MustCompile(`(?i)(?:Fee\s+charged|TRANSACTION\s+FEE|Fee\s+was):\s*(?:GHS\s*)?([\d,]+\.?\d*)`)
	rePhoneInRef    = regexp.MustCompile(`,\s*(\d{10,14})\b`)
	reTrailingPhone = regexp.MustCompile(`^(.*?)\s+(\d{10,14})$`)

	// Non-financial patterns (promos, balance inquiries, security OTPs)
	reNonFinancial = regexp.MustCompile(`(?i)(?:one[\s-]time\s+password|your\s+otp\s+is|enter\s+(?:mm\s+)?pin|dial\s+\*170#|win\s+with\s+mtn|y'ello!\s+enjoy|authorize\s+payment|do\s+not\s+share\s+your\s+(?:momo\s+)?pin|your\s+current\s+balance\s+is|dear\s+customer,?\s+your\s+balance\s+is|your\s+balance\s+is\s+ghs)`)

	// Balance inquiry with no transaction action
	rePureBalance = regexp.MustCompile(`(?i)^(?:Current\s+Balance:?\s*GHS|Your\s+balance\s+is|Dear\s+Customer).*?(?:Available\s+Balance|Balance\s+is)`)

	// Failed transaction pattern
	reFailed = regexp.MustCompile(`(?i)(?:transaction\s+(?:id:?\s*)?[A-Za-z0-9]+\s+failed|payment\s+.*?failed|failed\s+due\s+to|transaction\s+failed)`)

	// Known transaction prefixes (for detecting truncated / malformed messages)
	reKnownPrefix = regexp.MustCompile(`(?i)^(?:Cash\s+Out\s+made|Cash\s+In\s+received|Payment\s+received|Payment\s+made|Payment\s+of|Payment\s+for)`)

	// Flow-specific matchers
	reCashOut         = regexp.MustCompile(`(?i)^Cash\s+Out\s+made\s+for\s+GHS\s*([\d,]+\.?\d*)\s+to\s+(.+?)(?:\.\s*Current|\s+Current)`)
	reCashIn          = regexp.MustCompile(`(?i)^Cash\s+In\s+received\s+for\s+GHS\s*([\d,]+\.?\d*)\s+from\s+(.+?)(?:\.\s*Current|\s+Current)`)
	rePaymentReceived = regexp.MustCompile(`(?i)^Payment\s+received\s+for\s+GHS\s*([\d,]+\.?\d*)\s+from\s+(.+?)(?:\.\s*Current|\s+Current)`)
	rePaymentMade     = regexp.MustCompile(`(?i)^Payment\s+made\s+for\s+GHS\s*([\d,]+\.?\d*)\s+to\s+(.+?)(?:\.\s*Current|\s+Current)`)
	rePaymentOf       = regexp.MustCompile(`(?i)^Payment\s+of\s+GHS\s*([\d,]+\.?\d*)\s+made\s+to\s+(.+?)(?:\.\s*Current|\s+Current)`)
	rePaymentFor      = regexp.MustCompile(`(?i)^Payment\s+for\s+GHS\s*([\d,]+\.?\d*)\s+to\s+(.+?)(?:\.\.Current|\.\s*Current|\s+Current)`)
)

// ClassifyFlow classifies the SMS message into a specific FlowType.
func (p *MtnMomoParser) ClassifyFlow(body string) FlowType {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return FlowUnknown
	}

	// 1. Explicit Non-Financial notifications
	if (reNonFinancial.MatchString(trimmed) || rePureBalance.MatchString(trimmed)) && !reTxID.MatchString(trimmed) {
		return FlowNonFinancial
	}

	// 2. Failed transactions
	if reFailed.MatchString(trimmed) {
		return FlowFailedTransaction
	}

	// 3. Complete valid transaction flows
	switch {
	case reCashOut.MatchString(trimmed) && reTxID.MatchString(trimmed):
		return FlowCashOutWithdrawal
	case reCashIn.MatchString(trimmed) && reTxID.MatchString(trimmed):
		return FlowCashInDeposit
	case rePaymentReceived.MatchString(trimmed) && reTxID.MatchString(trimmed):
		return FlowPaymentReceived
	case (rePaymentMade.MatchString(trimmed) || rePaymentOf.MatchString(trimmed)) && reTxID.MatchString(trimmed):
		return FlowTransferSent
	case rePaymentFor.MatchString(trimmed) && reTxID.MatchString(trimmed):
		return FlowPaymentExpense
	}

	// 4. Truncated or malformed transaction notifications
	if reKnownPrefix.MatchString(trimmed) {
		return FlowMalformed
	}

	return FlowUnknown
}

// Parse extracts normalized transaction fields using a flow-based switch.
func (p *MtnMomoParser) Parse(sender, body string) (parsers.TransactionCandidate, error) {
	trimmed := strings.TrimSpace(body)
	flow := p.ClassifyFlow(trimmed)

	switch flow {
	case FlowCashOutWithdrawal:
		return p.parseCashOut(sender, trimmed)
	case FlowCashInDeposit:
		return p.parseCashIn(sender, trimmed)
	case FlowPaymentReceived:
		return p.parsePaymentReceived(sender, trimmed)
	case FlowTransferSent:
		return p.parseTransferSent(sender, trimmed)
	case FlowPaymentExpense:
		return p.parsePaymentExpense(sender, trimmed)
	case FlowFailedTransaction:
		return p.parseFailedTransaction(sender, trimmed)
	case FlowNonFinancial:
		return parsers.TransactionCandidate{}, parsers.ErrNonFinancialMessage
	case FlowMalformed:
		return parsers.TransactionCandidate{}, parsers.ErrMalformedMessage
	default:
		return parsers.TransactionCandidate{}, parsers.ErrUnrecognizedFormat
	}
}

func (p *MtnMomoParser) parseCashOut(sender, body string) (parsers.TransactionCandidate, error) {
	matches := reCashOut.FindStringSubmatch(body)
	if len(matches) < 3 {
		return parsers.TransactionCandidate{}, parsers.ErrMalformedMessage
	}

	amount, err := parseAmountToMinorUnits(matches[1])
	if err != nil {
		return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid cash out amount %s", parsers.ErrMalformedMessage, matches[1])
	}

	counterparty := cleanCounterparty(matches[2])
	return p.buildCandidate(sender, body, amount, parsers.DirectionOutbound, parsers.TxTypeCashWithdrawal, counterparty, nil)
}

func (p *MtnMomoParser) parseCashIn(sender, body string) (parsers.TransactionCandidate, error) {
	matches := reCashIn.FindStringSubmatch(body)
	if len(matches) < 3 {
		return parsers.TransactionCandidate{}, parsers.ErrMalformedMessage
	}

	amount, err := parseAmountToMinorUnits(matches[1])
	if err != nil {
		return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid cash in amount %s", parsers.ErrMalformedMessage, matches[1])
	}

	counterparty := cleanCounterparty(matches[2])
	return p.buildCandidate(sender, body, amount, parsers.DirectionInbound, parsers.TxTypeTransfer, counterparty, nil)
}

func (p *MtnMomoParser) parsePaymentReceived(sender, body string) (parsers.TransactionCandidate, error) {
	matches := rePaymentReceived.FindStringSubmatch(body)
	if len(matches) < 3 {
		return parsers.TransactionCandidate{}, parsers.ErrMalformedMessage
	}

	amount, err := parseAmountToMinorUnits(matches[1])
	if err != nil {
		return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid payment received amount %s", parsers.ErrMalformedMessage, matches[1])
	}

	counterparty := cleanCounterparty(matches[2])
	var counterpartyID *string
	if phoneMatch := rePhoneInRef.FindStringSubmatch(body); len(phoneMatch) >= 2 {
		pID := phoneMatch[1]
		counterpartyID = &pID
	}

	return p.buildCandidate(sender, body, amount, parsers.DirectionInbound, parsers.TxTypeTransfer, counterparty, counterpartyID)
}

func (p *MtnMomoParser) parseTransferSent(sender, body string) (parsers.TransactionCandidate, error) {
	var matches []string
	if rePaymentMade.MatchString(body) {
		matches = rePaymentMade.FindStringSubmatch(body)
	} else if rePaymentOf.MatchString(body) {
		matches = rePaymentOf.FindStringSubmatch(body)
	}

	if len(matches) < 3 {
		return parsers.TransactionCandidate{}, parsers.ErrMalformedMessage
	}

	amount, err := parseAmountToMinorUnits(matches[1])
	if err != nil {
		return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid transfer sent amount %s", parsers.ErrMalformedMessage, matches[1])
	}

	counterparty := cleanCounterparty(matches[2])
	var counterpartyID *string
	if m := reTrailingPhone.FindStringSubmatch(counterparty); len(m) >= 3 {
		counterparty = cleanCounterparty(m[1])
		pID := m[2]
		counterpartyID = &pID
	}

	return p.buildCandidate(sender, body, amount, parsers.DirectionOutbound, parsers.TxTypeTransfer, counterparty, counterpartyID)
}

func (p *MtnMomoParser) parsePaymentExpense(sender, body string) (parsers.TransactionCandidate, error) {
	matches := rePaymentFor.FindStringSubmatch(body)
	if len(matches) < 3 {
		return parsers.TransactionCandidate{}, parsers.ErrMalformedMessage
	}

	amount, err := parseAmountToMinorUnits(matches[1])
	if err != nil {
		return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid bill/bundle payment amount %s", parsers.ErrMalformedMessage, matches[1])
	}

	counterparty := cleanCounterparty(matches[2])
	return p.buildCandidate(sender, body, amount, parsers.DirectionOutbound, parsers.TxTypeExpense, counterparty, nil)
}

func (p *MtnMomoParser) parseFailedTransaction(sender, body string) (parsers.TransactionCandidate, error) {
	var providerRef *string
	if txMatch := reTxID.FindStringSubmatch(body); len(txMatch) >= 2 {
		ref := txMatch[1]
		providerRef = &ref
	}

	h := sha256.Sum256([]byte(body))
	payloadHash := hex.EncodeToString(h[:])

	return parsers.TransactionCandidate{
		Provider:          parsers.ProviderMTN,
		SourceType:        parsers.SourceTypeSMS,
		Sender:            sender,
		Status:            parsers.TxStatusFailed,
		AmountMinorUnits:  0, // Zero expense on failure
		Currency:          "GHS",
		Direction:         parsers.DirectionOutbound,
		TransactionType:   parsers.TxTypeExpense,
		ProviderReference: providerRef,
		EventTimestamp:    time.Now().UTC(),
		RawPayload:        body,
		PayloadHash:       payloadHash,
	}, nil
}

func (p *MtnMomoParser) buildCandidate(
	sender, rawBody string,
	amountMinor int64,
	direction, txType, counterparty string,
	counterpartyID *string,
) (parsers.TransactionCandidate, error) {
	txMatch := reTxID.FindStringSubmatch(rawBody)
	if len(txMatch) < 2 {
		return parsers.TransactionCandidate{}, parsers.ErrMalformedMessage
	}
	providerRef := txMatch[1]

	var balanceMinor *int64
	if balMatch := reBalance.FindStringSubmatch(rawBody); len(balMatch) >= 2 {
		if bal, err := parseAmountToMinorUnits(balMatch[1]); err == nil {
			balanceMinor = &bal
		}
	}

	var feeMinor *int64
	if feeMatch := reFee.FindStringSubmatch(rawBody); len(feeMatch) >= 2 {
		if fee, err := parseAmountToMinorUnits(feeMatch[1]); err == nil {
			feeMinor = &fee
		}
	}

	h := sha256.Sum256([]byte(rawBody))
	payloadHash := hex.EncodeToString(h[:])

	var cpPtr *string
	if counterparty != "" {
		cpPtr = &counterparty
	}

	return parsers.TransactionCandidate{
		Provider:               parsers.ProviderMTN,
		SourceType:             parsers.SourceTypeSMS,
		Sender:                 sender,
		Status:                 parsers.TxStatusCompleted,
		AmountMinorUnits:       amountMinor,
		Currency:               "GHS",
		Direction:              direction,
		TransactionType:        txType,
		Counterparty:           cpPtr,
		CounterpartyIdentifier: counterpartyID,
		ProviderReference:      &providerRef,
		BalanceMinorUnits:      balanceMinor,
		FeeMinorUnits:          feeMinor,
		EventTimestamp:         time.Now().UTC(),
		RawPayload:             rawBody,
		PayloadHash:            payloadHash,
	}, nil
}

// cleanCounterparty trims leading/trailing spaces, dots, and common separator noise.
func cleanCounterparty(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ".")
	return strings.TrimSpace(s)
}

// parseAmountToMinorUnits converts monetary string into integer minor units (pesewas) without floats.
func parseAmountToMinorUnits(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0, fmt.Errorf("empty amount string")
	}

	parts := strings.Split(s, ".")
	if len(parts) == 1 {
		val, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer amount %q: %w", s, err)
		}
		return val * 100, nil
	}

	if len(parts) == 2 {
		major, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid major units in %q: %w", s, err)
		}
		centsStr := parts[1]
		if len(centsStr) == 0 {
			return major * 100, nil
		}
		if len(centsStr) == 1 {
			centsStr += "0"
		} else if len(centsStr) > 2 {
			centsStr = centsStr[:2]
		}
		minor, err := strconv.ParseInt(centsStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minor units in %q: %w", s, err)
		}
		return major*100 + minor, nil
	}

	return 0, fmt.Errorf("invalid amount format %q", s)
}
