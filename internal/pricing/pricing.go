package pricing

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jcprog/token-trail/internal/database"
	"github.com/jcprog/token-trail/internal/models"
)

type Engine struct {
	db *database.DB
}

// NewEngine creates a new pricing engine
func NewEngine(db *database.DB) *Engine {
	return &Engine{db: db}
}

// CalculateCost calculates the cost of a usage event based on time-aware pricing
func (e *Engine) CalculateCost(event *models.UsageEvent) (float64, error) {
	pricing, err := e.db.GetPricingForModel(event.ProviderID, event.Model, event.Timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to get pricing: %w", err)
	}

	// Calculate cost based on tokens and pricing
	inputCost := (float64(event.InputTokens) / 1_000_000.0) * pricing.InputPricePerMtok
	outputCost := (float64(event.OutputTokens) / 1_000_000.0) * pricing.OutputPricePerMtok
	cacheReadCost := (float64(event.CacheReadTokens) / 1_000_000.0) * pricing.CacheReadPricePerMtok
	cacheWriteCost := (float64(event.CacheWriteTokens) / 1_000_000.0) * pricing.CacheWritePricePerMtok

	totalCost := inputCost + outputCost + cacheReadCost + cacheWriteCost
	return totalCost, nil
}

// GenerateDedupHash creates a unique hash for deduplication
func GenerateDedupHash(providerID, model string, timestamp time.Time, inputTokens, outputTokens int) string {
	data := fmt.Sprintf("%s:%s:%s:%d:%d",
		providerID,
		model,
		timestamp.Format(time.RFC3339),
		inputTokens,
		outputTokens,
	)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}
