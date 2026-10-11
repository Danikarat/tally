package parsers

import (
	"fmt"
	"strings"
	"sync"
)

// Registry maintains the collection of registered TransactionParser adapters.
type Registry struct {
	mu      sync.RWMutex
	parsers []TransactionParser
}

var (
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
)

// DefaultRegistry returns the singleton global parser registry.
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// NewRegistry creates a new, independent parser registry.
func NewRegistry() *Registry {
	return &Registry{
		parsers: make([]TransactionParser, 0),
	}
}

// Register adds a parser to the registry in a thread-safe manner.
func (r *Registry) Register(p TransactionParser) {
	if p == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// Replace existing parser for the same provider if registered, or append
	for i, existing := range r.parsers {
		if strings.EqualFold(existing.Provider(), p.Provider()) {
			r.parsers[i] = p
			return
		}
	}
	r.parsers = append(r.parsers, p)
}

// FindParser locates a parser capable of handling the message.
func (r *Registry) FindParser(sender, body string) (TransactionParser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.parsers {
		if p.CanParse(sender, body) {
			return p, nil
		}
	}

	return nil, fmt.Errorf("%w: sender=%q", ErrUnsupportedProvider, sender)
}

// Parse locates the appropriate parser and parses the message into a TransactionCandidate.
func (r *Registry) Parse(sender, body string) (TransactionCandidate, error) {
	parser, err := r.FindParser(sender, body)
	if err != nil {
		return TransactionCandidate{}, err
	}

	candidate, err := parser.Parse(sender, body)
	if err != nil {
		return TransactionCandidate{}, err
	}

	return candidate, nil
}

// ListProviders returns a list of unique provider names currently registered.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.parsers))
	for i, p := range r.parsers {
		names[i] = p.Provider()
	}
	return names
}

// RegisterDefault registers a parser with the default registry.
func RegisterDefault(p TransactionParser) {
	DefaultRegistry().Register(p)
}

// ParseDefault parses a message using the default registry.
func ParseDefault(sender, body string) (TransactionCandidate, error) {
	return DefaultRegistry().Parse(sender, body)
}
