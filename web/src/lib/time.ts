const MINUTE = 60
const HOUR = 3600
const DAY = 86400
const WEEK = 604800
const MONTH = 2592000

const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto', style: 'long' })

export function relativeTime(date: Date | string): string {
  const then = typeof date === 'string' ? new Date(date) : date
  const seconds = Math.floor((then.getTime() - Date.now()) / 1000)
  const abs = Math.abs(seconds)

  if (abs < MINUTE) return 'just now'

  const opts: [number, Intl.RelativeTimeFormatUnit][] = [
    [MINUTE, 'minute'],
    [HOUR, 'hour'],
    [DAY, 'day'],
    [WEEK, 'week'],
    [MONTH, 'month'],
  ]

  for (const [threshold, unit] of opts) {
    if (abs < threshold * 1.5 || unit === 'month') {
      return rtf.format(Math.round(seconds / threshold), unit)
    }
  }

  return rtf.format(Math.round(seconds / MONTH), 'month')
}

export function formatDate(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return d.toLocaleDateString('en', {
    month: 'short',
    day: 'numeric',
    year: d.getFullYear() !== new Date().getFullYear() ? 'numeric' : undefined,
    hour: 'numeric',
    minute: '2-digit',
  })
}