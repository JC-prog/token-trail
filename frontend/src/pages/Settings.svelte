<script>
  import { listProviders, addProvider, testProviderConnection, updateSettings as updateSettingsAPI, getDataStats } from '../lib/wailsbridge'
  import { settings, loadSettings, updateSettings } from '../lib/stores/settings'
  import { onMount } from 'svelte'

  let providers = []
  let dataStats = null
  let loading = false
  let error = null
  let success = null
  let apiKeys = {}
  let testingConnection = {}
  let savingSettings = false

  onMount(async () => {
    await loadProviders()
    await loadSettings()
    await loadDataStats()
  })

  async function loadProviders() {
    loading = true
    try {
      providers = await listProviders()
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  async function loadDataStats() {
    try {
      dataStats = await getDataStats()
    } catch (err) {
      console.error('Failed to load data stats:', err)
    }
  }

  async function handleAddKey(providerID) {
    if (!apiKeys[providerID]) {
      error = 'API key is required'
      return
    }

    loading = true
    error = null
    success = null

    try {
      await addProvider(providerID, apiKeys[providerID])
      apiKeys[providerID] = ''
      success = `${providerID} API key saved`
      await loadProviders()
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  async function handleTestConnection(providerID) {
    testingConnection[providerID] = true
    error = null
    success = null

    try {
      const result = await testProviderConnection(providerID)
      if (result.connected) {
        success = `Connected to ${providerID}`
      } else {
        error = result.error || `Failed to connect to ${providerID}`
      }
    } catch (err) {
      error = err.message
    } finally {
      testingConnection[providerID] = false
    }
  }

  async function handleSaveSettings() {
    if (!$settings) return

    savingSettings = true
    error = null
    success = null

    try {
      await updateSettings($settings)
      success = 'Settings saved'
    } catch (err) {
      error = err.message
    } finally {
      savingSettings = false
    }
  }
</script>

<div class="settings">
  <h1>Settings</h1>

  {#if error}
    <div class="error">{error}</div>
  {/if}

  {#if success}
    <div class="success">{success}</div>
  {/if}

  <div class="sections">
    <!-- API Keys Section -->
    <div class="section">
      <h2>API Keys</h2>
      <p class="subtitle">Add your API keys to fetch usage data from providers</p>

      {#if loading}
        <p>Loading providers...</p>
      {:else}
        <div class="provider-list">
          {#each providers as provider}
            <div class="provider-card">
              <div>
                <h3>{provider.display_name}</h3>
                <p class="provider-id">{provider.id}</p>
                {#if provider.last_synced_at}
                  <p class="sync-time">Last synced: {new Date(provider.last_synced_at).toLocaleString()}</p>
                {:else}
                  <p class="sync-time">Never synced</p>
                {/if}
              </div>

              <div class="provider-actions">
                <input
                  type="password"
                  placeholder="Enter API key"
                  bind:value={apiKeys[provider.id]}
                  on:keydown={(e) => e.key === 'Enter' && handleAddKey(provider.id)}
                />

                <div style="display: flex; gap: 0.5rem;">
                  <button
                    disabled={loading || !apiKeys[provider.id]}
                    on:click={() => handleAddKey(provider.id)}
                  >
                    Save
                  </button>

                  <button
                    disabled={testingConnection[provider.id] || loading}
                    on:click={() => handleTestConnection(provider.id)}
                    style="background-color: var(--accent);"
                  >
                    {testingConnection[provider.id] ? 'Testing...' : 'Test'}
                  </button>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- General Settings Section -->
    {#if $settings}
      <div class="section">
        <h2>General Settings</h2>

        <div class="form-group">
          <label>Polling Interval (hours)</label>
          <input
            type="number"
            min="1"
            max="168"
            bind:value={$settings.poll_interval_hours}
          />
          <small>How often to check for new usage data</small>
        </div>

        <div class="form-group">
          <label>Theme</label>
          <select bind:value={$settings.theme}>
            <option value="system">System</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </div>

        <div class="form-group">
          <label>
            <input type="checkbox" bind:checked={$settings.log_watch_enabled} />
            Watch Claude Code Logs
          </label>
          <small>Monitor ~/.claude/ for usage events</small>
        </div>

        {#if $settings.log_watch_enabled}
          <div class="form-group">
            <label>Log Watch Path</label>
            <input
              type="text"
              bind:value={$settings.log_watch_path}
              placeholder="Leave blank for default (~/.claude/)"
            />
          </div>
        {/if}

        <button
          on:click={handleSaveSettings}
          disabled={savingSettings}
          style="margin-top: 1rem;"
        >
          {savingSettings ? 'Saving...' : 'Save Settings'}
        </button>
      </div>
    {/if}

    <!-- Data Statistics Section -->
    {#if dataStats}
      <div class="section">
        <h2>Data Statistics</h2>
        <div class="stats-grid">
          <div class="stat">
            <div class="stat-label">Total Events</div>
            <div class="stat-value">{dataStats.total_events}</div>
          </div>
          <div class="stat">
            <div class="stat-label">This Month</div>
            <div class="stat-value">{dataStats.events_this_month}</div>
          </div>
          <div class="stat">
            <div class="stat-label">Providers</div>
            <div class="stat-value">{dataStats.providers_count}</div>
          </div>
          <div class="stat">
            <div class="stat-label">Projects</div>
            <div class="stat-value">{dataStats.projects_count}</div>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .settings {
    max-width: 800px;
  }

  .settings h1 {
    margin-bottom: 2rem;
    font-size: 2rem;
  }

  .sections {
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }

  .section {
    background-color: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    padding: 1.5rem;
  }

  .section h2 {
    margin: 0 0 0.5rem 0;
    font-size: 1.25rem;
  }

  .subtitle {
    margin: 0 0 1.5rem 0;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .provider-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .provider-card {
    background-color: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 0.375rem;
    padding: 1rem;
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
  }

  .provider-card h3 {
    margin: 0;
    font-size: 1rem;
  }

  .provider-id {
    margin: 0.25rem 0 0 0;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .sync-time {
    margin: 0.5rem 0 0 0;
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .provider-actions {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    min-width: 250px;
  }

  .provider-actions input {
    padding: 0.5rem;
    border: 1px solid var(--border);
    border-radius: 0.375rem;
  }

  .provider-actions button {
    padding: 0.5rem 1rem;
    background-color: var(--accent);
    color: white;
    border: none;
    border-radius: 0.375rem;
    cursor: pointer;
  }

  .form-group {
    margin-bottom: 1.5rem;
  }

  .form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
    font-size: 0.875rem;
  }

  .form-group input[type='text'],
  .form-group input[type='number'],
  .form-group select {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid var(--border);
    border-radius: 0.375rem;
    background-color: var(--bg-secondary);
    color: var(--text-primary);
  }

  .form-group small {
    display: block;
    margin-top: 0.25rem;
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .form-group input[type='checkbox'] {
    margin-right: 0.5rem;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1rem;
    margin-top: 1rem;
  }

  .stat {
    background-color: var(--bg-secondary);
    padding: 1rem;
    border-radius: 0.375rem;
    text-align: center;
  }

  .stat-label {
    font-size: 0.875rem;
    color: var(--text-secondary);
    margin-bottom: 0.5rem;
  }

  .stat-value {
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--accent);
  }
</style>
