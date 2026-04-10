# TokenTrail — Technical Specification

## 1. Project Structure

```
token-trail/
├── CLAUDE.md                     # Claude Code project instructions
├── README.md
├── wails.json                    # Wails project config
├── go.mod
├── go.sum
├── main.go                       # Wails app entry point
├── app.go                        # Core app struct, lifecycle hooks
│
├── internal/
│   ├── database/
│   │   ├── database.go           # SQLite connection, migrations runner
│   │   ├── migrations/
│   │   │   ├── 001_initial.sql   # Tables: providers, usage_events, projects, budgets, pricing
│   │   │   └── ...
│   │   └── queries.go            # Query helpers (sqlc-style, hand-written)
│   │
│   ├── provider/
│   │   ├── provider.go           # Provider interface definition
│   │   ├── anthropic.go          # Anthropic API integration
│   │   ├── openai.go             # OpenAI API integration
│   │   └── registry.go           # Provider registry (lookup by ID)
│   │
│   ├── collector/
│   │   ├── poller.go             # Scheduled API polling (ticker-based)
│   │   ├── logwatcher.go         # File watcher for Claude Code logs
│   │   └── importer.go           # CSV/JSON manual import
│   │
│   ├── pricing/
│   │   ├── pricing.go            # Cost calculation engine
│   │   ├── bundled.go            # Embedded default pricing data
│   │   └── pricing.json          # Bundled pricing (go:embed)
│   │
│   ├── budget/
│   │   └── budget.go             # Budget tracking, threshold checks, alert triggers
│   │
│   ├── keystore/
│   │   └── keystore.go           # API key encryption (AES-256-GCM, machine-bound key)
│   │
│   └── models/
│       └── models.go             # Shared structs: UsageEvent, Provider, Project, Budget, etc.
│
├── frontend/
│   ├── package.json
│   ├── svelte.config.js
│   ├── vite.config.js
│   ├── src/
│   │   ├── main.js               # Svelte app entry
│   │   ├── App.svelte            # Root layout (sidebar + content area)
│   │   ├── lib/
│   │   │   ├── wailsbridge.js    # Typed wrappers around Wails runtime bindings
│   │   │   ├── stores/
│   │   │   │   ├── dashboard.js  # Dashboard data store
│   │   │   │   ├── settings.js   # Settings/preferences store
│   │   │   │   └── providers.js  # Provider state store
│   │   │   ├── utils/
│   │   │   │   ├── format.js     # Token/cost/date formatting helpers
│   │   │   │   └── colors.js     # Chart color palette
│   │   │   └── components/
│   │   │       ├── SummaryCard.svelte
│   │   │       ├── SpendChart.svelte
│   │   │       ├── UsageByProvider.svelte
│   │   │       ├── UsageByModel.svelte
│   │   │       ├── TokenRatioChart.svelte
│   │   │       ├── BudgetBar.svelte
│   │   │       ├── UsageTable.svelte
│   │   │       └── ...
│   │   ├── pages/
│   │   │   ├── Dashboard.svelte
│   │   │   ├── History.svelte
│   │   │   ├── Projects.svelte
│   │   │   └── Settings.svelte
│   │   └── assets/
│   │       └── global.css        # Base styles, CSS variables for theming
│   └── index.html
│
├── docs/
│   ├── internal/
│   │   ├── PRD.md
│   │   ├── TECH_SPEC.md
│   │   └── ARCHITECTURE.md
│   ├── guide/
│   │   ├── getting-started.md
│   │   ├── configuration.md
│   │   └── providers.md
│   └── mkdocs.yml
│
└── build/
    └── appicon.png               # App icon for all platforms
```

---

## 2. Go Backend Architecture

### 2.1 App Struct (app.go)

The central Wails app struct exposes methods to the frontend via Wails bindings.

```go
type App struct {
    ctx       context.Context
    db        *database.DB
    poller    *collector.Poller
    watcher   *collector.LogWatcher
    keystore  *keystore.Keystore
    pricing   *pricing.Engine
    budget    *budget.Tracker
    providers *provider.Registry
}
```

**Lifecycle hooks:**

- `OnStartup(ctx)` — Initialize SQLite, run migrations, start poller, start log watcher (if enabled)
- `OnShutdown()` — Stop poller, close watcher, close DB

### 2.2 Wails-Bound Methods

These are the Go methods exposed to the Svelte frontend. Wails auto-generates TypeScript bindings.

```
Dashboard:
  GetDashboardSummary(period string) → DashboardSummary
  GetSpendOverTime(period string, granularity string) → []TimeSeriesPoint
  GetUsageByProvider(period string) → []ProviderBreakdown
  GetUsageByModel(period string) → []ModelBreakdown
  GetTokenRatio(period string) → TokenRatio

History:
  GetUsageEvents(filter EventFilter) → PaginatedEvents
  ExportUsageEvents(filter EventFilter, format string) → string (file path)

Providers:
  ListProviders() → []ProviderInfo
  AddProvider(id string, apiKey string) → error
  UpdateProviderKey(id string, apiKey string) → error
  RemoveProvider(id string) → error
  TestProviderConnection(id string) → ConnectionResult
  SyncProvider(id string) → SyncResult
  SyncAll() → []SyncResult

Projects:
  ListProjects() → []Project
  CreateProject(name string, color string) → Project
  UpdateProject(id string, name string, color string) → error
  DeleteProject(id string) → error
  SetAutoTagRules(projectId string, rules []AutoTagRule) → error
  TagEvents(eventIds []string, projectId string) → error

Budget:
  GetBudgets() → []Budget
  SetBudget(scope string, monthlyLimitUSD float64, thresholds []int) → error
  DeleteBudget(scope string) → error
  GetBudgetStatus() → []BudgetStatus

Settings:
  GetSettings() → AppSettings
  UpdateSettings(settings AppSettings) → error
  GetDataStats() → DataStats
  ExportAllData(format string) → string (file path)
  PurgeData(olderThan string) → int (rows deleted)
  ResetAllData() → error

Sync:
  GetLastSyncTimes() → map[string]time.Time
```

### 2.3 Provider Interface

```go
type Provider interface {
    // ID returns the unique provider identifier (e.g., "anthropic", "openai")
    ID() string

    // DisplayName returns the human-readable name
    DisplayName() string

    // ValidateKey checks if the API key is valid and has usage access
    ValidateKey(ctx context.Context, apiKey string) error

    // FetchUsage retrieves usage data for the given time range
    // Returns raw usage events that the collector will deduplicate and store
    FetchUsage(ctx context.Context, apiKey string, from time.Time, to time.Time) ([]UsageEvent, error)

    // SupportedModels returns the list of known models for pricing lookup
    SupportedModels() []string
}
```

Each provider implementation lives in its own file. Adding a new provider means:
1. Create `internal/provider/newprovider.go` implementing the interface
2. Register it in `registry.go`
3. Add pricing data to `pricing.json`
4. No frontend changes required

### 2.4 Collector System

#### Poller (`collector/poller.go`)

- Uses a `time.Ticker` with configurable interval (default 6 hours)
- On each tick: iterates enabled providers, calls `FetchUsage`, deduplicates, writes to DB
- Deduplication: hash of (provider + model + timestamp + input_tokens + output_tokens) — skip if hash exists
- Respects rate limits: back off if provider returns 429
- Emits Wails events on sync completion so frontend can refresh

#### Log Watcher (`collector/logwatcher.go`)

- Uses `fsnotify` to watch Claude Code log directories
- Default paths (auto-detected):
  - macOS: `~/.claude/`
  - Linux: `~/.claude/`
  - Windows: `%USERPROFILE%\.claude\`
- On file change: parse new entries, extract model/tokens/timestamp, tag with source "log_parse"
- Deduplicates against API-polled data using timestamp + token count fuzzy matching
- Configurable: can be enabled/disabled, path can be overridden

#### Importer (`collector/importer.go`)

- Accepts CSV or JSON files
- CSV expected columns: `timestamp, provider, model, input_tokens, output_tokens`
- JSON: array of objects with the same fields
- Validates data, assigns source "manual_import", inserts into DB
- Returns import summary (rows imported, rows skipped, errors)

---

## 3. Database Layer

### 3.1 SQLite Configuration

```go
// Connection string flags
// _journal_mode=WAL       — concurrent reads during writes
// _synchronous=NORMAL     — safe with WAL, better performance
// _foreign_keys=ON        — enforce referential integrity
// _busy_timeout=5000      — wait 5s on lock instead of failing
```

DB file location (OS-appropriate app data):
- macOS: `~/Library/Application Support/TokenTrail/tokentrail.db`
- Linux: `~/.local/share/TokenTrail/tokentrail.db`
- Windows: `%APPDATA%\TokenTrail\tokentrail.db`

Use Wails' `environment` package to resolve paths.

### 3.2 Migration 001 — Initial Schema

```sql
CREATE TABLE providers (
    id              TEXT PRIMARY KEY,
    display_name    TEXT NOT NULL,
    api_key_enc     BLOB,
    last_synced_at  DATETIME,
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE projects (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    color           TEXT NOT NULL DEFAULT '#6366f1',
    auto_tag_rules  TEXT DEFAULT '[]',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE usage_events (
    id                  TEXT PRIMARY KEY,
    provider_id         TEXT NOT NULL REFERENCES providers(id),
    model               TEXT NOT NULL,
    input_tokens        INTEGER NOT NULL DEFAULT 0,
    output_tokens       INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens   INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens  INTEGER NOT NULL DEFAULT 0,
    cost_usd            REAL NOT NULL DEFAULT 0.0,
    timestamp           DATETIME NOT NULL,
    source              TEXT NOT NULL CHECK(source IN ('api_poll', 'log_parse', 'manual_import')),
    project_id          TEXT REFERENCES projects(id) ON DELETE SET NULL,
    session_id          TEXT,
    dedup_hash          TEXT UNIQUE,
    metadata            TEXT DEFAULT '{}',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_usage_events_timestamp ON usage_events(timestamp);
CREATE INDEX idx_usage_events_provider ON usage_events(provider_id);
CREATE INDEX idx_usage_events_model ON usage_events(model);
CREATE INDEX idx_usage_events_project ON usage_events(project_id);
CREATE INDEX idx_usage_events_dedup ON usage_events(dedup_hash);

CREATE TABLE budgets (
    id                  TEXT PRIMARY KEY,
    scope               TEXT NOT NULL UNIQUE,
    monthly_limit_usd   REAL NOT NULL,
    alert_thresholds    TEXT NOT NULL DEFAULT '[50, 80, 100]',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pricing (
    id                          TEXT PRIMARY KEY,
    provider_id                 TEXT NOT NULL REFERENCES providers(id),
    model                       TEXT NOT NULL,
    input_price_per_mtok        REAL NOT NULL,
    output_price_per_mtok       REAL NOT NULL,
    cache_read_price_per_mtok   REAL DEFAULT 0.0,
    cache_write_price_per_mtok  REAL DEFAULT 0.0,
    effective_from              DATE NOT NULL,
    UNIQUE(provider_id, model, effective_from)
);

CREATE TABLE settings (
    key     TEXT PRIMARY KEY,
    value   TEXT NOT NULL
);

-- Default settings
INSERT INTO settings (key, value) VALUES
    ('poll_interval_hours', '6'),
    ('log_watch_enabled', 'false'),
    ('log_watch_path', ''),
    ('theme', 'system'),
    ('data_retention_months', '0');

-- Default providers (keys added by user)
INSERT INTO providers (id, display_name, enabled) VALUES
    ('anthropic', 'Anthropic', 1),
    ('openai', 'OpenAI', 1);
```

### 3.3 Key Queries

```sql
-- Dashboard: Monthly spend summary
SELECT
    COALESCE(SUM(cost_usd), 0) AS total_spend,
    COALESCE(SUM(input_tokens + output_tokens), 0) AS total_tokens,
    COALESCE(SUM(cost_usd) / NULLIF(julianday('now') - julianday(date('now', 'start of month')), 0), 0) AS avg_daily_spend
FROM usage_events
WHERE timestamp >= date('now', 'start of month');

-- Dashboard: Spend over time (daily granularity)
SELECT
    date(timestamp) AS day,
    SUM(cost_usd) AS daily_spend,
    SUM(input_tokens) AS daily_input_tokens,
    SUM(output_tokens) AS daily_output_tokens
FROM usage_events
WHERE timestamp >= date('now', '-30 days')
GROUP BY date(timestamp)
ORDER BY day;

-- Dashboard: Usage by provider
SELECT
    provider_id,
    SUM(cost_usd) AS total_cost,
    SUM(input_tokens + output_tokens) AS total_tokens
FROM usage_events
WHERE timestamp >= date('now', 'start of month')
GROUP BY provider_id;

-- Dashboard: Usage by model
SELECT
    model,
    provider_id,
    SUM(cost_usd) AS total_cost,
    SUM(input_tokens) AS input_tokens,
    SUM(output_tokens) AS output_tokens
FROM usage_events
WHERE timestamp >= date('now', 'start of month')
GROUP BY model, provider_id
ORDER BY total_cost DESC;

-- Budget status
SELECT
    b.scope,
    b.monthly_limit_usd,
    COALESCE(SUM(e.cost_usd), 0) AS spent,
    b.monthly_limit_usd - COALESCE(SUM(e.cost_usd), 0) AS remaining
FROM budgets b
LEFT JOIN usage_events e
    ON (b.scope = 'global' OR b.scope = e.provider_id)
    AND e.timestamp >= date('now', 'start of month')
GROUP BY b.id;

-- Cost calculation using time-aware pricing
SELECT
    p.input_price_per_mtok,
    p.output_price_per_mtok,
    p.cache_read_price_per_mtok,
    p.cache_write_price_per_mtok
FROM pricing p
WHERE p.provider_id = ? AND p.model = ?
    AND p.effective_from <= date(?)
ORDER BY p.effective_from DESC
LIMIT 1;

-- History: Paginated with filters
SELECT * FROM usage_events
WHERE (:provider IS NULL OR provider_id = :provider)
    AND (:model IS NULL OR model = :model)
    AND (:project IS NULL OR project_id = :project)
    AND (:from IS NULL OR timestamp >= :from)
    AND (:to IS NULL OR timestamp <= :to)
ORDER BY timestamp DESC
LIMIT :limit OFFSET :offset;
```

---

## 4. Frontend Architecture

### 4.1 Routing

Use `svelte-spa-router` (hash-based routing, no server needed).

```
#/               → Dashboard.svelte
#/history        → History.svelte
#/projects       → Projects.svelte
#/settings       → Settings.svelte
```

### 4.2 Layout

```
┌──────────────────────────────────────────────┐
│  Sidebar (fixed)         │  Content Area      │
│                          │                    │
│  ◉ Dashboard             │  (routed page)     │
│  ◉ History               │                    │
│  ◉ Projects              │                    │
│  ◉ Settings              │                    │
│                          │                    │
│  ─────────────           │                    │
│  Last sync: 2h ago       │                    │
│  [Sync Now]              │                    │
└──────────────────────────────────────────────┘
```

### 4.3 Dashboard Page Layout

```
┌─────────────────────────────────────────────────────────┐
│  [This Month ▾]   [All Providers ▾]                     │
├──────────┬──────────┬──────────┬───────────┤            │
│  Spend   │  Tokens  │ Avg/Day  │ Projected │            │
│  $47.23  │  2.1M    │ $4.72    │ $141.69   │            │
├──────────┴──────────┴──────────┴───────────┤            │
│                                                         │
│  ┌─ Spend Over Time ────────────────────┐               │
│  │  (line/bar chart, daily/weekly toggle)│               │
│  └──────────────────────────────────────┘               │
│                                                         │
│  ┌─ By Provider ───────┐ ┌─ By Model ────────┐         │
│  │  (donut chart)      │ │  (horizontal bar)  │         │
│  └─────────────────────┘ └────────────────────┘         │
│                                                         │
│  ┌─ Input vs Output ───┐ ┌─ Budget ──────────┐         │
│  │  (stacked bar)      │ │  (progress bar)    │         │
│  └─────────────────────┘ └────────────────────┘         │
└─────────────────────────────────────────────────────────┘
```

### 4.4 Theming

CSS custom properties for light/dark/system themes.

```css
:root {
    --bg-primary: #ffffff;
    --bg-secondary: #f8fafc;
    --bg-sidebar: #1e293b;
    --text-primary: #0f172a;
    --text-secondary: #64748b;
    --text-sidebar: #e2e8f0;
    --border: #e2e8f0;
    --accent: #6366f1;
    --accent-hover: #4f46e5;
    --success: #22c55e;
    --warning: #f59e0b;
    --danger: #ef4444;
    --chart-1: #6366f1;
    --chart-2: #06b6d4;
    --chart-3: #f59e0b;
    --chart-4: #ef4444;
    --chart-5: #22c55e;
}

[data-theme="dark"] {
    --bg-primary: #0f172a;
    --bg-secondary: #1e293b;
    --bg-sidebar: #0f172a;
    --text-primary: #f1f5f9;
    --text-secondary: #94a3b8;
    --border: #334155;
}
```

### 4.5 Wails Bridge

Wails generates TypeScript bindings from Go methods. The bridge module wraps these with error handling.

```javascript
// lib/wailsbridge.js
import * as App from '../../wailsjs/go/main/App';

export async function getDashboardSummary(period) {
    try {
        return await App.GetDashboardSummary(period);
    } catch (err) {
        console.error('Failed to fetch dashboard summary:', err);
        throw err;
    }
}

// ... etc for all bound methods
```

### 4.6 Charts (uPlot)

uPlot configuration pattern:

```javascript
// lib/components/SpendChart.svelte
import uPlot from 'uplot';

// Data format: [timestamps[], values[]]
// uPlot expects unix seconds for timestamps
const data = [
    timestamps.map(t => new Date(t).getTime() / 1000),
    spendValues
];

const opts = {
    width: containerWidth,
    height: 300,
    scales: { x: { time: true }, y: { auto: true } },
    axes: [
        { stroke: 'var(--text-secondary)', grid: { stroke: 'var(--border)' } },
        { stroke: 'var(--text-secondary)', grid: { stroke: 'var(--border)' },
          values: (u, vals) => vals.map(v => '$' + v.toFixed(2)) }
    ],
    series: [
        {},
        { stroke: 'var(--accent)', fill: 'rgba(99, 102, 241, 0.1)', width: 2 }
    ]
};
```

---

## 5. API Key Security

### 5.1 Encryption

- AES-256-GCM encryption for stored API keys
- Encryption key derived from machine-specific data (machine ID + app-specific salt) using PBKDF2
- On first launch, generate and store a random salt in the app data directory
- Keys are decrypted in-memory only when needed for API calls, never held longer than necessary

```go
// internal/keystore/keystore.go
type Keystore struct {
    encryptionKey []byte
}

func (k *Keystore) Encrypt(plaintext string) ([]byte, error)
func (k *Keystore) Decrypt(ciphertext []byte) (string, error)
```

### 5.2 Threat Model (v1)

This is a personal desktop app, not a server. The threat model is:

- **Protected against**: casual file browsing revealing raw API keys
- **Not protected against**: determined attacker with access to the machine and app binary (they could extract the key derivation logic)
- **Acceptable for v1**: full OS keychain integration is a post-MVP enhancement

---

## 6. Wails Event System

The Go backend emits events that the Svelte frontend listens to for real-time updates.

```go
// Events emitted by backend
const (
    EventSyncStarted   = "sync:started"    // payload: provider_id
    EventSyncCompleted = "sync:completed"   // payload: {provider_id, events_added, errors}
    EventSyncFailed    = "sync:failed"      // payload: {provider_id, error}
    EventBudgetAlert   = "budget:alert"     // payload: {scope, percent_used, threshold}
    EventLogParsed     = "log:parsed"       // payload: {events_added, source_file}
)
```

Frontend listens:

```javascript
import { EventsOn } from '../../wailsjs/runtime/runtime';

EventsOn('sync:completed', (data) => {
    // refresh dashboard data
    refreshDashboard();
    showToast(`Synced ${data.events_added} events from ${data.provider_id}`);
});

EventsOn('budget:alert', (data) => {
    showNotification(`${data.scope} budget: ${data.percent_used}% used`);
});
```

---

## 7. Build & Distribution

### 7.1 Development

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Dev mode (hot reload frontend + Go rebuild)
wails dev

# Build for current platform
wails build
```

### 7.2 Cross-Platform Build (GitHub Actions)

```yaml
# .github/workflows/release.yml
strategy:
  matrix:
    include:
      - os: macos-latest
        platform: darwin/universal
      - os: ubuntu-latest
        platform: linux/amd64
      - os: windows-latest
        platform: windows/amd64

steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5
    with:
      go-version: '1.22'
  - uses: actions/setup-node@v4
    with:
      node-version: '20'
  - run: go install github.com/wailsapp/wails/v2/cmd/wails@latest
  - run: wails build -platform ${{ matrix.platform }}
  - uses: softprops/action-gh-release@v2
    with:
      files: build/bin/*
```

### 7.3 Output Artifacts

| Platform | Output | Expected Size |
|----------|--------|---------------|
| macOS | `TokenTrail.app` (universal binary) | ~12 MB |
| Linux | `TokenTrail` (ELF binary) | ~10 MB |
| Windows | `TokenTrail.exe` | ~12 MB |

---

## 8. Implementation Phases

### Phase 1 — Skeleton (Week 1)

1. Initialize Wails project with Svelte frontend
2. Set up SQLite with migration system
3. Create provider interface and registry
4. Implement settings store
5. Build app shell: sidebar navigation, routing, theme toggle
6. Wire up Wails bindings for settings CRUD

### Phase 2 — Data Collection (Week 2)

1. Implement Anthropic provider (API key validation + usage fetching)
2. Implement OpenAI provider (API key validation + usage fetching)
3. Build poller with configurable interval
4. Build manual CSV/JSON importer
5. Implement pricing engine with bundled pricing.json
6. Implement keystore encryption
7. Build Settings page UI (provider management, key add/test/remove)

### Phase 3 — Dashboard (Week 3)

1. Implement all dashboard query methods in Go
2. Build summary cards component
3. Build spend-over-time chart (uPlot)
4. Build usage-by-provider donut chart
5. Build usage-by-model horizontal bar chart
6. Build input/output token ratio chart
7. Add period selector and provider filter
8. Wire up Wails events for live refresh after sync

### Phase 4 — History, Projects, Budget (Week 4)

1. Build History page: paginated table with sort/filter
2. Implement CSV export
3. Build Projects page: CRUD, color picker, auto-tag rule editor
4. Build Budget page: set limits, threshold config
5. Implement budget status checks and alert events
6. Add desktop notifications via Wails

### Phase 5 — Claude Code Logs (Week 5)

1. Research and document Claude Code log file format
2. Implement log parser
3. Implement file watcher with fsnotify
4. Deduplication logic against API-polled data
5. Auto-tagging by working directory → project mapping
6. Settings UI for log watch path config

### Phase 6 — Polish & Release (Week 6)

1. Light/dark theme refinement
2. Empty states for all views
3. Error handling and user-friendly error messages
4. Loading states and skeleton screens
5. App icon and branding
6. GitHub Actions CI/CD pipeline
7. README with screenshots
8. First GitHub release

---

## 9. Go Dependencies

```
github.com/wailsapp/wails/v2          # App framework
github.com/mattn/go-sqlite3            # SQLite driver (CGO)
github.com/google/uuid                 # UUID generation
github.com/fsnotify/fsnotify           # File watching
github.com/denisbrodbeck/machineid     # Machine-unique ID for key derivation
golang.org/x/crypto                    # AES-GCM, PBKDF2
```

### Frontend Dependencies

```
uplot                                  # Charting
svelte-spa-router                      # Client-side routing
```

---

## 10. Testing Strategy

**Go backend:**

- Unit tests for pricing calculation (given tokens + model + date → expected cost)
- Unit tests for deduplication hash generation
- Unit tests for each provider's response parsing (mock HTTP responses)
- Integration tests for database queries (use in-memory SQLite)
- Integration tests for importer (sample CSV/JSON fixtures)

**Frontend:**

- Manual testing during development (Wails dev mode)
- Snapshot tests for component rendering (optional, post-MVP)

**Test commands:**

```bash
# Go tests
go test ./internal/... -v

# Frontend (if added)
cd frontend && npm test
```