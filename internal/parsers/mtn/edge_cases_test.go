package mtn_test

import (
	"errors"
	"testing"

	"github.com/Danikarat/tally/internal/parsers"
	"github.com/Danikarat/tally/internal/parsers/mtn"
)

func TestMtnMomoParser_NonFinancialAndBalanceEdgeCases(t *testing.T) {
	parser := mtn.New()

	nonFinancialCases := []struct {
		name string
		body string
	}{
		{
			name: "pure balance notification with current and available",
			body: "Current Balance: GHS 150.00. Available Balance: GHS 150.00.",
		},
		{
			name: "your current balance inquiry",
			body: "Your current balance is GHS 240.50.",
		},
		{
			name: "dear customer balance notice",
			body: "Dear Customer, your balance is GHS 12.30. Thank you for using MTN.",
		},
		{
			name: "security OTP alert",
			body: "Your one-time password is 458921. Do not share your PIN or OTP with anyone.",
		},
		{
			name: "standard OTP notice",
			body: "Your OTP is 781290. Valid for 10 minutes.",
		},
		{
			name: "authorization prompt without transaction",
			body: "Authorize payment of GHS 50.00 to Merchant? Dial *170# to approve.",
		},
		{
			name: "MoMo marketing promotional banner",
			body: "Win with MTN MoMo! Dial *170# now to win exciting prizes.",
		},
		{
			name: "recharge bonus promo",
			body: "Y'ello! Enjoy 50% extra bonus on your next recharge this weekend.",
		},
		{
			name: "pin security reminder",
			body: "Do not share your MoMo PIN with anyone, including MTN staff.",
		},
	}

	for _, tc := range nonFinancialCases {
		t.Run(tc.name, func(t *testing.T) {
			flow := parser.ClassifyFlow(tc.body)
			if flow != mtn.FlowNonFinancial {
				t.Errorf("expected FlowNonFinancial for %q, got flow=%v", tc.name, flow)
			}

			_, err := parser.Parse("MobileMoney", tc.body)
			if !errors.Is(err, parsers.ErrNonFinancialMessage) {
				t.Fatalf("expected ErrNonFinancialMessage, got err=%v", err)
			}
		})
	}
}

func TestMtnMomoParser_FailedTransactions(t *testing.T) {
	parser := mtn.New()

	failedCases := []struct {
		name        string
		body        string
		expectedRef string
	}{
		{
			name:        "transfer failed notification with transaction id",
			body:        "Payment for GHS 25.00 to John Doe failed. Transaction ID: 91380000010. Your balance was not debited.",
			expectedRef: "91380000010",
		},
		{
			name:        "payment failed due to insufficient funds",
			body:        "Payment of GHS 100.00 to Vendor failed due to insufficient funds. Transaction Id: 91380000020.",
			expectedRef: "91380000020",
		},
		{
			name:        "transaction failed generic notice",
			body:        "Transaction 91380000030 failed. Please try again later.",
			expectedRef: "",
		},
	}

	for _, tc := range failedCases {
		t.Run(tc.name, func(t *testing.T) {
			flow := parser.ClassifyFlow(tc.body)
			if flow != mtn.FlowFailedTransaction {
				t.Errorf("expected FlowFailedTransaction, got flow=%v", flow)
			}

			cand, err := parser.Parse("MobileMoney", tc.body)
			if err != nil {
				t.Fatalf("unexpected error parsing failed transaction: %v", err)
			}

			if cand.Status != parsers.TxStatusFailed {
				t.Errorf("expected Status=%q, got %q", parsers.TxStatusFailed, cand.Status)
			}

			if cand.AmountMinorUnits != 0 {
				t.Errorf("expected AmountMinorUnits=0 (zero expense for failed transaction), got %d", cand.AmountMinorUnits)
			}

			if tc.expectedRef != "" {
				if cand.ProviderReference == nil || *cand.ProviderReference != tc.expectedRef {
					t.Errorf("expected ProviderReference=%q, got %v", tc.expectedRef, cand.ProviderReference)
				}
			}
		})
	}
}

func TestMtnMomoParser_MalformedAndTruncatedMessages(t *testing.T) {
	parser := mtn.New()

	malformedCases := []struct {
		name string
		body string
	}{
		{
			name: "truncated payment made without transaction ID",
			body: "Payment made for GHS 35.00 to KOFI MENSAH Current Balance: GHS 22.91",
		},
		{
			name: "truncated cash out without transaction ID",
			body: "Cash Out made for GHS50.00 to ACCRA AGENT VENTURES. Current Balance: GHS3.41",
		},
		{
			name: "truncated cash in without body",
			body: "Cash In received for GHS 100.00 from",
		},
		{
			name: "truncated payment for bill without ID",
			body: "Payment for GHS10.00 to Bills.INV ..Current Balance: GHS 56.91.",
		},
		{
			name: "payment with empty or invalid amount",
			body: "Payment for GHS to Bills.INV. Current Balance: GHS. Transaction Id: 91333000004.",
		},
	}

	for _, tc := range malformedCases {
		t.Run(tc.name, func(t *testing.T) {
			flow := parser.ClassifyFlow(tc.body)
			if flow != mtn.FlowMalformed {
				t.Errorf("expected FlowMalformed for %q, got flow=%v", tc.name, flow)
			}

			_, err := parser.Parse("MobileMoney", tc.body)
			if !errors.Is(err, parsers.ErrMalformedMessage) {
				t.Fatalf("expected ErrMalformedMessage for %q, got err=%v", tc.name, err)
			}
		})
	}
}

func TestMtnMomoParser_UnrecognizedFormat(t *testing.T) {
	parser := mtn.New()

	unrecognizedCases := []struct {
		name string
		body string
	}{
		{
			name: "empty string",
			body: "",
		},
		{
			name: "whitespace only",
			body: "    \t\n   ",
		},
		{
			name: "arbitrary unrelated SMS",
			body: "Hello, meeting is scheduled for 3pm at the office.",
		},
		{
			name: "gibberish text",
			body: "asdkjh123891723 aksjdhakjsd",
		},
	}

	for _, tc := range unrecognizedCases {
		t.Run(tc.name, func(t *testing.T) {
			flow := parser.ClassifyFlow(tc.body)
			if flow != mtn.FlowUnknown {
				t.Errorf("expected FlowUnknown for %q, got flow=%v", tc.name, flow)
			}

			_, err := parser.Parse("MobileMoney", tc.body)
			if !errors.Is(err, parsers.ErrUnrecognizedFormat) {
				t.Fatalf("expected ErrUnrecognizedFormat for %q, got err=%v", tc.name, err)
			}
		})
	}
}
