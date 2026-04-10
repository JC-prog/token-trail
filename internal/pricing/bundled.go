package pricing

import (
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jcprog/token-trail/internal/database"
)

//go:embed pricing.json
var pricingFS embed.FS

type PricingEntry struct {
	ProviderID              string `json:"provider_id"`
	Model                   string `json:"model"`
	InputPricePerMtok       float64 `json:"input_price_per_mtok"`
	OutputPricePerMtok      float64 `json:"output_price_per_mtok"`
	CacheReadPricePerMtok   float64 `json:"cache_read_price_per_mtok"`
	CacheWritePricePerMtok  float64 `json:"cache_write_price_per_mtok"`
	EffectiveFrom           string  `json:"effective_from"`
}

// LoadBundledPricing loads the bundled pricing.json into the database
func LoadBundledPricing(db *database.DB) error {
	data, err := pricingFS.ReadFile("pricing.json")
	if err != nil {
		return fmt.Errorf("failed to read bundled pricing: %w", err)
	}

	var entries []PricingEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal pricing data: %w", err)
	}

	// Insert pricing entries, ignoring duplicates
	for _, entry := range entries {
		id := uuid.New().String()
		effectiveFrom, err := time.Parse("2006-01-02", entry.EffectiveFrom)
		if err != nil {
			return fmt.Errorf("invalid effective_from date: %w", err)
		}

		query := `
			INSERT OR IGNORE INTO pricing
			(id, provider_id, model, input_price_per_mtok, output_price_per_mtok,
			 cache_read_price_per_mtok, cache_write_price_per_mtok, effective_from)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err = db.Exec(query, id, entry.ProviderID, entry.Model,
			entry.InputPricePerMtok, entry.OutputPricePerMtok,
			entry.CacheReadPricePerMtok, entry.CacheWritePricePerMtok,
			effectiveFrom)
		if err != nil {
			return fmt.Errorf("failed to insert pricing: %w", err)
		}
	}

	return nil
}
