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

const ProviderName = "MTN"

// MtnMomoParser parses MTN MoMo Ghana transaction SMS messages.
type MtnMomoParser struct{}

func init() {
	parsers.RegisterDefault(New())
}

// New creates a new instance of MtnMomoParser.
func New() *MtnMomoParser {
	return &MtnMomoParser{}
}

// Provider returns the canonical identifier for MTN.
func (p *MtnMomoParser) Provider() string {
	return ProviderName
}

// CanParse checks if the sender and/or body indicate an MTN MoMo message.
func (p *MtnMomoParser) CanParse(sender, body string) bool {
	senderNormalized := strings.ToLower(strings.TrimSpace(sender))
	if senderNormalized == "mobilemoney" ||
		senderNormalized == "mtn" ||
		senderNormalized == "mtnmomo" ||
		senderNormalized == "mtn mobilemoney" ||
		senderNormalized == "momo" {
		return true
	}

	bodyLower := strings.ToLower(body)
	if strings.Contains(bodyLower, "mtn mobile money") ||
		strings.Contains(bodyLower, "mtnmymomo.onelink.me") ||
		strings.Contains(bodyLower, "mtn momo") {
		return true
	}

	return false
}

var (
	// Regex for transaction reference / ID
	reTxID = regexp.MustCompile(`(?i)(?:Financial\s+)?Transaction\s+Id:\s*([A-Za-z0-9]+)`)

	// Regex for balance
	reBalance = regexp.MustCompile(`(?i)Current\s+Balance:?\s*(?:GHS\s*)?([\d,]+\.?\d*)`)

	// Regex for fee
	reFee = regexp.MustCompile(`(?i)(?:Fee\s+charged|TRANSACTION\s+FEE|Fee\s+was):\s*(?:GHS\s*)?([\d,]+\.?\d*)`)

	// Regex for phone in reference field: e.g. "Reference: KWAME ASANTE ,233500000001,1 from VODAFONE."
	rePhoneInRef = regexp.MustCompile(`,\s*(\d{10,14})\b`)

	// Regex for non-financial indicators
	reNonFinancial = regexp.MustCompile(`(?i)(?:one[\s-]time\s+password|your\s+otp\s+is|enter\s+pin\s+to\s+approve|dial\s+\*170#|win\s+with\s+mtn|y'ello!\s+enjoy)`)

	// Specific flow matchers
	// 1. Cash Out made
	reCashOut = regexp.MustCompile(`(?i)^Cash\s+Out\s+made\s+for\s+GHS\s*([\d,]+\.?\d*)\s+to\s+(.+?)(?:\.\s*Current|\s+Current)`)

	// 2. Cash In received
	reCashIn = regexp.MustCompile(`(?i)^Cash\s+In\s+received\s+for\s+GHS\s*([\d,]+\.?\d*)\s+from\s+(.+?)(?:\.\s*Current|\s+Current)`)

	// 3. Payment received
	rePaymentReceived = regexp.MustCompile(`(?i)^Payment\s+received\s+for\s+GHS\s*([\d,]+\.?\d*)\s+from\s+(.+?)(?:\.\s*Current|\s+Current)`)

	// 4. Payment made for
	rePaymentMade = regexp.MustCompile(`(?i)^Payment\s+made\s+for\s+GHS\s*([\d,]+\.?\d*)\s+to\s+(.+?)(?:\.\s*Current|\s+Current)`)

	// 5. Payment of ... made to
	rePaymentOf = regexp.MustCompile(`(?i)^Payment\s+of\s+GHS\s*([\d,]+\.?\d*)\s+made\s+to\s+(.+?)(?:\.\s*Current|\s+Current)`)

	// 6. Payment for (Bills / Bundles)
	rePaymentFor = regexp.MustCompile(`(?i)^Payment\s+for\s+GHS\s*([\d,]+\.?\d*)\s+to\s+(.+?)(?:\.\.Current|\.\s*Current|\s+Current)`)
)

// Parse extracts normalized transaction fields from the message text.
func (p *MtnMomoParser) Parse(sender, body string) (parsers.TransactionCandidate, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return parsers.TransactionCandidate{}, parsers.ErrUnrecognizedFormat
	}

	// Check for non-financial patterns
	if reNonFinancial.MatchString(trimmed) && !reTxID.MatchString(trimmed) {
		return parsers.TransactionCandidate{}, parsers.ErrNonFinancialMessage
	}

	// Extract Transaction ID
	txMatch := reTxID.FindStringSubmatch(trimmed)
	if len(txMatch) < 2 {
		return parsers.TransactionCandidate{}, parsers.ErrUnrecognizedFormat
	}
	providerRef := txMatch[1]

	var (
		amountMinor    int64
		direction      string
		txType         string
		counterparty   string
		counterpartyID *string
		err            error
	)

	switch {
	case reCashOut.MatchString(trimmed):
		matches := reCashOut.FindStringSubmatch(trimmed)
		amountMinor, err = parseAmountToMinorUnits(matches[1])
		if err != nil {
			return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid amount %s", parsers.ErrUnrecognizedFormat, matches[1])
		}
		direction = "OUTBOUND"
		txType = "CASH_WITHDRAWAL"
		counterparty = cleanCounterparty(matches[2])

	case reCashIn.MatchString(trimmed):
		matches := reCashIn.FindStringSubmatch(trimmed)
		amountMinor, err = parseAmountToMinorUnits(matches[1])
		if err != nil {
			return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid amount %s", parsers.ErrUnrecognizedFormat, matches[1])
		}
		direction = "INBOUND"
		txType = "TRANSFER"
		counterparty = cleanCounterparty(matches[2])

	case rePaymentReceived.MatchString(trimmed):
		matches := rePaymentReceived.FindStringSubmatch(trimmed)
		amountMinor, err = parseAmountToMinorUnits(matches[1])
		if err != nil {
			return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid amount %s", parsers.ErrUnrecognizedFormat, matches[1])
		}
		direction = "INBOUND"
		txType = "TRANSFER"
		counterparty = cleanCounterparty(matches[2])

		// Check for phone number in Reference if present
		if phoneMatch := rePhoneInRef.FindStringSubmatch(trimmed); len(phoneMatch) >= 2 {
			pID := phoneMatch[1]
			counterpartyID = &pID
		}

	case rePaymentMade.MatchString(trimmed):
		matches := rePaymentMade.FindStringSubmatch(trimmed)
		amountMinor, err = parseAmountToMinorUnits(matches[1])
		if err != nil {
			return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid amount %s", parsers.ErrUnrecognizedFormat, matches[1])
		}
		direction = "OUTBOUND"
		txType = "TRANSFER"
		counterparty = cleanCounterparty(matches[2])

	case rePaymentOf.MatchString(trimmed):
		matches := rePaymentOf.FindStringSubmatch(trimmed)
		amountMinor, err = parseAmountToMinorUnits(matches[1])
		if err != nil {
			return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid amount %s", parsers.ErrUnrecognizedFormat, matches[1])
		}
		direction = "OUTBOUND"
		txType = "TRANSFER"
		counterparty = cleanCounterparty(matches[2])

	case rePaymentFor.MatchString(trimmed):
		matches := rePaymentFor.FindStringSubmatch(trimmed)
		amountMinor, err = parseAmountToMinorUnits(matches[1])
		if err != nil {
			return parsers.TransactionCandidate{}, fmt.Errorf("%w: invalid amount %s", parsers.ErrUnrecognizedFormat, matches[1])
		}
		direction = "OUTBOUND"
		txType = "EXPENSE"
		counterparty = cleanCounterparty(matches[2])

	default:
		return parsers.TransactionCandidate{}, parsers.ErrUnrecognizedFormat
	}

	// Extract Balance if present
	var balanceMinor *int64
	if balMatch := reBalance.FindStringSubmatch(trimmed); len(balMatch) >= 2 {
		if bal, err := parseAmountToMinorUnits(balMatch[1]); err == nil {
			balanceMinor = &bal
		}
	}

	// Extract Fee if present
	var feeMinor *int64
	if feeMatch := reFee.FindStringSubmatch(trimmed); len(feeMatch) >= 2 {
		if fee, err := parseAmountToMinorUnits(feeMatch[1]); err == nil {
			feeMinor = &fee
		}
	}

	// If counterparty has phone number appended at end (e.g., "John Doe 0241234567")
	if counterpartyID == nil {
		reTrailingPhone := regexp.MustCompile(`^(.*?)\s+(\d{10,14})$`)
		if m := reTrailingPhone.FindStringSubmatch(counterparty); len(m) >= 3 {
			counterparty = cleanCounterparty(m[1])
			pID := m[2]
			counterpartyID = &pID
		}
	}

	// Hash raw payload
	h := sha256.Sum256([]byte(body))
	payloadHash := hex.EncodeToString(h[:])

	var cpPtr *string
	if counterparty != "" {
		cpPtr = &counterparty
	}

	return parsers.TransactionCandidate{
		Provider:               ProviderName,
		SourceType:             "SMS",
		Sender:                 sender,
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
		RawPayload:             body,
		PayloadHash:            payloadHash,
	}, nil
}

// cleanCounterparty trims leading/trailing spaces, dots, and common separator noise.
func cleanCounterparty(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ".")
	return strings.TrimSpace(s)
}

// parseAmountToMinorUnits converts monetary string (e.g. "35.00", "0", "1,250.50")
// into integer minor units (pesewas) without using floating-point math.
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
