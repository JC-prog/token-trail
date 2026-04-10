import { writable } from 'svelte/store'
import { getDashboardSummary, getSpendOverTime, getUsageByProvider, getUsageByModel, getTokenRatio } from '../wailsbridge'

export const dashboardSummary = writable(null)
export const spendOverTime = writable([])
export const usageByProvider = writable([])
export const usageByModel = writable([])
export const tokenRatio = writable(null)
export const loading = writable(false)
export const error = writable(null)

export async function refreshDashboard(period = 'month') {
  loading.set(true)
  error.set(null)

  try {
    const [summary, spend, provider, model, ratio] = await Promise.all([
      getDashboardSummary(period),
      getSpendOverTime(period),
      getUsageByProvider(period),
      getUsageByModel(period),
      getTokenRatio(period),
    ])

    dashboardSummary.set(summary)
    spendOverTime.set(spend || [])
    usageByProvider.set(provider || [])
    usageByModel.set(model || [])
    tokenRatio.set(ratio)
  } catch (err) {
    error.set(err.message)
    console.error('Failed to refresh dashboard:', err)
  } finally {
    loading.set(false)
  }
}
