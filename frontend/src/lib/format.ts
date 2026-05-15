// 时长 / 百分比 / 时间戳格式化 helpers。

export function fmtDurationHM(startedAt: string | Date): string {
  if (!startedAt) return '—'
  const start = typeof startedAt === 'string' ? new Date(startedAt) : startedAt
  if (isNaN(start.getTime())) return '—'
  const mins = Math.floor((Date.now() - start.getTime()) / 60000)
  return `${Math.floor(mins / 60)}h ${mins % 60}m`
}

export function fmtMillis(ms: number): string {
  if (ms < 0) ms = 0
  const totalSec = Math.floor(ms / 1000)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

export function fmtTs(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  return `${hh}:${mm}:${ss}`
}
