<script>
  import { getUsageEvents } from '../lib/wailsbridge'
  import { formatCost, formatTokens, formatDateTime } from '../lib/utils/format'
  import { onMount } from 'svelte'

  let events = []
  let total = 0
  let limit = 50
  let offset = 0
  let loading = false
  let error = null
  let filter = {
    provider: null,
    model: null,
    project: null,
    from_date: null,
    to_date: null,
    limit: 50,
    offset: 0,
  }

  onMount(() => {
    loadEvents()
  })

  async function loadEvents() {
    loading = true
    error = null

    try {
      const result = await getUsageEvents(filter)
      events = result.events || []
      total = result.total || 0
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  function nextPage() {
    offset += limit
    filter.offset = offset
    loadEvents()
  }

  function prevPage() {
    offset = Math.max(0, offset - limit)
    filter.offset = offset
    loadEvents()
  }
</script>

<div class="history">
  <div class="header">
    <h1>Usage History</h1>
  </div>

  {#if error}
    <div class="error">{error}</div>
  {/if}

  {#if loading}
    <div style="text-align: center; padding: 2rem;">
      <div class="loading"></div>
      <p>Loading events...</p>
    </div>
  {:else if events.length > 0}
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Provider</th>
            <th>Model</th>
            <th>Input</th>
            <th>Output</th>
            <th>Total</th>
            <th>Cost</th>
            <th>Source</th>
          </tr>
        </thead>
        <tbody>
          {#each events as event}
            <tr>
              <td>{formatDateTime(event.timestamp)}</td>
              <td>{event.provider_id}</td>
              <td>{event.model}</td>
              <td>{formatTokens(event.input_tokens)}</td>
              <td>{formatTokens(event.output_tokens)}</td>
              <td>{formatTokens(event.input_tokens + event.output_tokens)}</td>
              <td>{formatCost(event.cost_usd)}</td>
              <td><span class="badge">{event.source}</span></td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <div class="pagination">
      <button disabled={offset === 0} on:click={prevPage}>← Previous</button>
      <span>Showing {offset + 1}-{Math.min(offset + limit, total)} of {total}</span>
      <button disabled={offset + limit >= total} on:click={nextPage}>Next →</button>
    </div>
  {:else}
    <div style="text-align: center; padding: 2rem;">
      <p>No events found. Try adding an API key and syncing data.</p>
    </div>
  {/if}
</div>

<style>
  .history {
    max-width: 1200px;
  }

  .header {
    margin-bottom: 2rem;
  }

  .header h1 {
    margin: 0;
    font-size: 2rem;
  }

  .table-container {
    background-color: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    overflow-x: auto;
    margin-bottom: 2rem;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.875rem;
  }

  th,
  td {
    padding: 1rem;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }

  th {
    background-color: var(--bg-primary);
    font-weight: 600;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  tr:hover {
    background-color: var(--bg-primary);
  }

  .badge {
    display: inline-block;
    padding: 0.25rem 0.75rem;
    border-radius: 0.25rem;
    font-size: 0.75rem;
    font-weight: 600;
    background-color: var(--chart-1);
    color: white;
  }

  .pagination {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 1rem;
    margin-top: 2rem;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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
</style>
