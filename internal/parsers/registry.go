package parsers

import (
	"fmt"
	"strings"
	"sync"
)

// DetectProvider inspects sender identifier and message body to determine the financial provider using a switch.
func DetectProvider(sender, body string) Provider {
	normalizedSender := strings.ToLower(strings.TrimSpace(sender))
	normalizedBody := strings.ToLower(body)

	switch {
	case isMTNSender(normalizedSender, normalizedBody):
		return ProviderMTN
	case isTelecelSender(normalizedSender, normalizedBody):
		return ProviderTelecel
	case isATMoneySender(normalizedSender, normalizedBody):
		return ProviderATMoney
	default:
		return ProviderUnknown
	}
}

func isMTNSender(sender, body string) bool {
	switch sender {
	case "mobilemoney", "mtn", "mtnmomo", "mtn mobilemoney", "momo":
		return true
	}
	return strings.Contains(body, "mtn mobile money") ||
		strings.Contains(body, "mtnmymomo.onelink.me") ||
		strings.Contains(body, "mtn momo")
}

func isTelecelSender(sender, body string) bool {
	switch sender {
	case "telecelcash", "telecel", "vodafonecash", "vodafone":
		return true
	}
	return strings.Contains(body, "telecel cash") || strings.Contains(body, "vodafone cash")
}

func isATMoneySender(sender, body string) bool {
	switch sender {
	case "atmoney", "at money", "airteltigo", "airteltigomoney":
		return true
	}
	return strings.Contains(body, "at money") || strings.Contains(body, "airteltigo money")
}

// Registry maintains the collection of registered ProviderAdapter instances.
type Registry struct {
	mu       sync.RWMutex
	adapters map[Provider]ProviderAdapter
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
		adapters: make(map[Provider]ProviderAdapter),
	}
}

// Register adds an adapter to the registry in a thread-safe manner.
func (r *Registry) Register(adapter ProviderAdapter) {
	if adapter == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Provider()] = adapter
}

// GetAdapter retrieves the adapter for a given Provider using a switch.
func (r *Registry) GetAdapter(p Provider) (ProviderAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[p]
	if !ok {
		return nil, fmt.Errorf("%w: provider=%q", ErrUnsupportedProvider, p)
	}
	return adapter, nil
}

// FindParser locates an adapter capable of handling the message.
func (r *Registry) FindParser(sender, body string) (ProviderAdapter, error) {
	// 1. Detect provider via sender/body switch
	provider := DetectProvider(sender, body)
	if provider != ProviderUnknown {
		if adapter, err := r.GetAdapter(provider); err == nil {
			return adapter, nil
		}
	}

	// 2. Fallback: inspect registered adapters via CanParse
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.adapters {
		if a.CanParse(sender, body) {
			return a, nil
		}
	}

	return nil, fmt.Errorf("%w: sender=%q", ErrUnsupportedProvider, sender)
}

// Parse locates the appropriate provider adapter and parses the message into a TransactionCandidate.
func (r *Registry) Parse(sender, body string) (TransactionCandidate, error) {
	parser, err := r.FindParser(sender, body)
	if err != nil {
		return TransactionCandidate{}, err
	}

	return parser.Parse(sender, body)
}

// ListProviders returns a list of unique provider names currently registered.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for p := range r.adapters {
		names = append(names, string(p))
	}
	return names
}

// RegisterDefault registers an adapter with the default registry.
func RegisterDefault(p ProviderAdapter) {
	DefaultRegistry().Register(p)
}

// ParseDefault parses a message using the default registry.
func ParseDefault(sender, body string) (TransactionCandidate, error) {
	return DefaultRegistry().Parse(sender, body)
}
