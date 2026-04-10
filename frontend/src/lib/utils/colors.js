// Chart color palette
export const colors = {
  primary: 'rgb(99, 102, 241)',
  secondary: 'rgb(6, 182, 212)',
  tertiary: 'rgb(245, 158, 11)',
  danger: 'rgb(239, 68, 68)',
  success: 'rgb(34, 197, 94)',
}

export const chartColors = [
  colors.primary,
  colors.secondary,
  colors.tertiary,
  colors.danger,
  colors.success,
  'rgb(168, 85, 247)',
  'rgb(236, 72, 153)',
  'rgb(20, 184, 166)',
]

export function getColor(index) {
  return chartColors[index % chartColors.length]
}
