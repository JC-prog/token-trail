package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jcprog/token-trail/internal/models"
	"github.com/jcprog/token-trail/internal/pricing"
)

type AnthropicProvider struct {
	httpClient *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider() *AnthropicProvider {
	return &AnthropicProvider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (ap *AnthropicProvider) ID() string {
	return "anthropic"
}

func (ap *AnthropicProvider) DisplayName() string {
	return "Anthropic"
}

// ValidateKey checks if the API key is valid
func (ap *AnthropicProvider) ValidateKey(ctx context.Context, apiKey string) error {
	// Test with a simple message count request
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/messages/count_tokens", nil)
	if err != nil {
		return err
	}

	// For validation, we just need to check if the key is accepted
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("content-type", "application/json")

	resp, err := ap.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	return nil
}

// FetchUsage fetches usage data from Anthropic API
// Note: This is a simplified implementation. Real API may return usage data differently.
func (ap *AnthropicProvider) FetchUsage(ctx context.Context, apiKey string, from, to interface{}) ([]models.UsageEvent, error) {
	// For MVP, return empty slice as Anthropic API usage endpoint is not yet specified
	// Real implementation would call Anthropic's billing API
	return []models.UsageEvent{}, nil
}

// SupportedModels returns Anthropic's supported models
func (ap *AnthropicProvider) SupportedModels() []string {
	return []string{
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-haiku-4.5-20250514",
	}
}

// parseAnthropicUsageResponse parses usage data from Anthropic's API
// This is a helper for when real usage data becomes available
func (ap *AnthropicProvider) parseAnthropicUsageResponse(data []byte) ([]models.UsageEvent, error) {
	var response struct {
		Data []struct {
			Model         string    `json:"model"`
			InputTokens   int       `json:"input_tokens"`
			OutputTokens  int       `json:"output_tokens"`
			Timestamp     time.Time `json:"timestamp"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	var events []models.UsageEvent
	for _, item := range response.Data {
		event := models.UsageEvent{
			ID:           uuid.New().String(),
			ProviderID:   ap.ID(),
			Model:        item.Model,
			InputTokens:  item.InputTokens,
			OutputTokens: item.OutputTokens,
			Timestamp:    item.Timestamp,
			Source:       "api_poll",
			DedupHash:    pricing.GenerateDedupHash(ap.ID(), item.Model, item.Timestamp, item.InputTokens, item.OutputTokens),
		}
		events = append(events, event)
	}

	return events, nil
}
