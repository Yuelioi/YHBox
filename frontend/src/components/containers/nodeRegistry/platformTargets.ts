import type { SupportedTargetKind } from './index'

export interface PlatformBadge {
  key: 'windows' | 'android' | 'common'
  labelKey: string
  class: string
}

const WINDOWS_BADGE = 'border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-300'
const ANDROID_BADGE =
  'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
const COMMON_BADGE = 'border-zinc-500/40 bg-zinc-500/10 text-zinc-700 dark:text-zinc-300'

export interface PlatformBadgeOptions {
  isPureData?: boolean
}

export function platformBadgesForTargets(
  targets: SupportedTargetKind[] | undefined,
  options: PlatformBadgeOptions = {},
): PlatformBadge[] {
  if (!targets || targets.length === 0) {
    if (options.isPureData) {
      return [{ key: 'common', labelKey: 'nodeExplorer.platform_common', class: COMMON_BADGE }]
    }
    return []
  }
  const hasWindows = targets.includes('win32-window')
  const hasAndroid = targets.includes('android-adb')
  const single = targets.length === 1
  const badges: PlatformBadge[] = []
  if (hasWindows) {
    badges.push({
      key: 'windows',
      labelKey: single ? 'nodeExplorer.platform_windows_only' : 'nodeExplorer.platform_windows',
      class: WINDOWS_BADGE,
    })
  }
  if (hasAndroid) {
    badges.push({
      key: 'android',
      labelKey: single ? 'nodeExplorer.platform_android_only' : 'nodeExplorer.platform_android',
      class: ANDROID_BADGE,
    })
  }
  return badges
}
