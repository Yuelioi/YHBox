const invalidTime = '—'

export function formatTimelineClock(value: string): string {
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return invalidTime
  const date = new Date(timestamp)
  return (
    [pad(date.getHours(), 2), pad(date.getMinutes(), 2), pad(date.getSeconds(), 2)].join(':') +
    `.${pad(date.getMilliseconds(), 3)}`
  )
}

export function formatTimelineDateTime(value: string): string {
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return invalidTime
  const date = new Date(timestamp)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1, 2)}-${pad(date.getDate(), 2)} ${formatTimelineClock(value)}`
}

export function formatTimelineOffset(value: string, origin: string): string {
  const timestamp = Date.parse(value)
  const originTimestamp = Date.parse(origin)
  if (!Number.isFinite(timestamp) || !Number.isFinite(originTimestamp)) return invalidTime

  const milliseconds = Math.max(0, timestamp - originTimestamp)
  if (milliseconds < 1000) return `+${milliseconds}ms`
  if (milliseconds < 60_000) return `+${(milliseconds / 1000).toFixed(3)}s`

  const minutes = Math.floor(milliseconds / 60_000)
  const seconds = ((milliseconds % 60_000) / 1000).toFixed(3)
  return `+${minutes}m ${seconds}s`
}

function pad(value: number, length: number): string {
  return String(value).padStart(length, '0')
}
