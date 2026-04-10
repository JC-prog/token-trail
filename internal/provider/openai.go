package provider

import (
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

type OpenAIProvider struct {
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (op *OpenAIProvider) ID() string {
	return "openai"
}

func (op *OpenAIProvider) DisplayName() string {
	return "OpenAI"
}

// ValidateKey checks if the API key is valid
func (op *OpenAIProvider) ValidateKey(ctx context.Context, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := op.httpClient.Do(req)
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

// FetchUsage fetches usage data from OpenAI API
// Note: This is a simplified implementation. Real API endpoint is /v1/organization/usage
func (op *OpenAIProvider) FetchUsage(ctx context.Context, apiKey string, from, to interface{}) ([]models.UsageEvent, error) {
	// For MVP, return empty slice as real implementation requires organization API access
	// Real implementation would call /v1/organization/usage
	return []models.UsageEvent{}, nil
}

// SupportedModels returns OpenAI's supported models
func (op *OpenAIProvider) SupportedModels() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"o1",
		"o3-mini",
	}
}

// parseOpenAIUsageResponse parses usage data from OpenAI's API
// This is a helper for when real usage data becomes available
func (op *OpenAIProvider) parseOpenAIUsageResponse(data []byte) ([]models.UsageEvent, error) {
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
			ProviderID:   op.ID(),
			Model:        item.Model,
			InputTokens:  item.InputTokens,
			OutputTokens: item.OutputTokens,
			Timestamp:    item.Timestamp,
			Source:       "api_poll",
			DedupHash:    pricing.GenerateDedupHash(op.ID(), item.Model, item.Timestamp, item.InputTokens, item.OutputTokens),
		}
		events = append(events, event)
	}

	return events, nil
}
