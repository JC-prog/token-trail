<script>
  import { dashboardSummary, spendOverTime, usageByProvider, usageByModel, tokenRatio, loading, error, refreshDashboard } from '../lib/stores/dashboard'
  import { lastSyncTimes } from '../lib/stores/providers'
  import { syncAll } from '../lib/wailsbridge'
  import { formatCost, formatTokens, formatPercent } from '../lib/utils/format'
  import { onMount } from 'svelte'

  let syncing = false

  onMount(() => {
    refreshDashboard()
  })

  async function handleSync() {
    syncing = true
    try {
      await syncAll()
      await refreshDashboard()
    } finally {
      syncing = false
    }
  }
</script>

<div class="dashboard">
  <div class="header">
    <h1>Dashboard</h1>
    <button disabled={syncing || $loading} on:click={handleSync}>
      {syncing ? '⏳ Syncing...' : '🔄 Sync Now'}
    </button>
  </div>

  {#if $error}
    <div class="error">{$error}</div>
  {/if}

  {#if $loading}
    <div style="text-align: center; padding: 2rem;">
      <div class="loading"></div>
      <p>Loading dashboard...</p>
    </div>
  {:else if $dashboardSummary}
    <div class="grid grid-2" style="margin-bottom: 2rem;">
      <div class="card">
        <h2>Total Spend (This Month)</h2>
        <p>{formatCost($dashboardSummary.total_spend)}</p>
      </div>
      <div class="card">
        <h2>Total Tokens</h2>
        <p>{formatTokens($dashboardSummary.total_tokens)}</p>
      </div>
      <div class="card">
        <h2>Daily Average</h2>
        <p>{formatCost($dashboardSummary.avg_daily_spend)}</p>
      </div>
      <div class="card">
        <h2>Projected Month-End</h2>
        <p>{formatCost($dashboardSummary.projected_spend)}</p>
      </div>
    </div>

    <div class="grid grid-2" style="margin-bottom: 2rem;">
      <div class="chart-container">
        <h2>Top Models</h2>
        {#if $usageByModel && $usageByModel.length > 0}
          <table>
            <thead>
              <tr>
                <th>Model</th>
                <th>Cost</th>
              </tr>
            </thead>
            <tbody>
              {#each $usageByModel.slice(0, 5) as model}
                <tr>
                  <td>{model.model}</td>
                  <td>{formatCost(model.total_cost)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p>No usage data yet</p>
        {/if}
      </div>

      <div class="chart-container">
        <h2>By Provider</h2>
        {#if $usageByProvider && $usageByProvider.length > 0}
          <table>
            <thead>
              <tr>
                <th>Provider</th>
                <th>Cost</th>
                <th>%</th>
              </tr>
            </thead>
            <tbody>
              {#each $usageByProvider as provider}
                <tr>
                  <td>{provider.display_name}</td>
                  <td>{formatCost(provider.total_cost)}</td>
                  <td>{formatPercent(provider.percentage)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p>No provider data yet</p>
        {/if}
      </div>
    </div>

    {#if $tokenRatio}
      <div class="chart-container">
        <h2>Input vs Output Tokens</h2>
        <table>
          <tr>
            <th>Type</th>
            <th>Count</th>
            <th>Percentage</th>
          </tr>
          <tr>
            <td>Input</td>
            <td>{formatTokens($tokenRatio.input_tokens)}</td>
            <td>{formatPercent(100 / (1 + $tokenRatio.ratio), 1)}</td>
          </tr>
          <tr>
            <td>Output</td>
            <td>{formatTokens($tokenRatio.output_tokens)}</td>
            <td>{formatPercent((100 * $tokenRatio.ratio) / (1 + $tokenRatio.ratio), 1)}</td>
          </tr>
        </table>
      </div>
    {/if}
  {/if}
</div>

<style>
  .dashboard {
    max-width: 1200px;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  .header h1 {
    margin: 0;
    font-size: 2rem;
  }

  .loading {
    display: inline-block;
    width: 2rem;
    height: 2rem;
    border: 3px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.875rem;
  }

  th,
  td {
    padding: 0.75rem;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }

  th {
    background-color: var(--bg-primary);
    font-weight: 600;
    color: var(--text-secondary);
  }

  tr:hover {
    background-color: var(--bg-secondary);
  }
</style>
