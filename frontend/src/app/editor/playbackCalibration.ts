import type { InstalledAutomationTargetProfile } from '@/lib/backend'

export interface PlaybackCalibration {
  sourceCounts: number
  targetCounts: number
  targetSource: 'custom' | 'active' | 'missing' | 'unsupported'
}

export function resolvePlaybackCalibration(
  sourceCounts: number,
  target: InstalledAutomationTargetProfile | undefined,
  activeCounts: number,
): PlaybackCalibration {
  let targetCounts = 0
  let targetSource: PlaybackCalibration['targetSource'] = 'missing'
  if (target && target.targetKind !== 'desktop-window') {
    targetSource = 'unsupported'
  } else if (target) {
    const configured = Number((target.profile as Record<string, unknown>).mouseCounts360 ?? 0)
    if (Number.isFinite(configured) && configured > 0) {
      targetCounts = configured
      targetSource = 'custom'
    } else if (Number.isFinite(activeCounts) && activeCounts > 0) {
      targetCounts = activeCounts
      targetSource = 'active'
    }
  }
  const validSource = Number.isFinite(sourceCounts) && sourceCounts > 0
  return {
    sourceCounts: validSource ? sourceCounts : 0,
    targetCounts,
    targetSource,
  }
}
