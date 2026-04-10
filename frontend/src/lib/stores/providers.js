import { writable } from 'svelte/store'
import { listProviders as listProvidersAPI, getLastSyncTimes } from '../wailsbridge'

export const providers = writable([])
export const lastSyncTimes = writable({})
export const loading = writable(false)
export const error = writable(null)

export async function loadProviders() {
  loading.set(true)
  error.set(null)

  try {
    const [providerList, syncTimes] = await Promise.all([
      listProvidersAPI(),
      getLastSyncTimes(),
    ])

    providers.set(providerList || [])
    lastSyncTimes.set(syncTimes || {})
  } catch (err) {
    error.set(err.message)
    console.error('Failed to load providers:', err)
  } finally {
    loading.set(false)
  }
}

export async function refreshSyncTimes() {
  try {
    const syncTimes = await getLastSyncTimes()
    lastSyncTimes.set(syncTimes || {})
  } catch (err) {
    console.error('Failed to refresh sync times:', err)
  }
}
