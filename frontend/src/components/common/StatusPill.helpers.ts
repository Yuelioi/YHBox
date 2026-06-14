// StatusPill 纯映射: status → 字面 class (动态 text-${x} 会被 Tailwind purge, 故用字面表)。
export type PillStatus = 'online' | 'ready' | 'paused' | 'failed'

const MAP: Record<PillStatus, string> = {
  online: 'bg-primary/15 text-primary',
  ready: 'bg-elevated text-muted',
  paused: 'bg-warning/15 text-warning',
  failed: 'bg-error/15 text-error',
}

export function statusPillClass(status: PillStatus): string {
  return MAP[status]
}
