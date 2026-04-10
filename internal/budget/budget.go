package budget

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jcprog/token-trail/internal/database"
	"github.com/jcprog/token-trail/internal/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Tracker struct {
	db *database.DB
}

// NewTracker creates a new budget tracker
func NewTracker(db *database.DB) *Tracker {
	return &Tracker{db: db}
}

// GetBudgets retrieves all budgets
func (bt *Tracker) GetBudgets() ([]models.Budget, error) {
	// For MVP, this is stubbed
	return []models.Budget{}, nil
}

// SetBudget creates or updates a budget
func (bt *Tracker) SetBudget(scope string, monthlyLimitUSD float64, thresholds []int) error {
	id := uuid.New().String()
	thresholdJSON, _ := json.Marshal(thresholds)

	query := `INSERT OR REPLACE INTO budgets (id, scope, monthly_limit_usd, alert_thresholds) VALUES (?, ?, ?, ?)`
	_, err := bt.db.Exec(query, id, scope, monthlyLimitUSD, string(thresholdJSON))
	return err
}

// DeleteBudget deletes a budget
func (bt *Tracker) DeleteBudget(scope string) error {
	query := `DELETE FROM budgets WHERE scope = ?`
	_, err := bt.db.Exec(query, scope)
	return err
}

// CheckBudgets checks if any budgets have been exceeded and emits alerts
func (bt *Tracker) CheckBudgets(ctx context.Context) error {
	statuses, err := bt.db.GetBudgetStatus()
	if err != nil {
		return err
	}

	for _, status := range statuses {
		for _, threshold := range status.AlertThresholds {
			if status.PercentUsed >= float64(threshold) {
				runtime.EventsEmit(ctx, "budget:alert", map[string]interface{}{
					"scope":         status.Scope,
					"percent_used":  status.PercentUsed,
					"threshold":     threshold,
					"spent":         status.SpentUSD,
					"limit":         status.MonthlyLimitUSD,
				})
			}
		}
	}

	return nil
}

// CalculateProjectedSpend projects the monthly spend based on current usage
func (bt *Tracker) CalculateProjectedSpend() (float64, error) {
	summary, err := bt.db.GetDashboardSummary()
	if err != nil {
		return 0, fmt.Errorf("failed to get summary: %w", err)
	}

	return summary.ProjectedSpend, nil
}
