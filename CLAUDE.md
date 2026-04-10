# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TokenTrail is a cross-platform desktop application (Wails v2: Go + Svelte) for tracking, visualizing, and managing LLM token usage and costs across multiple providers (Anthropic, OpenAI). Local-first, no server or account required.

See `docs/internal/PRD.md` for product vision and `docs/internal/TECH_SPECS.md` for detailed architecture.

## Commands

### Setup (one-time)
```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# From repository root:
# Go dependencies are declared in go.mod (created during first implementation)
go mod download

# Frontend dependencies
cd frontend
npm install
```

### Development
```bash
# Hot-reload frontend + auto-rebuild Go backend
wails dev

# Run Go tests
go test ./internal/... -v

# Run single test
go test ./internal/provider -run TestProviderRegistry -v
```

### Build
```bash
# Build for current platform
wails build

# Cross-platform builds (used in CI)
wails build -platform darwin/universal  # macOS universal binary
wails build -platform linux/amd64
wails build -platform windows/amd64
```

## Architecture

### High-Level Model (Wails v2)

```
┌─────────────────────────────────────────┐
│        Svelte SPA Frontend (webview)     │
│  Hash router, uPlot charts, Svelte store│
│  Calls: App.GetDashboardSummary(), etc. │
│  Listens: EventsOn('sync:completed')    │
└───────────────┬─────────────────────────┘
                │ Wails IPC bindings
┌───────────────▼─────────────────────────┐
│     Go Backend (app.go / App struct)    │
│                                         │
│  ┌─────────────┐ ┌────────────────┐    │
│  │  Collector  │ │    Provider    │    │
│  │  Poller     │ │    Registry    │    │
│  │  LogWatcher │ │  Anthropic     │    │
│  │  Importer   │ │  OpenAI        │    │
│  └─────┬───────┘ └────────┬───────┘    │
│        │                  │             │
│        └──────────┬───────┘             │
│                   │                     │
│        ┌──────────▼────────────┐        │
│        │  SQLite Database      │        │
│        │  (WAL mode)           │        │
│        │  Migrations in        │        │
│        │  internal/database/   │        │
│        │  migrations/          │        │
│        └───────────────────────┘        │
│                                         │
│  ┌─────────────┐ ┌─────────────┐       │
│  │   Pricing   │ │  Keystore   │       │
│  │   Engine    │ │ (AES-256GCM)│       │
│  └─────────────┘ └─────────────┘       │
└─────────────────────────────────────────┘
```

### Key Packages

| Package | Role |
|---------|------|
| `app.go` | Central `App` struct; lifecycle hooks (`OnStartup`, `OnShutdown`); all Wails-bound methods |
| `internal/database/` | SQLite connection, migration runner, query helpers |
| `internal/database/migrations/` | Sequential `.sql` migration files (001_initial.sql, etc.) |
| `internal/provider/` | `Provider` interface + per-provider implementations; registry pattern |
| `internal/collector/` | Poller (ticker-based), LogWatcher (fsnotify), Importer (CSV/JSON) |
| `internal/pricing/` | Cost calculation; bundled pricing data via `go:embed` |
| `internal/budget/` | Budget tracking, threshold checks, alert event emission |
| `internal/keystore/` | AES-256-GCM encryption; machine-bound key derivation (PBKDF2) |
| `internal/models/` | Shared structs: `UsageEvent`, `Provider`, `Project`, `Budget`, etc. |
| `frontend/src/` | Svelte SPA entry point, pages, components, stores |
| `frontend/src/lib/wailsbridge.js` | Typed wrappers around auto-generated Wails bindings |
| `frontend/src/lib/stores/` | Svelte stores: dashboard, settings, providers |
| `frontend/src/pages/` | Dashboard, History, Projects, Settings pages |
| `frontend/src/lib/components/` | Reusable: SummaryCard, SpendChart, BudgetBar, UsageTable, etc. |

### Wails Communication

- Go methods marked `func (app *App) MethodName() Type {}` auto-generate TypeScript stubs in `frontend/wails/go_models.ts`
- Frontend calls via auto-generated bindings; all communication is bidirectional IPC
- Backend emits named events: `EventsEmit("sync:started")`, `EventsEmit("sync:completed")`, `EventsEmit("sync:failed")`, `EventsEmit("budget:alert")`, `EventsEmit("log:parsed")`
- Frontend subscribes via `EventsOn('sync:completed', callback)` for live UI updates

### Database (SQLite)

- Single file: OS-conventional paths
  - macOS: `~/Library/Application Support/TokenTrail/tokentrail.db`
  - Linux: `~/.local/share/TokenTrail/tokentrail.db`
  - Windows: `%APPDATA%\TokenTrail\tokentrail.db`
- Configuration: WAL journal mode, `NORMAL` synchronous, foreign keys ON, 5s busy timeout
- Migrations are sequential SQL files in `internal/database/migrations/` (001_initial.sql, etc.)
- No ORM — hand-written query helpers in `internal/database/`

## Key Conventions

### Provider Pattern
All providers implement a common Go interface. Adding a provider:
1. Create new file in `internal/provider/` (e.g., `newprovider.go`)
2. Implement the `Provider` interface
3. Register in the provider registry
4. Add pricing data to bundled `pricing.json`
5. No frontend changes needed

### Usage Events
All events carry a `source` field:
- `"api_poll"` — fetched from provider API
- `"log_parse"` — parsed from Claude Code logs (~/.claude/)
- `"manual_import"` — user-imported CSV/JSON

### Deduplication
Events are deduplicated by hashing `(provider + model + timestamp + input_tokens + output_tokens)`. The hash is stored as a `UNIQUE` constraint (`dedup_hash` column).

### Time-Aware Pricing
Cost is calculated using the pricing row with the highest `effective_from` date that is still ≤ the event's timestamp. This allows retroactive cost recalculation if pricing changes.

### Keystore (AES-256-GCM)
- API keys encrypted at rest
- Encryption key derived from machine-specific data (machine ID + app salt) via PBKDF2
- Full OS keychain integration is deferred post-MVP

### Wails Event Names
Backend emits these events; frontend subscribes with `EventsOn()`:
- `sync:started` — data collection started
- `sync:completed` — data collection succeeded
- `sync:failed` — data collection failed
- `budget:alert` — budget threshold crossed
- `log:parsed` — Claude Code log parsed

### Theme
- CSS custom properties for theming (`--bg-primary`, `--accent`, etc.)
- Light/dark/system theme support via `[data-theme="dark"]` override block

### Claude Code Log Watching
- Watches `~/.claude/` (all platforms; `%USERPROFILE%\.claude\` on Windows)
- Uses fsnotify for file watching
- Deduplicates against API-polled data via fuzzy timestamp + token-count matching

## Implementation Phasing
See `docs/internal/TECH_SPECS.md` for the 6-week implementation roadmap:
1. Skeleton (Wails setup, DB schema, basic UI)
2. Data collection (APIs, deduplication)
3. Dashboard (core visualization)
4. History, Projects, Budgets
5. Claude Code log integration
6. Polish and release

## Binary Size & Distribution
Expected ~10-12 MB per platform. Distributed via GitHub Releases with GitHub Actions matrix builds.
