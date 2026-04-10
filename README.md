# TokenTrail

**Track LLM token usage and costs across providers — locally, for you.**

A lightweight, cross-platform desktop application for monitoring and visualizing your usage and spending across Anthropic Claude, OpenAI, and other LLM providers. No server, no cloud dependency, no account required.

## ✨ Features

### 📊 Dashboard
- **Real-time metrics**: Total spend, token counts, daily averages, month-end projections
- **Multi-provider view**: Consolidated spending across all your API keys
- **Model breakdown**: See which models cost the most
- **Input/output ratio**: Understand your prompt vs. completion token usage
- **Live updates**: Auto-refreshes after syncs via Wails events

### 📋 Usage History
- **Paginated table**: All usage events with sortable columns
- **Filters**: By provider, model, project, date range
- **Export**: Download events as CSV
- **Sources**: Track whether data came from API polling, log parsing, or manual import

### 💰 Budget Tracking
- **Global & per-provider budgets**: Set spending limits
- **Threshold alerts**: Notifications at 50%, 80%, 100% of budget
- **Visual progress bar**: See remaining budget at a glance

### 📁 Projects
- **Organize by project**: Tag usage events by project (e.g., "client-A", "research")
- **Auto-tagging rules**: Automatically categorize events by directory or metadata
- **Per-project reports**: Break down spend by project

### ⚙️ Settings
- **API key management**: Securely store and manage keys for each provider
- **Test connectivity**: Verify your API keys work before saving
- **Log watching**: (Optional) Monitor `~/.claude/` for Claude Code sessions
- **Polling interval**: Customize how often to fetch data (default: 6 hours)
- **Data retention**: Configure how long to keep historical data

## 🚀 Quick Start

### Prerequisites
- **Go 1.22+** (for building from source)
- **Node.js 20+** (for frontend dependencies)
- **Wails v2** CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/jcprog/token-trail.git
   cd token-trail
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   cd frontend && npm install && cd ..
   ```

3. **Run in development** (with hot reload):
   ```bash
   wails dev
   ```

4. **Build for your platform**:
   ```bash
   wails build
   ```

### First Launch
1. Open the app and go to **Settings**
2. Add your **Anthropic API key** (or OpenAI key)
3. Click **Test** to verify connectivity
4. Click **Dashboard** → **Sync Now** to fetch initial data
5. Watch your usage and spending appear in real-time

## 🏗️ Architecture

### Backend (Go)
- **Database**: SQLite with WAL mode, migrations, and hand-written queries
- **Providers**: Pluggable provider interface (Anthropic, OpenAI, extensible for more)
- **Collector**: Scheduled poller, log watcher, and CSV/JSON importer
- **Security**: AES-256-GCM encryption for API keys, PBKDF2 key derivation from machine ID
- **Pricing**: Time-aware cost calculation with bundled pricing data

### Frontend (Svelte)
- **Routing**: Hash-based SPA using `svelte-spa-router`
- **Stores**: Reactive state management for dashboard, settings, and providers
- **Styling**: CSS custom properties for light/dark theming
- **Charts**: (Future) uPlot for spend trends and breakdowns

### Wails IPC
- **40+ bound methods**: Dashboard queries, provider management, data import/export
- **Event system**: Real-time notifications for syncs and budget alerts

## 📂 Project Structure

```
token-trail/
├── main.go                    # Wails entry point
├── app.go                     # App struct + Wails-bound methods
├── go.mod, go.sum            # Go dependencies
├── wails.json                 # Wails config
│
├── internal/
│   ├── models/               # Data structures
│   ├── database/             # SQLite connection, migrations, queries
│   ├── provider/             # Provider interface + implementations
│   ├── collector/            # Poller, log watcher, importer
│   ├── pricing/              # Cost calculation, bundled pricing
│   ├── budget/               # Budget tracking
│   └── keystore/             # API key encryption
│
├── frontend/
│   ├── package.json          # npm dependencies
│   ├── svelte.config.js      # Svelte config
│   ├── vite.config.js        # Vite bundler config
│   ├── index.html
│   └── src/
│       ├── main.js           # Entry point
│       ├── App.svelte        # Root layout + router
│       ├── assets/global.css # Theming + styles
│       ├── lib/
│       │   ├── wailsbridge.js      # Wails binding wrappers
│       │   ├── stores/             # Svelte stores
│       │   ├── utils/              # Format, colors helpers
│       │   └── components/         # Reusable components
│       └── pages/
│           ├── Dashboard.svelte
│           ├── History.svelte
│           ├── Projects.svelte
│           └── Settings.svelte
│
├── .github/workflows/
│   └── release.yml           # GitHub Actions: cross-platform builds
│
└── docs/
    ├── internal/
    │   ├── PRD.md            # Product requirements
    │   └── TECH_SPECS.md     # Technical specifications
    └── guide/                # End-user documentation (planned)
```

## 🔐 Security

- **API keys encrypted at rest** using AES-256-GCM
- **Machine-bound encryption**: Key derived from machine ID + app salt via PBKDF2
- **No internet required**: All data stays local; internet only for API polling
- **No telemetry**: Your usage data never leaves your machine

## 📊 Supported Providers (MVP)

| Provider | Status | Features |
|----------|--------|----------|
| **Anthropic** | ✅ MVP | API polling, Claude Code log parsing, prompt caching pricing |
| **OpenAI** | ✅ MVP | API polling, reasoning token support |
| **Gemini, Mistral, Groq, Ollama** | 🔄 Post-MVP | Architecture supports easy addition |

## 🛠️ Development

### Commands

```bash
# Development with hot reload
wails dev

# Build for current platform
wails build

# Cross-platform builds (for CI/CD)
wails build -platform darwin/universal   # macOS
wails build -platform linux/amd64        # Linux
wails build -platform windows/amd64      # Windows

# Run tests
go test ./internal/... -v

# Check Go builds
go build ./...
```

### Database

- **Location**: OS-appropriate app data directory
  - macOS: `~/Library/Application Support/TokenTrail/tokentrail.db`
  - Linux: `~/.local/share/TokenTrail/tokentrail.db`
  - Windows: `%APPDATA%\TokenTrail\tokentrail.db`

- **Migrations**: Sequential SQL files in `internal/database/migrations/`
- **Queries**: Hand-written helpers in `internal/database/queries.go`

### Adding a New Provider

1. Create `internal/provider/myprovider.go`
2. Implement the `Provider` interface:
   ```go
   type Provider interface {
       ID() string
       DisplayName() string
       ValidateKey(ctx context.Context, apiKey string) error
       FetchUsage(ctx context.Context, apiKey string, from, to interface{}) ([]UsageEvent, error)
       SupportedModels() []string
   }
   ```
3. Register in `app.go`:
   ```go
   a.providers.Register(provider.NewMyProvider())
   ```
4. Add pricing data to `internal/pricing/pricing.json`
5. No frontend changes needed!

## 📈 Roadmap

### Phase 1-3 (MVP) ✅
- Project skeleton, database, providers
- Dashboard with metrics and charts
- History, projects, budget tracking
- Settings and API key management

### Phase 4-6 (Post-MVP)
- Claude Code log parsing (JSONL format)
- Advanced theming and customization
- Per-project budgets and alerts
- Ollama and local model tracking
- Browser extension for web chat
- Team/multi-user features (deferred)

## 📝 License

MIT License — See LICENSE file for details.

## 🤝 Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## 💬 Questions?

See the [technical specifications](docs/internal/TECH_SPECS.md) and [product requirements](docs/internal/PRD.md) for detailed documentation.

---

**Built with ❤️ using Wails v2, Go, and Svelte.**