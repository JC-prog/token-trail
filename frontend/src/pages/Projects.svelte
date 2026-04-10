<script>
  import { listProjects, createProject, deleteProject } from '../lib/wailsbridge'
  import { onMount } from 'svelte'

  let projects = []
  let loading = false
  let error = null
  let showForm = false
  let newProject = { name: '', color: '#6366f1' }

  onMount(() => {
    loadProjects()
  })

  async function loadProjects() {
    loading = true
    error = null

    try {
      projects = await listProjects()
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  async function handleCreate() {
    if (!newProject.name) {
      error = 'Project name is required'
      return
    }

    loading = true
    error = null

    try {
      await createProject(newProject.name, newProject.color)
      newProject = { name: '', color: '#6366f1' }
      showForm = false
      await loadProjects()
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  async function handleDelete(id) {
    if (!confirm('Are you sure you want to delete this project?')) {
      return
    }

    loading = true
    error = null

    try {
      await deleteProject(id)
      await loadProjects()
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }
</script>

<div class="projects">
  <div class="header">
    <h1>Projects</h1>
    <button on:click={() => (showForm = !showForm)}>
      {showForm ? '✕ Cancel' : '+ New Project'}
    </button>
  </div>

  {#if error}
    <div class="error">{error}</div>
  {/if}

  {#if showForm}
    <div class="card" style="margin-bottom: 2rem;">
      <h2>Create New Project</h2>
      <div style="display: grid; gap: 1rem;">
        <div>
          <label>Project Name</label>
          <input
            type="text"
            placeholder="e.g., Productivity Platform"
            bind:value={newProject.name}
          />
        </div>
        <div>
          <label>Color</label>
          <input type="color" bind:value={newProject.color} />
        </div>
        <button on:click={handleCreate} disabled={loading}>
          {loading ? 'Creating...' : 'Create Project'}
        </button>
      </div>
    </div>
  {/if}

  {#if loading}
    <div style="text-align: center; padding: 2rem;">
      <div class="loading"></div>
      <p>Loading projects...</p>
    </div>
  {:else if projects.length > 0}
    <div class="grid">
      {#each projects as project}
        <div class="card">
          <div style="display: flex; justify-content: space-between; align-items: start;">
            <div>
              <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 0.5rem;">
                <div
                  style="width: 1rem; height: 1rem; border-radius: 0.25rem; background-color: {project.color};"
                />
                <h3 style="margin: 0; font-size: 1.125rem;">{project.name}</h3>
              </div>
              <p style="margin: 0; font-size: 0.875rem; color: var(--text-secondary);">
                Created {new Date(project.created_at).toLocaleDateString()}
              </p>
            </div>
            <button
              style="padding: 0.5rem; background-color: var(--danger); color: white; border: none; border-radius: 0.25rem; cursor: pointer; font-size: 0.875rem;"
              on:click={() => handleDelete(project.id)}
            >
              Delete
            </button>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div style="text-align: center; padding: 2rem;">
      <p>No projects yet. Create one to start organizing your usage data.</p>
    </div>
  {/if}
</div>

<style>
  .projects {
    max-width: 800px;
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

  .grid {
    display: grid;
    gap: 1.5rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
    color: var(--text-secondary);
    font-size: 0.875rem;
  }

  input {
    width: 100%;
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
