<script>
  import Router, { link } from 'svelte-spa-router'
  import { onMount } from 'svelte'
  import Dashboard from './pages/Dashboard.svelte'
  import History from './pages/History.svelte'
  import Projects from './pages/Projects.svelte'
  import Settings from './pages/Settings.svelte'
  import { loadProviders } from './lib/stores/providers'
  import { loadSettings } from './lib/stores/settings'
  import { refreshDashboard } from './lib/stores/dashboard'
  import { onSyncCompleted, onBudgetAlert } from './lib/wailsbridge'

  let currentPage = '/'
  let theme = 'system'

  const routes = {
    '/': Dashboard,
    '/history': History,
    '/projects': Projects,
    '/settings': Settings,
  }

  onMount(async () => {
    // Load initial data
    await loadProviders()
    await loadSettings()
    await refreshDashboard()

    // Listen for sync events
    onSyncCompleted(() => {
      refreshDashboard()
    })

    // Listen for budget alerts
    onBudgetAlert((data) => {
      console.log('Budget alert:', data)
      // Could show a toast notification here
    })

    // Apply theme
    const savedTheme = localStorage.getItem('theme') || 'system'
    theme = savedTheme
    applyTheme(theme)
  })

  function applyTheme(t) {
    theme = t

    if (t === 'system') {
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light')
    } else {
      document.documentElement.setAttribute('data-theme', t)
    }

    localStorage.setItem('theme', t)
  }

  function handleRouteChange(event) {
    currentPage = event.detail.location
  }
</script>

<div class="container">
  <div class="sidebar">
    <h1>TokenTrail</h1>

    <nav>
      <a
        use:link
        href="/"
        class:active={currentPage === '/'}
        on:click={() => (currentPage = '/')}
      >
        📊 Dashboard
      </a>
      <a
        use:link
        href="/history"
        class:active={currentPage === '/history'}
        on:click={() => (currentPage = '/history')}
      >
        📋 History
      </a>
      <a
        use:link
        href="/projects"
        class:active={currentPage === '/projects'}
        on:click={() => (currentPage = '/projects')}
      >
        📁 Projects
      </a>
      <a
        use:link
        href="/settings"
        class:active={currentPage === '/settings'}
        on:click={() => (currentPage = '/settings')}
      >
        ⚙️ Settings
      </a>
    </nav>

    <div style="margin-top: auto; padding-top: 2rem; border-top: 1px solid rgba(255,255,255,0.1);">
      <small style="color: var(--text-secondary);">TokenTrail v0.1.0</small>
    </div>
  </div>

  <div class="content">
    <Router {routes} on:routeEvent={handleRouteChange} />
  </div>
</div>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
  }

  .container {
    display: flex;
    height: 100vh;
  }

  .sidebar {
    width: 240px;
    background-color: var(--bg-sidebar);
    color: var(--text-sidebar);
    padding: 1.5rem;
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }

  .sidebar h1 {
    font-size: 1.5rem;
    font-weight: 700;
    margin: 0 0 2rem 0;
  }

  nav {
    flex: 1;
  }

  a {
    display: block;
    padding: 0.75rem 1rem;
    margin-bottom: 0.5rem;
    border-radius: 0.375rem;
    color: var(--text-sidebar);
    text-decoration: none;
    cursor: pointer;
    transition: background-color 0.2s;
  }

  a:hover,
  a.active {
    background-color: rgba(99, 102, 241, 0.2);
  }

  .content {
    flex: 1;
    overflow-y: auto;
    padding: 2rem;
    background-color: var(--bg-primary);
  }
</style>
