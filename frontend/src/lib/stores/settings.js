import { writable } from 'svelte/store'
import { getSettings, updateSettings as updateSettingsAPI } from '../wailsbridge'

export const settings = writable(null)
export const loading = writable(false)
export const error = writable(null)

export async function loadSettings() {
  loading.set(true)
  error.set(null)

  try {
    const appSettings = await getSettings()
    settings.set(appSettings)
  } catch (err) {
    error.set(err.message)
    console.error('Failed to load settings:', err)
  } finally {
    loading.set(false)
  }
}

export async function updateSettings(newSettings) {
  loading.set(true)
  error.set(null)

  try {
    await updateSettingsAPI(newSettings)
    settings.set(newSettings)
  } catch (err) {
    error.set(err.message)
    console.error('Failed to update settings:', err)
    throw err
  } finally {
    loading.set(false)
  }
}
