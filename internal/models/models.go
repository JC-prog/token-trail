package models

import "time"

// UsageEvent represents a single LLM usage event
type UsageEvent struct {
	ID                 string    `json:"id"`
	ProviderID         string    `json:"provider_id"`
	Model              string    `json:"model"`
	InputTokens        int       `json:"input_tokens"`
	OutputTokens       int       `json:"output_tokens"`
	CacheReadTokens    int       `json:"cache_read_tokens"`
	CacheWriteTokens   int       `json:"cache_write_tokens"`
	CostUSD            float64   `json:"cost_usd"`
	Timestamp          time.Time `json:"timestamp"`
	Source             string    `json:"source"` // "api_poll", "log_parse", "manual_import"
	ProjectID          *string   `json:"project_id"`
	SessionID          *string   `json:"session_id"`
	DedupHash          string    `json:"dedup_hash"`
	Metadata           string    `json:"metadata"` // JSON string
	CreatedAt          time.Time `json:"created_at"`
}

// Provider represents an API provider (Anthropic, OpenAI, etc.)
type Provider struct {
	ID            string     `json:"id"`
	DisplayName   string     `json:"display_name"`
	LastSyncedAt  *time.Time `json:"last_synced_at"`
	Enabled       bool       `json:"enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	APIKeyEncoded string     `json:"api_key_encoded,omitempty"` // Only in UI responses
}

// Project represents a user-created project tag
type Project struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Color         string    `json:"color"` // hex color
	AutoTagRules  string    `json:"auto_tag_rules"` // JSON string
	CreatedAt     time.Time `json:"created_at"`
}

// Budget represents a spending limit
type Budget struct {
	ID                 string `json:"id"`
	Scope              string `json:"scope"` // "global" or provider_id
	MonthlyLimitUSD    float64 `json:"monthly_limit_usd"`
	AlertThresholds    string `json:"alert_thresholds"` // JSON array of ints [50, 80, 100]
	CreatedAt          time.Time `json:"created_at"`
}

// BudgetStatus represents current budget usage
type BudgetStatus struct {
	Scope             string  `json:"scope"`
	MonthlyLimitUSD   float64 `json:"monthly_limit_usd"`
	SpentUSD          float64 `json:"spent_usd"`
	RemainingUSD      float64 `json:"remaining_usd"`
	PercentUsed       float64 `json:"percent_used"`
	AlertThresholds   []int   `json:"alert_thresholds"`
}

// Pricing represents model pricing
type Pricing struct {
	ID                      string    `json:"id"`
	ProviderID              string    `json:"provider_id"`
	Model                   string    `json:"model"`
	InputPricePerMtok       float64   `json:"input_price_per_mtok"`
	OutputPricePerMtok      float64   `json:"output_price_per_mtok"`
	CacheReadPricePerMtok   float64   `json:"cache_read_price_per_mtok"`
	CacheWritePricePerMtok  float64   `json:"cache_write_price_per_mtok"`
	EffectiveFrom           time.Time `json:"effective_from"`
}

// DashboardSummary contains aggregated dashboard metrics
type DashboardSummary struct {
	TotalSpend      float64 `json:"total_spend"`
	TotalTokens     int64   `json:"total_tokens"`
	AvgDailySpend   float64 `json:"avg_daily_spend"`
	ProjectedSpend  float64 `json:"projected_spend"`
	DaysInMonth     int     `json:"days_in_month"`
	CurrentDay      int     `json:"current_day"`
}

// TimeSeriesPoint represents a single point in a time series
type TimeSeriesPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	Spend          float64   `json:"spend"`
	InputTokens    int64     `json:"input_tokens"`
	OutputTokens   int64     `json:"output_tokens"`
	TotalTokens    int64     `json:"total_tokens"`
}

// ProviderBreakdown represents spend breakdown by provider
type ProviderBreakdown struct {
	ProviderID   string  `json:"provider_id"`
	DisplayName  string  `json:"display_name"`
	TotalCost    float64 `json:"total_cost"`
	TotalTokens  int64   `json:"total_tokens"`
	Percentage   float64 `json:"percentage"`
}

// ModelBreakdown represents spend breakdown by model
type ModelBreakdown struct {
	Model           string  `json:"model"`
	ProviderID      string  `json:"provider_id"`
	DisplayName     string  `json:"display_name"`
	TotalCost       float64 `json:"total_cost"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	TotalTokens     int64   `json:"total_tokens"`
	Percentage      float64 `json:"percentage"`
}

// TokenRatio represents input vs output token breakdown
type TokenRatio struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Ratio        float64 `json:"ratio"` // output / input
}

// EventFilter represents query filters for usage events
type EventFilter struct {
	Provider  *string    `json:"provider"`
	Model     *string    `json:"model"`
	Project   *string    `json:"project"`
	FromDate  *time.Time `json:"from_date"`
	ToDate    *time.Time `json:"to_date"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// PaginatedEvents represents paginated usage event results
type PaginatedEvents struct {
	Events      []UsageEvent `json:"events"`
	Total       int          `json:"total"`
	Limit       int          `json:"limit"`
	Offset      int          `json:"offset"`
}

// SyncResult represents the result of a provider sync
type SyncResult struct {
	ProviderID   string `json:"provider_id"`
	DisplayName  string `json:"display_name"`
	Success      bool   `json:"success"`
	EventsAdded  int    `json:"events_added"`
	EventsSkipped int   `json:"events_skipped"`
	Error        string `json:"error,omitempty"`
}

// ConnectionResult represents API connection test result
type ConnectionResult struct {
	Connected bool   `json:"connected"`
	Error     string `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
}

// AppSettings represents user preferences
type AppSettings struct {
	PollIntervalHours  int    `json:"poll_interval_hours"`
	LogWatchEnabled    bool   `json:"log_watch_enabled"`
	LogWatchPath       string `json:"log_watch_path"`
	Theme              string `json:"theme"` // "light", "dark", "system"
	DataRetentionMonths int   `json:"data_retention_months"`
}

// DataStats represents database statistics
type DataStats struct {
	TotalEvents    int       `json:"total_events"`
	EventsThisMonth int      `json:"events_this_month"`
	ProvidersCount int       `json:"providers_count"`
	ProjectsCount  int       `json:"projects_count"`
	DBSizeMB       float64   `json:"db_size_mb"`
	DateRange      string    `json:"date_range"`
}

// AutoTagRule represents a rule for auto-tagging events
type AutoTagRule struct {
	Type  string `json:"type"` // "path", "header", etc.
	Value string `json:"value"`
}
