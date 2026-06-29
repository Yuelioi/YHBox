import { describe, expect, it } from 'vitest'
import { platformBadgesForTargets } from './platformTargets'

describe('platformBadgesForTargets', () => {
  it('returns no badges when a node has no target semantics', () => {
    expect(platformBadgesForTargets(undefined)).toEqual([])
    expect(platformBadgesForTargets([])).toEqual([])
  })

  it('labels Windows-only nodes', () => {
    expect(platformBadgesForTargets(['win32-window'])).toEqual([
      { key: 'windows', labelKey: 'nodeExplorer.platform_windows_only', class: 'border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-300' },
    ])
  })

  it('labels Android-only nodes', () => {
    expect(platformBadgesForTargets(['android-adb'])).toEqual([
      { key: 'android', labelKey: 'nodeExplorer.platform_android_only', class: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300' },
    ])
  })

  it('labels cross-target nodes without "only"', () => {
    expect(platformBadgesForTargets(['win32-window', 'android-adb'])).toEqual([
      { key: 'windows', labelKey: 'nodeExplorer.platform_windows', class: 'border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-300' },
      { key: 'android', labelKey: 'nodeExplorer.platform_android', class: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300' },
    ])
  })
})
