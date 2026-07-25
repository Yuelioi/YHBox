import { describe, expect, it } from 'vitest'
import type { InstalledAutomationTargetProfile } from '@/lib/backend'
import { compatibleAutomationTargets, profileSettingsIssue } from './workflowInstallationSettings'

const targets: InstalledAutomationTargetProfile[] = [
  {
    slot: 'desktop',
    label: 'Desktop',
    targetKind: 'desktop-window',
    adapterKind: 'win32',
    profileVersion: '1',
    profile: {},
  },
  {
    slot: 'android',
    label: 'Android',
    targetKind: 'android-device',
    adapterKind: 'android-adb',
    profileVersion: '1',
    profile: {},
  },
  {
    slot: 'future',
    label: 'Future desktop',
    targetKind: 'desktop-window',
    adapterKind: 'win32',
    profileVersion: '2',
    profile: {},
  },
]

describe('workflow installation settings', () => {
  it('offers only exact target kind, adapter, and profile version matches', () => {
    expect(
      compatibleAutomationTargets(targets, {
        targetKind: 'desktop-window',
        adapterKind: 'win32',
        profileVersion: '1',
      }).map((target) => target.slot),
    ).toEqual(['desktop'])
  })

  it('requires target profile settings to be a JSON object', () => {
    expect(profileSettingsIssue('{"timeout":1000}')).toBeNull()
    expect(profileSettingsIssue('[]')).toBe('object-required')
    expect(profileSettingsIssue('null')).toBe('object-required')
    expect(profileSettingsIssue('{')).toBe('invalid-json')
  })
})
