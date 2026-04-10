package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jcprog/token-trail/internal/database"
	"github.com/jcprog/token-trail/internal/models"
	"github.com/jcprog/token-trail/internal/pricing"
	"github.com/jcprog/token-trail/internal/provider"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Poller struct {
	db        *database.DB
	registry  *provider.Registry
	pricing   *pricing.Engine
	interval  time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
	ticker    *time.Ticker
	mu        sync.Mutex
	isRunning bool
}

// NewPoller creates a new poller
func NewPoller(db *database.DB, registry *provider.Registry, pricing *pricing.Engine) *Poller {
	return &Poller{
		db:       db,
		registry: registry,
		pricing:  pricing,
		interval: 6 * time.Hour, // default polling interval
	}
}

// Start starts the polling ticker
func (p *Poller) Start(appCtx context.Context, intervalHours int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("poller already running")
	}

	p.ctx, p.cancel = context.WithCancel(appCtx)
	p.interval = time.Duration(intervalHours) * time.Hour

	go p.run()
	p.isRunning = true
	return nil
}

// Stop stops the polling ticker
func (p *Poller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return
	}

	p.cancel()
	if p.ticker != nil {
		p.ticker.Stop()
	}
	p.isRunning = false
}

// run executes the polling loop
func (p *Poller) run() {
	p.ticker = time.NewTicker(p.interval)
	defer p.ticker.Stop()

	// Poll immediately on start
	p.pollOnce()

	// Then poll at intervals
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.ticker.C:
			p.pollOnce()
		}
	}
}

// pollOnce polls all enabled providers
func (p *Poller) pollOnce() {
	// Emit sync started event
	runtime.EventsEmit(p.ctx, "sync:started")

	providers := p.registry.List()
	for _, prov := range providers {
		p.pollProvider(prov)
	}

	// Emit sync completed event
	runtime.EventsEmit(p.ctx, "sync:completed", map[string]interface{}{
		"timestamp": time.Now(),
	})
}

// pollProvider polls a single provider
func (p *Poller) pollProvider(prov provider.Provider) {
	// Get API key
	keyEnc, err := p.db.GetProviderAPIKey(prov.ID())
	if err != nil || len(keyEnc) == 0 {
		runtime.EventsEmit(p.ctx, "sync:failed", map[string]interface{}{
			"provider_id": prov.ID(),
			"error":       "API key not configured",
		})
		return
	}

	// For now, since providers return empty slices, we just update the sync time
	now := time.Now()
	err = p.db.UpdateProviderSyncTime(prov.ID())
	if err != nil {
		runtime.EventsEmit(p.ctx, "sync:failed", map[string]interface{}{
			"provider_id": prov.ID(),
			"error":       err.Error(),
		})
		return
	}

	runtime.EventsEmit(p.ctx, "sync:completed", map[string]interface{}{
		"provider_id":   prov.ID(),
		"events_added":  0,
		"last_synced":   now,
	})
}

// SyncProviderNow triggers an immediate sync for a provider
func (p *Poller) SyncProviderNow(providerID string) (*models.SyncResult, error) {
	prov, err := p.registry.Get(providerID)
	if err != nil {
		return nil, err
	}

	result := &models.SyncResult{
		ProviderID:  providerID,
		DisplayName: prov.DisplayName(),
		Success:     true,
	}

	// Update sync time
	if err := p.db.UpdateProviderSyncTime(providerID); err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	return result, nil
}

// SyncAllNow triggers an immediate sync for all providers
func (p *Poller) SyncAllNow() []models.SyncResult {
	var results []models.SyncResult
	providers := p.registry.List()

	for _, prov := range providers {
		result, _ := p.SyncProviderNow(prov.ID())
		if result != nil {
			results = append(results, *result)
		}
	}

	return results
}
