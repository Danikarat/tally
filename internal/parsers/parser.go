package parsers

// ProviderAdapter defines the standard adapter interface for provider-specific message parsers.
type ProviderAdapter interface {
	// Provider returns the canonical identifier for the provider (e.g. ProviderMTN, ProviderTelecel).
	Provider() Provider

	// CanParse evaluates whether the adapter can handle a message given its sender and body.
	CanParse(sender, body string) bool

	// Parse extracts normalized transaction fields from the message text.
	Parse(sender, body string) (TransactionCandidate, error)
}

// TransactionParser is an alias for ProviderAdapter to preserve compatibility across existing call sites.
type TransactionParser = ProviderAdapter
