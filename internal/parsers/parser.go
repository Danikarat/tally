package parsers

// TransactionParser defines the standard adapter interface for provider-specific message parsers.
type TransactionParser interface {
	// Provider returns the canonical identifier for the provider (e.g. "MTN", "TELECEL", "AT_MONEY").
	Provider() string

	// CanParse evaluates whether the parser can handle a message given its sender and body.
	CanParse(sender, body string) bool

	// Parse extracts normalized transaction fields from the message text.
	Parse(sender, body string) (TransactionCandidate, error)
}
