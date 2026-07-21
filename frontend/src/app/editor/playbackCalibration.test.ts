import { describe, expect, it } from 'vitest'
import type { InstalledAutomationTargetProfile } from '@/lib/backend'
import { resolvePlaybackCalibration } from './playbackCalibration'

function desktopTarget(mouseCounts360: number): InstalledAutomationTargetProfile {
  return {
    slot: 'game',
    label: 'Game',
    targetKind: 'desktop-window',
    adapterKind: 'win32',
    profileVersion: '1',
    profile: {
      applicationSlot: 'game',
      windowTitle: '',
      windowTitleMatch: 'exact',
      windowSelection: 'unique',
      windowClass: '',
      inputBackend: 'sendinput',
      captureBackend: 'gdi',
      mouseCounts360,
      resolveTimeoutMilliseconds: 3000,
    },
  }
}

describe('playback calibration projection', () => {
  it('uses a target 360 degree calibration before the active profile', () => {
    expect(resolvePlaybackCalibration(400, desktopTarget(800), 1200)).toEqual({
      sourceCounts: 400,
      targetCounts: 800,
      targetSource: 'custom',
    })
  })

  it('follows the active 360 degree calibration when the target has no override', () => {
    expect(resolvePlaybackCalibration(4132, desktopTarget(0), 4132)).toEqual({
      sourceCounts: 4132,
      targetCounts: 4132,
      targetSource: 'active',
    })
  })
})
