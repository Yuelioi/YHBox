import { describe, expect, it } from 'vitest'
import { groupSettingsThemes, SETTINGS_THEMES, SETTINGS_THEME_KEYS } from './registry'

describe('settings hierarchy', () => {
  it('places every stable settings route in exactly one user-facing group', () => {
    const groups = groupSettingsThemes(SETTINGS_THEMES)
    const keys = groups.flatMap((group) => group.themes.map((theme) => theme.key))

    expect(groups.map((group) => group.key)).toEqual([
      'common',
      'connections',
      'automation',
      'advanced',
    ])
    expect(keys).toHaveLength(new Set(keys).size)
    expect(new Set(keys)).toEqual(new Set(SETTINGS_THEME_KEYS))
  })

  it('drops empty groups after search without changing route identities', () => {
    const groups = groupSettingsThemes(
      SETTINGS_THEMES.filter((theme) => theme.key === 'applications'),
    )

    expect(groups).toHaveLength(1)
    expect(groups[0]?.key).toBe('connections')
    expect(groups[0]?.themes[0]?.key).toBe('applications')
  })
})
