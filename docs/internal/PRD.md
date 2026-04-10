# TokenTrail — Product Requirements Document

## Overview

TokenTrail is a lightweight, cross-platform desktop application for individual users to track, visualize, and manage their LLM token usage and costs across multiple providers. It runs locally as a native app with no server, no cloud dependency, and no account required.

---

## Problem Statement

Power users of LLM APIs and subscriptions (developers, researchers, freelancers) interact with multiple providers — Anthropic, OpenAI, and others — through various interfaces: CLI tools (Claude Code, OpenAI CLI), direct API calls, and third-party wrappers. There is no unified, local-first tool to answer basic questions:

- How much have I spent this month across all providers?
- Which model am I burning the most tokens on?
- Am I on track to exceed my budget?
- How does my usage break down by project or session?

Existing options are either provider-specific billing dashboards (requiring login, no cross-provider view), spreadsheet tracking (manual and tedious), or team-oriented SaaS tools (overkill for a single user).

---

## Target User

A single local user who:

- Uses LLM APIs from one or more providers (Anthropic, OpenAI to start)
- Wants visibility into spending without switching between provider dashboards
- Values privacy and local-first tooling
- Runs macOS, Linux, or Windows

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| App framework | Wails 2 (Go backend + native webview) |
| Frontend | Svelte |
| Database | SQLite (single file, embedded via Go) |
| Charting | uPlot |
| Distribution | GitHub Releases (cross-compiled binaries) |

---

## Core Features (MVP)

### 1. Provider API Key Management

- User adds API keys for each provider (Anthropic, OpenAI)
- Keys are stored locally in an encrypted SQLite column or OS keychain
- Keys are never transmitted anywhere except to the provider's own API
- User can add, edit, delete, and test keys (verify connectivity)

### 2. Automated Usage Polling

- App periodically fetches usage data from each provider's billing/usage API
  - **Anthropic**: `/v1/messages/count_tokens` responses + billing API
  - **OpenAI**: `/v1/usage` or `/dashboard/billing/usage` endpoints
- Polling interval is configurable (default: every 6 hours)
- Manual "Sync Now" button for on-demand refresh
- Last sync timestamp displayed per provider

### 3. Local Log Ingestion (Claude Code)

- Option to watch Claude Code's local session logs (`~/.claude/`)
- Parses session files to extract per-message token counts, model used, and timestamps
- Deduplicates against API-fetched data to avoid double counting
- Runs as a background file watcher when enabled

### 4. Manual Import

- Import usage data via CSV or JSON
- Predefined templates for common export formats
- Drag-and-drop or file picker

### 5. Dashboard — Home View

The primary screen the user sees on launch.

**Summary cards (top row):**

- Total spend (current month)
- Total tokens (current month)
- Average daily spend
- Projected month-end spend (linear extrapolation)

**Charts:**

- Spend over time (daily/weekly/monthly toggle) — line or bar chart
- Usage by provider — stacked bar or donut
- Usage by model — horizontal bar (e.g., Claude Sonnet 4 vs GPT-4o vs Claude Haiku)
- Input vs output token ratio — useful for understanding prompt vs completion balance

### 6. History View

- Scrollable table of all recorded usage events
- Columns: timestamp, provider, model, input tokens, output tokens, total tokens, estimated cost, source (API poll / log parse / manual import), project tag (if any)
- Sortable and filterable by provider, model, date range, project
- Export to CSV

### 7. Budget & Alerts

- User sets a monthly budget (global or per-provider)
- Visual budget bar on the dashboard (spent / remaining)
- In-app notification when usage crosses configurable thresholds (e.g., 50%, 80%, 100%)
- Optional system-level desktop notification (via Wails notification API)

### 8. Project Tagging

- User can create project labels (e.g., "pedestrian-cv", "productivity-platform", "freelance-client-A")
- Usage events can be tagged manually or via auto-tagging rules:
  - Claude Code sessions matching a working directory path → auto-tag to a project
  - API calls matching a custom header or metadata field → auto-tag
- Dashboard filters by project
- Per-project spend breakdown view

### 9. Settings

- API key management (add/edit/delete/test)
- Polling interval configuration
- Claude Code log directory path (auto-detected with manual override)
- Budget thresholds
- Data retention period (default: keep forever, option to prune after N months)
- Theme: light / dark / system
- Export all data (full SQLite dump or CSV)
- Reset / clear all data

---

## Data Model

### `providers`

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT (PK) | e.g., "anthropic", "openai" |
| display_name | TEXT | e.g., "Anthropic", "OpenAI" |
| api_key_enc | BLOB | Encrypted API key |
| last_synced_at | DATETIME | Last successful poll |
| enabled | BOOLEAN | Active or paused |

### `usage_events`

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT (PK) | UUID |
| provider_id | TEXT (FK) | Provider reference |
| model | TEXT | e.g., "claude-sonnet-4-20250514" |
| input_tokens | INTEGER | Prompt/input token count |
| output_tokens | INTEGER | Completion/output token count |
| cache_read_tokens | INTEGER | Cached input tokens (if applicable) |
| cache_write_tokens | INTEGER | Cache creation tokens (if applicable) |
| cost_usd | REAL | Calculated cost in USD |
| timestamp | DATETIME | When the usage occurred |
| source | TEXT | "api_poll", "log_parse", "manual_import" |
| project_id | TEXT (FK, nullable) | Optional project tag |
| session_id | TEXT (nullable) | Session grouping (e.g., Claude Code session) |
| metadata | TEXT (JSON) | Flexible extra data |

### `projects`

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT (PK) | UUID |
| name | TEXT | Display name |
| color | TEXT | Hex color for charts |
| auto_tag_rules | TEXT (JSON) | Rules for automatic tagging |
| created_at | DATETIME | |

### `budgets`

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT (PK) | UUID |
| scope | TEXT | "global" or provider_id |
| monthly_limit_usd | REAL | Budget cap |
| alert_thresholds | TEXT (JSON) | e.g., [50, 80, 100] |

### `pricing`

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT (PK) | UUID |
| provider_id | TEXT (FK) | |
| model | TEXT | Model identifier |
| input_price_per_mtok | REAL | Price per 1M input tokens |
| output_price_per_mtok | REAL | Price per 1M output tokens |
| cache_read_price_per_mtok | REAL | Cached input price (if applicable) |
| cache_write_price_per_mtok | REAL | Cache write price (if applicable) |
| effective_from | DATE | When this pricing took effect |

---

## Pricing Data Strategy

Model pricing changes over time. TokenTrail handles this by:

1. Shipping with a bundled `pricing.json` containing current known prices for all supported models
2. The `pricing` table stores historical pricing with `effective_from` dates
3. Cost calculation uses the pricing row that was effective at the time of the usage event
4. User can manually edit pricing if needed (e.g., for custom enterprise rates)
5. Future: optional check for pricing updates from a community-maintained GitHub JSON file

---

## Provider Integration Details

### Anthropic

- **API usage polling**: Use the Admin API's usage endpoint or parse `usage` fields from stored responses
- **Claude Code log parsing**: Read session files from `~/.claude/projects/` and `~/.claude/` — extract model, token counts, timestamps
- **Models to track**: Claude Opus 4, Claude Sonnet 4, Claude Haiku 3.5, and any future models
- **Pricing fields**: input, output, cache_read, cache_write (Anthropic has prompt caching)

### OpenAI

- **API usage polling**: Use the `/v1/organization/usage` endpoint (requires admin-level API key)
- **Models to track**: GPT-4o, GPT-4o-mini, o1, o3, and variants
- **Pricing fields**: input, output (plus reasoning tokens for o-series models)

---

## Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| App binary size | < 15 MB |
| RAM usage at idle | < 50 MB |
| Startup time | < 2 seconds |
| SQLite DB size (1 year heavy use) | < 50 MB |
| Supported OS | macOS (arm64, amd64), Linux (amd64), Windows (amd64) |
| No internet required | App works fully offline with cached data; internet only for syncing |

---

## Out of Scope (MVP)

These are explicitly deferred to post-MVP:

- Multi-user or team features
- Cloud sync or backup
- More providers (Google Gemini, Mistral, Groq, local Ollama) — architecture supports it, just not in v1
- Real-time streaming token counting (intercepting live API calls)
- Browser extension for web-based chat interfaces
- Mobile app
- Automatic pricing update service

---

## Future Considerations

- **Plugin architecture**: Allow community-contributed provider adapters (Go plugin or embedded Wasm)
- **Ollama / local model tracking**: Track local LLM usage for completeness (tokens are free but tracking is useful for benchmarking)
- **API proxy mode**: Optional local proxy that sits between your app and provider APIs, logging every call transparently — the most accurate collection method
- **Claude Code MCP integration**: If Claude Code supports MCP servers for extensions, TokenTrail could register as one

---

## Success Criteria

1. User can install a single binary, launch it, add an API key, and see usage data within 60 seconds
2. Dashboard accurately reflects spend within 5% of provider's own billing page
3. App runs continuously in the background with negligible resource impact
4. Adding a new provider requires only implementing a Go interface — no frontend changes needed

---

## Open Questions

1. **Anthropic usage API access**: What level of API access is needed to pull historical usage? Does a standard API key suffice or is an admin/organization key required?
2. **Claude Code log format stability**: Are the session log files in `~/.claude/` a stable format or subject to change without notice?
3. **OpenAI usage API authentication**: Does the `/v1/organization/usage` endpoint require an organization-level key or does a standard project key work?
4. **OS keychain integration**: Should API keys use the OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service) or is encrypted SQLite sufficient for v1?