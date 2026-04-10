package provider

import (
	"context"

	"github.com/jcprog/token-trail/internal/models"
)

// Provider defines the interface for LLM usage data providers
type Provider interface {
	// ID returns the unique provider identifier (e.g., "anthropic", "openai")
	ID() string

	// DisplayName returns the human-readable name
	DisplayName() string

	// ValidateKey checks if the API key is valid and has access to usage data
	ValidateKey(ctx context.Context, apiKey string) error

	// FetchUsage retrieves usage data for the given time range
	// Returns raw usage events that will be deduplicated and stored
	FetchUsage(ctx context.Context, apiKey string, from, to interface{}) ([]models.UsageEvent, error)

	// SupportedModels returns the list of known models for this provider
	SupportedModels() []string
}
