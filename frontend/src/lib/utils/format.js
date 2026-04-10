// Format currency to USD
export function formatCost(value) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

// Format token count
export function formatTokens(value) {
  if (value >= 1_000_000) {
    return (value / 1_000_000).toFixed(1) + 'M'
  }
  if (value >= 1_000) {
    return (value / 1_000).toFixed(1) + 'K'
  }
  return value.toString()
}

// Format date
export function formatDate(date) {
  if (typeof date === 'string') {
    date = new Date(date)
  }
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

// Format datetime
export function formatDateTime(date) {
  if (typeof date === 'string') {
    date = new Date(date)
  }
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

// Format time ago
export function formatTimeAgo(date) {
  if (typeof date === 'string') {
    date = new Date(date)
  }

  const seconds = Math.floor((new Date() - date) / 1000)

  let interval = seconds / 31536000
  if (interval > 1) return Math.floor(interval) + 'y'

  interval = seconds / 2592000
  if (interval > 1) return Math.floor(interval) + 'mo'

  interval = seconds / 86400
  if (interval > 1) return Math.floor(interval) + 'd'

  interval = seconds / 3600
  if (interval > 1) return Math.floor(interval) + 'h'

  interval = seconds / 60
  if (interval > 1) return Math.floor(interval) + 'm'

  return Math.floor(seconds) + 's'
}

// Format percentage
export function formatPercent(value, decimals = 1) {
  return value.toFixed(decimals) + '%'
}
