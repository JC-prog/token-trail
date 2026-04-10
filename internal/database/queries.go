package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jcprog/token-trail/internal/models"
)

// InsertUsageEvent inserts a usage event into the database
func (db *DB) InsertUsageEvent(event *models.UsageEvent) error {
	query := `
		INSERT INTO usage_events
		(id, provider_id, model, input_tokens, output_tokens, cache_read_tokens,
		 cache_write_tokens, cost_usd, timestamp, source, project_id, session_id, dedup_hash, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(dedup_hash) DO NOTHING
	`
	_, err := db.Exec(query, event.ID, event.ProviderID, event.Model, event.InputTokens,
		event.OutputTokens, event.CacheReadTokens, event.CacheWriteTokens, event.CostUSD,
		event.Timestamp, event.Source, event.ProjectID, event.SessionID, event.DedupHash,
		event.Metadata, time.Now())
	return err
}

// GetDashboardSummary gets the current month's spending summary
func (db *DB) GetDashboardSummary() (*models.DashboardSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(cost_usd), 0) as total_spend,
			COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens,
			CAST(strftime('%d', 'now') AS INTEGER) as current_day,
			CAST(strftime('%d', datetime('now', 'start of month', '+1 month', '-1 day')) AS INTEGER) as days_in_month,
			CASE
				WHEN CAST(strftime('%d', 'now') AS INTEGER) > 1
				THEN COALESCE(SUM(cost_usd), 0) / (CAST(strftime('%d', 'now') AS INTEGER) - 1)
				ELSE 0
			END as avg_daily_spend
		FROM usage_events
		WHERE timestamp >= datetime('now', 'start of month')
	`

	var totalSpend, totalTokens, currentDay, daysInMonth sql.NullInt64
	var avgDailySpend sql.NullFloat64

	err := db.QueryRow(query).Scan(&totalSpend, &totalTokens, &currentDay, &daysInMonth, &avgDailySpend)
	if err != nil {
		return nil, err
	}

	summary := &models.DashboardSummary{
		TotalSpend:   float64(totalSpend.Int64),
		TotalTokens:  totalTokens.Int64,
		CurrentDay:   int(currentDay.Int64),
		DaysInMonth:  int(daysInMonth.Int64),
	}

	if avgDailySpend.Valid {
		summary.AvgDailySpend = avgDailySpend.Float64
		if summary.CurrentDay > 0 && summary.DaysInMonth > 0 {
			summary.ProjectedSpend = (summary.AvgDailySpend * float64(summary.DaysInMonth))
		}
	}

	return summary, nil
}

// GetSpendOverTime returns daily spend data for charts
func (db *DB) GetSpendOverTime(days int) ([]models.TimeSeriesPoint, error) {
	query := `
		SELECT
			date(timestamp) as day,
			SUM(cost_usd) as daily_spend,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens
		FROM usage_events
		WHERE timestamp >= date('now', ? || ' days')
		GROUP BY date(timestamp)
		ORDER BY day
	`

	rows, err := db.Query(query, fmt.Sprintf("-%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []models.TimeSeriesPoint
	for rows.Next() {
		var day string
		var spend float64
		var inputTokens, outputTokens int64

		if err := rows.Scan(&day, &spend, &inputTokens, &outputTokens); err != nil {
			return nil, err
		}

		t, err := time.Parse("2006-01-02", day)
		if err != nil {
			return nil, err
		}

		points = append(points, models.TimeSeriesPoint{
			Timestamp:    t,
			Spend:        spend,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
		})
	}

	return points, rows.Err()
}

// GetUsageByProvider returns spend breakdown by provider
func (db *DB) GetUsageByProvider(days int) ([]models.ProviderBreakdown, error) {
	query := `
		SELECT
			e.provider_id,
			p.display_name,
			COALESCE(SUM(e.cost_usd), 0) as total_cost,
			COALESCE(SUM(e.input_tokens + e.output_tokens), 0) as total_tokens
		FROM usage_events e
		LEFT JOIN providers p ON e.provider_id = p.id
		WHERE e.timestamp >= datetime('now', ? || ' days')
		GROUP BY e.provider_id
		ORDER BY total_cost DESC
	`

	rows, err := db.Query(query, fmt.Sprintf("-%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdowns []models.ProviderBreakdown
	var totalCost float64

	// First pass: calculate totals
	tempBreakdowns := []models.ProviderBreakdown{}
	for rows.Next() {
		var providerID, displayName string
		var cost float64
		var tokens int64

		if err := rows.Scan(&providerID, &displayName, &cost, &tokens); err != nil {
			return nil, err
		}

		tempBreakdowns = append(tempBreakdowns, models.ProviderBreakdown{
			ProviderID:  providerID,
			DisplayName: displayName,
			TotalCost:   cost,
			TotalTokens: tokens,
		})
		totalCost += cost
	}

	// Second pass: calculate percentages
	for _, bd := range tempBreakdowns {
		if totalCost > 0 {
			bd.Percentage = (bd.TotalCost / totalCost) * 100
		}
		breakdowns = append(breakdowns, bd)
	}

	return breakdowns, rows.Err()
}

// GetUsageByModel returns spend breakdown by model
func (db *DB) GetUsageByModel(days int) ([]models.ModelBreakdown, error) {
	query := `
		SELECT
			e.model,
			e.provider_id,
			p.display_name,
			COALESCE(SUM(e.cost_usd), 0) as total_cost,
			COALESCE(SUM(e.input_tokens), 0) as input_tokens,
			COALESCE(SUM(e.output_tokens), 0) as output_tokens
		FROM usage_events e
		LEFT JOIN providers p ON e.provider_id = p.id
		WHERE e.timestamp >= datetime('now', ? || ' days')
		GROUP BY e.model, e.provider_id
		ORDER BY total_cost DESC
	`

	rows, err := db.Query(query, fmt.Sprintf("-%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdowns []models.ModelBreakdown
	var totalCost float64

	tempBreakdowns := []models.ModelBreakdown{}
	for rows.Next() {
		var model, providerID, displayName string
		var cost float64
		var inputTokens, outputTokens int64

		if err := rows.Scan(&model, &providerID, &displayName, &cost, &inputTokens, &outputTokens); err != nil {
			return nil, err
		}

		tempBreakdowns = append(tempBreakdowns, models.ModelBreakdown{
			Model:        model,
			ProviderID:   providerID,
			DisplayName:  displayName,
			TotalCost:    cost,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
		})
		totalCost += cost
	}

	for _, bd := range tempBreakdowns {
		if totalCost > 0 {
			bd.Percentage = (bd.TotalCost / totalCost) * 100
		}
		breakdowns = append(breakdowns, bd)
	}

	return breakdowns, rows.Err()
}

// GetTokenRatio returns input vs output token breakdown
func (db *DB) GetTokenRatio(days int) (*models.TokenRatio, error) {
	query := `
		SELECT
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens
		FROM usage_events
		WHERE timestamp >= datetime('now', ? || ' days')
	`

	var inputTokens, outputTokens int64
	err := db.QueryRow(query, fmt.Sprintf("-%d", days)).Scan(&inputTokens, &outputTokens)
	if err != nil {
		return nil, err
	}

	ratio := &models.TokenRatio{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}

	if inputTokens > 0 {
		ratio.Ratio = float64(outputTokens) / float64(inputTokens)
	}

	return ratio, nil
}

// GetUsageEvents returns paginated usage events with optional filters
func (db *DB) GetUsageEvents(filter *models.EventFilter) (*models.PaginatedEvents, error) {
	query := `
		SELECT COUNT(*) FROM usage_events
		WHERE 1=1
	`
	countArgs := []interface{}{}

	if filter.Provider != nil {
		query += ` AND provider_id = ?`
		countArgs = append(countArgs, *filter.Provider)
	}
	if filter.Model != nil {
		query += ` AND model = ?`
		countArgs = append(countArgs, *filter.Model)
	}
	if filter.Project != nil {
		query += ` AND project_id = ?`
		countArgs = append(countArgs, *filter.Project)
	}
	if filter.FromDate != nil {
		query += ` AND timestamp >= ?`
		countArgs = append(countArgs, *filter.FromDate)
	}
	if filter.ToDate != nil {
		query += ` AND timestamp <= ?`
		countArgs = append(countArgs, *filter.ToDate)
	}

	var total int
	err := db.QueryRow(query, countArgs...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Fetch events
	query = `
		SELECT id, provider_id, model, input_tokens, output_tokens, cache_read_tokens,
		       cache_write_tokens, cost_usd, timestamp, source, project_id, session_id, dedup_hash, metadata, created_at
		FROM usage_events
		WHERE 1=1
	`

	if filter.Provider != nil {
		query += ` AND provider_id = ?`
	}
	if filter.Model != nil {
		query += ` AND model = ?`
	}
	if filter.Project != nil {
		query += ` AND project_id = ?`
	}
	if filter.FromDate != nil {
		query += ` AND timestamp >= ?`
	}
	if filter.ToDate != nil {
		query += ` AND timestamp <= ?`
	}

	query += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	countArgs = append(countArgs, filter.Limit, filter.Offset)

	rows, err := db.Query(query, countArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.UsageEvent
	for rows.Next() {
		var event models.UsageEvent
		var projectID, sessionID sql.NullString
		var metadata string

		err := rows.Scan(&event.ID, &event.ProviderID, &event.Model, &event.InputTokens, &event.OutputTokens,
			&event.CacheReadTokens, &event.CacheWriteTokens, &event.CostUSD, &event.Timestamp,
			&event.Source, &projectID, &sessionID, &event.DedupHash, &metadata, &event.CreatedAt)
		if err != nil {
			return nil, err
		}

		if projectID.Valid {
			event.ProjectID = &projectID.String
		}
		if sessionID.Valid {
			event.SessionID = &sessionID.String
		}
		event.Metadata = metadata

		events = append(events, event)
	}

	return &models.PaginatedEvents{
		Events: events,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, rows.Err()
}

// GetBudgetStatus returns current budget usage
func (db *DB) GetBudgetStatus() ([]models.BudgetStatus, error) {
	query := `
		SELECT
			b.scope,
			b.monthly_limit_usd,
			COALESCE(SUM(e.cost_usd), 0) as spent
		FROM budgets b
		LEFT JOIN usage_events e
			ON (b.scope = 'global' OR b.scope = e.provider_id)
			AND e.timestamp >= datetime('now', 'start of month')
		GROUP BY b.scope
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []models.BudgetStatus
	for rows.Next() {
		var scope string
		var limit, spent float64
		var thresholds string

		if err := rows.Scan(&scope, &limit, &spent); err != nil {
			return nil, err
		}

		// Get thresholds
		err := db.QueryRow(`SELECT alert_thresholds FROM budgets WHERE scope = ?`, scope).Scan(&thresholds)
		var alertThresholds []int
		if err == nil {
			json.Unmarshal([]byte(thresholds), &alertThresholds)
		}

		status := models.BudgetStatus{
			Scope:           scope,
			MonthlyLimitUSD: limit,
			SpentUSD:        spent,
			RemainingUSD:    limit - spent,
			AlertThresholds: alertThresholds,
		}

		if limit > 0 {
			status.PercentUsed = (spent / limit) * 100
		}

		statuses = append(statuses, status)
	}

	return statuses, rows.Err()
}

// GetLastSyncTime returns when a provider was last synced
func (db *DB) GetLastSyncTime(providerID string) (*time.Time, error) {
	query := `SELECT last_synced_at FROM providers WHERE id = ?`
	var syncTime sql.NullTime
	err := db.QueryRow(query, providerID).Scan(&syncTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if syncTime.Valid {
		return &syncTime.Time, nil
	}
	return nil, nil
}

// UpdateProviderSyncTime updates the last sync timestamp
func (db *DB) UpdateProviderSyncTime(providerID string) error {
	query := `UPDATE providers SET last_synced_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, providerID)
	return err
}

// GetSetting retrieves a setting value
func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	return value, err
}

// SetSetting updates a setting value
func (db *DB) SetSetting(key, value string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	return err
}

// GetProviderAPIKey retrieves a provider's encrypted API key
func (db *DB) GetProviderAPIKey(providerID string) ([]byte, error) {
	var keyEnc []byte
	err := db.QueryRow(`SELECT api_key_enc FROM providers WHERE id = ?`, providerID).Scan(&keyEnc)
	return keyEnc, err
}

// SetProviderAPIKey stores an encrypted API key
func (db *DB) SetProviderAPIKey(providerID string, keyEnc []byte) error {
	_, err := db.Exec(`UPDATE providers SET api_key_enc = ? WHERE id = ?`, keyEnc, providerID)
	return err
}

// GetPricingForModel gets pricing for a model at a given time
func (db *DB) GetPricingForModel(providerID, model string, timestamp time.Time) (*models.Pricing, error) {
	query := `
		SELECT id, provider_id, model, input_price_per_mtok, output_price_per_mtok,
		       cache_read_price_per_mtok, cache_write_price_per_mtok, effective_from
		FROM pricing
		WHERE provider_id = ? AND model = ? AND effective_from <= date(?)
		ORDER BY effective_from DESC
		LIMIT 1
	`

	var pricing models.Pricing
	err := db.QueryRow(query, providerID, model, timestamp).Scan(
		&pricing.ID, &pricing.ProviderID, &pricing.Model,
		&pricing.InputPricePerMtok, &pricing.OutputPricePerMtok,
		&pricing.CacheReadPricePerMtok, &pricing.CacheWritePricePerMtok,
		&pricing.EffectiveFrom,
	)

	return &pricing, err
}

// CountEvents returns total count of events in database
func (db *DB) CountEvents() (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM usage_events`).Scan(&count)
	return count, err
}

// GetEventCount returns count of events for a given month
func (db *DB) GetEventCount(year, month int) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM usage_events
		WHERE strftime('%Y', timestamp) = ? AND strftime('%m', timestamp) = ?
	`
	err := db.QueryRow(query, fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", month)).Scan(&count)
	return count, err
}
