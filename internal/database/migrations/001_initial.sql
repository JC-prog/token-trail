-- 001_initial.sql - TokenTrail initial schema

CREATE TABLE IF NOT EXISTS providers (
    id              TEXT PRIMARY KEY,
    display_name    TEXT NOT NULL,
    api_key_enc     BLOB,
    last_synced_at  DATETIME,
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS projects (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    color           TEXT NOT NULL DEFAULT '#6366f1',
    auto_tag_rules  TEXT DEFAULT '[]',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS usage_events (
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

CREATE INDEX IF NOT EXISTS idx_usage_events_timestamp ON usage_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_usage_events_provider ON usage_events(provider_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_model ON usage_events(model);
CREATE INDEX IF NOT EXISTS idx_usage_events_project ON usage_events(project_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_dedup ON usage_events(dedup_hash);

CREATE TABLE IF NOT EXISTS budgets (
    id                  TEXT PRIMARY KEY,
    scope               TEXT NOT NULL UNIQUE,
    monthly_limit_usd   REAL NOT NULL,
    alert_thresholds    TEXT NOT NULL DEFAULT '[50, 80, 100]',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pricing (
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

CREATE INDEX IF NOT EXISTS idx_pricing_provider_model ON pricing(provider_id, model);

CREATE TABLE IF NOT EXISTS settings (
    key     TEXT PRIMARY KEY,
    value   TEXT NOT NULL
);

-- Insert default providers
INSERT OR IGNORE INTO providers (id, display_name, enabled) VALUES
    ('anthropic', 'Anthropic', 1),
    ('openai', 'OpenAI', 1);

-- Insert default settings
INSERT OR IGNORE INTO settings (key, value) VALUES
    ('poll_interval_hours', '6'),
    ('log_watch_enabled', 'false'),
    ('log_watch_path', ''),
    ('theme', 'system'),
    ('data_retention_months', '0');
