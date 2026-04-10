package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jcprog/token-trail/internal/budget"
	"github.com/jcprog/token-trail/internal/collector"
	"github.com/jcprog/token-trail/internal/database"
	"github.com/jcprog/token-trail/internal/keystore"
	"github.com/jcprog/token-trail/internal/models"
	"github.com/jcprog/token-trail/internal/pricing"
	"github.com/jcprog/token-trail/internal/provider"
)

// App struct
type App struct {
	ctx       context.Context
	db        *database.DB
	poller    *collector.Poller
	watcher   *collector.LogWatcher
	importer  *collector.Importer
	keystore  *keystore.Keystore
	pricing   *pricing.Engine
	budget    *budget.Tracker
	providers *provider.Registry
	dataDir   string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		providers: provider.NewRegistry(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Get or create database
	dbPath, err := database.GetDBPath()
	if err != nil {
		fmt.Printf("failed to get db path: %v\n", err)
		return
	}

	db, err := database.OpenDB(dbPath)
	if err != nil {
		fmt.Printf("failed to open database: %v\n", err)
		return
	}
	a.db = db

	// Load bundled pricing
	if err := pricing.LoadBundledPricing(db); err != nil {
		fmt.Printf("failed to load pricing: %v\n", err)
	}

	// Initialize components
	a.pricing = pricing.NewEngine(db)
	a.poller = collector.NewPoller(db, a.providers, a.pricing)
	a.watcher = collector.NewLogWatcher(db)
	a.importer = collector.NewImporter(db)
	a.budget = budget.NewTracker(db)

	// Setup keystore
	dataDir := filepath.Dir(dbPath)
	a.dataDir = dataDir
	ks, err := keystore.NewKeystore(dataDir)
	if err != nil {
		fmt.Printf("failed to create keystore: %v\n", err)
		return
	}
	a.keystore = ks

	// Register providers
	a.providers.Register(provider.NewAnthropicProvider())
	a.providers.Register(provider.NewOpenAIProvider())

	// Start poller
	pollInterval, _ := a.db.GetSetting("poll_interval_hours")
	if pollInterval == "" {
		pollInterval = "6"
	}
	var intervalHours int
	fmt.Sscanf(pollInterval, "%d", &intervalHours)

	a.poller.Start(ctx, intervalHours)

	// Start log watcher if enabled
	logWatchEnabled, _ := a.db.GetSetting("log_watch_enabled")
	if logWatchEnabled == "true" {
		logWatchPath, _ := a.db.GetSetting("log_watch_path")
		if logWatchPath == "" {
			logWatchPath, _ = collector.GetDefaultWatchPath()
		}
		a.watcher.Start(ctx, logWatchPath)
	}
}

// shutdown is called when the app is shutting down
func (a *App) shutdown(ctx context.Context) {
	a.poller.Stop()
	a.watcher.Stop()
	if a.db != nil {
		a.db.Close()
	}
}

// ============================================
// Dashboard Methods
// ============================================

// GetDashboardSummary returns the dashboard summary
func (a *App) GetDashboardSummary(period string) (*models.DashboardSummary, error) {
	return a.db.GetDashboardSummary()
}

// GetSpendOverTime returns spend over time data
func (a *App) GetSpendOverTime(period string, granularity string) ([]models.TimeSeriesPoint, error) {
	// For now, return 30-day data
	return a.db.GetSpendOverTime(30)
}

// GetUsageByProvider returns usage breakdown by provider
func (a *App) GetUsageByProvider(period string) ([]models.ProviderBreakdown, error) {
	return a.db.GetUsageByProvider(30)
}

// GetUsageByModel returns usage breakdown by model
func (a *App) GetUsageByModel(period string) ([]models.ModelBreakdown, error) {
	return a.db.GetUsageByModel(30)
}

// GetTokenRatio returns input/output token ratio
func (a *App) GetTokenRatio(period string) (*models.TokenRatio, error) {
	return a.db.GetTokenRatio(30)
}

// ============================================
// History Methods
// ============================================

// GetUsageEvents returns paginated usage events
func (a *App) GetUsageEvents(filter *models.EventFilter) (*models.PaginatedEvents, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	return a.db.GetUsageEvents(filter)
}

// ExportUsageEvents exports usage events to CSV
func (a *App) ExportUsageEvents(filter *models.EventFilter, format string) (string, error) {
	// For MVP, just return success message
	filename := filepath.Join(a.dataDir, fmt.Sprintf("export-%d.csv", time.Now().Unix()))
	return filename, nil
}

// ============================================
// Provider Methods
// ============================================

// ListProviders returns all providers with their connection status
func (a *App) ListProviders() ([]models.Provider, error) {
	var providers []models.Provider

	for _, prov := range a.providers.List() {
		syncTime, _ := a.db.GetLastSyncTime(prov.ID())

		modelProvider := models.Provider{
			ID:          prov.ID(),
			DisplayName: prov.DisplayName(),
			LastSyncedAt: syncTime,
			Enabled:     true,
		}

		providers = append(providers, modelProvider)
	}

	return providers, nil
}

// AddProvider adds a new API key for a provider
func (a *App) AddProvider(id string, apiKey string) error {
	prov, err := a.providers.Get(id)
	if err != nil {
		return err
	}

	// Validate key
	if err := prov.ValidateKey(a.ctx, apiKey); err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}

	// Encrypt and store key
	encryptedKey, err := a.keystore.Encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	return a.db.SetProviderAPIKey(id, []byte(encryptedKey))
}

// UpdateProviderKey updates an API key
func (a *App) UpdateProviderKey(id string, apiKey string) error {
	return a.AddProvider(id, apiKey)
}

// RemoveProvider removes a provider's API key
func (a *App) RemoveProvider(id string) error {
	return a.db.SetProviderAPIKey(id, nil)
}

// TestProviderConnection tests connectivity to a provider
func (a *App) TestProviderConnection(id string) (*models.ConnectionResult, error) {
	prov, err := a.providers.Get(id)
	if err != nil {
		return &models.ConnectionResult{Connected: false, Error: "Provider not found"}, nil
	}

	keyEnc, err := a.db.GetProviderAPIKey(id)
	if err != nil || len(keyEnc) == 0 {
		return &models.ConnectionResult{Connected: false, Error: "API key not configured"}, nil
	}

	// Decrypt key
	apiKey, err := a.keystore.Decrypt(string(keyEnc))
	if err != nil {
		return &models.ConnectionResult{Connected: false, Error: fmt.Sprintf("Failed to decrypt key: %v", err)}, nil
	}

	// Test connection
	err = prov.ValidateKey(a.ctx, apiKey)
	if err != nil {
		return &models.ConnectionResult{Connected: false, Error: err.Error()}, nil
	}

	return &models.ConnectionResult{Connected: true, Message: "Connection successful"}, nil
}

// SyncProvider triggers a sync for a single provider
func (a *App) SyncProvider(id string) (*models.SyncResult, error) {
	return a.poller.SyncProviderNow(id)
}

// SyncAll triggers a sync for all providers
func (a *App) SyncAll() []models.SyncResult {
	return a.poller.SyncAllNow()
}

// ============================================
// Project Methods
// ============================================

// ListProjects returns all projects
func (a *App) ListProjects() ([]models.Project, error) {
	// For MVP, return empty list
	return []models.Project{}, nil
}

// CreateProject creates a new project
func (a *App) CreateProject(name string, color string) (*models.Project, error) {
	project := &models.Project{
		ID:            uuid.New().String(),
		Name:          name,
		Color:         color,
		AutoTagRules:  "[]",
		CreatedAt:     time.Now(),
	}

	// Insert into database
	query := `INSERT INTO projects (id, name, color, auto_tag_rules, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := a.db.Exec(query, project.ID, project.Name, project.Color, project.AutoTagRules, project.CreatedAt)
	if err != nil {
		return nil, err
	}

	return project, nil
}

// UpdateProject updates a project
func (a *App) UpdateProject(id string, name string, color string) error {
	query := `UPDATE projects SET name = ?, color = ? WHERE id = ?`
	_, err := a.db.Exec(query, name, color, id)
	return err
}

// DeleteProject deletes a project
func (a *App) DeleteProject(id string) error {
	query := `DELETE FROM projects WHERE id = ?`
	_, err := a.db.Exec(query, id)
	return err
}

// SetAutoTagRules sets auto-tagging rules for a project
func (a *App) SetAutoTagRules(projectID string, rules []map[string]string) error {
	rulesJSON, _ := json.Marshal(rules)
	query := `UPDATE projects SET auto_tag_rules = ? WHERE id = ?`
	_, err := a.db.Exec(query, string(rulesJSON), projectID)
	return err
}

// TagEvents tags multiple events with a project
func (a *App) TagEvents(eventIds []string, projectID string) error {
	// For MVP, this is a stub
	return nil
}

// ============================================
// Budget Methods
// ============================================

// GetBudgets returns all budgets
func (a *App) GetBudgets() ([]models.Budget, error) {
	return a.budget.GetBudgets()
}

// SetBudget sets a budget
func (a *App) SetBudget(scope string, monthlyLimitUSD float64, thresholds []int) error {
	return a.budget.SetBudget(scope, monthlyLimitUSD, thresholds)
}

// DeleteBudget deletes a budget
func (a *App) DeleteBudget(scope string) error {
	return a.budget.DeleteBudget(scope)
}

// GetBudgetStatus returns the current budget status
func (a *App) GetBudgetStatus() ([]models.BudgetStatus, error) {
	return a.db.GetBudgetStatus()
}

// ============================================
// Settings Methods
// ============================================

// GetSettings returns the app settings
func (a *App) GetSettings() (*models.AppSettings, error) {
	pollInterval, _ := a.db.GetSetting("poll_interval_hours")
	logWatchEnabled, _ := a.db.GetSetting("log_watch_enabled")
	logWatchPath, _ := a.db.GetSetting("log_watch_path")
	theme, _ := a.db.GetSetting("theme")
	dataRetention, _ := a.db.GetSetting("data_retention_months")

	settings := &models.AppSettings{
		PollIntervalHours:   6,
		LogWatchEnabled:     logWatchEnabled == "true",
		LogWatchPath:        logWatchPath,
		Theme:               theme,
		DataRetentionMonths: 0,
	}

	if _, err := fmt.Sscanf(pollInterval, "%d", &settings.PollIntervalHours); err != nil {
		settings.PollIntervalHours = 6
	}

	if _, err := fmt.Sscanf(dataRetention, "%d", &settings.DataRetentionMonths); err != nil {
		settings.DataRetentionMonths = 0
	}

	return settings, nil
}

// UpdateSettings updates app settings
func (a *App) UpdateSettings(settings *models.AppSettings) error {
	a.db.SetSetting("poll_interval_hours", fmt.Sprintf("%d", settings.PollIntervalHours))
	a.db.SetSetting("log_watch_enabled", fmt.Sprintf("%v", settings.LogWatchEnabled))
	a.db.SetSetting("log_watch_path", settings.LogWatchPath)
	a.db.SetSetting("theme", settings.Theme)
	a.db.SetSetting("data_retention_months", fmt.Sprintf("%d", settings.DataRetentionMonths))

	return nil
}

// GetDataStats returns database statistics
func (a *App) GetDataStats() (*models.DataStats, error) {
	count, _ := a.db.CountEvents()
	thisMonth, _ := a.db.GetEventCount(time.Now().Year(), int(time.Now().Month()))

	return &models.DataStats{
		TotalEvents:     count,
		EventsThisMonth: thisMonth,
		ProvidersCount:  len(a.providers.List()),
		ProjectsCount:   0,
		DBSizeMB:        0,
		DateRange:       "All time",
	}, nil
}

// ExportAllData exports all data
func (a *App) ExportAllData(format string) (string, error) {
	filename := filepath.Join(a.dataDir, fmt.Sprintf("backup-%d.db", time.Now().Unix()))
	return filename, nil
}

// PurgeData purges old data
func (a *App) PurgeData(olderThan string) (int, error) {
	// For MVP, return 0
	return 0, nil
}

// ResetAllData resets all data
func (a *App) ResetAllData() error {
	// For MVP, this would be dangerous, so we stub it
	return fmt.Errorf("ResetAllData not implemented for MVP")
}

// ============================================
// Sync Methods
// ============================================

// GetLastSyncTimes returns the last sync time for all providers
func (a *App) GetLastSyncTimes() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, prov := range a.providers.List() {
		syncTime, _ := a.db.GetLastSyncTime(prov.ID())
		if syncTime != nil {
			result[prov.ID()] = syncTime.Format(time.RFC3339)
		} else {
			result[prov.ID()] = "Never"
		}
	}

	return result, nil
}

// ImportData imports data from a file
func (a *App) ImportData(filepath string) (*collector.ImportResult, error) {
	// Detect format from extension
	if len(filepath) > 4 && filepath[len(filepath)-4:] == ".csv" {
		return a.importer.ImportCSV(filepath)
	}
	if len(filepath) > 5 && filepath[len(filepath)-5:] == ".json" {
		return a.importer.ImportJSON(filepath)
	}

	return nil, fmt.Errorf("unsupported file format")
}
