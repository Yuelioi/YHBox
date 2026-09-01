export const SETTINGS_THEME_KEYS = [
  'general',
  'hotkeys',
  'input',
  'launcher',
  'ai',
  'mcp',
  'network',
  'applications',
  'automation',
] as const

export type SettingsThemeKey = (typeof SETTINGS_THEME_KEYS)[number]
export type SettingsThemeGroupKey = 'common' | 'connections' | 'automation' | 'advanced'

export interface SettingsThemeDefinition {
  key: SettingsThemeKey
  group: SettingsThemeGroupKey
  labelKey: string
  descriptionKey: string
  icon: string
}

export const SETTINGS_THEMES: readonly SettingsThemeDefinition[] = [
  {
    key: 'general',
    group: 'common',
    labelKey: 'settingsTab.general',
    descriptionKey: 'settingsCenter.theme.general',
    icon: 'i-tabler-adjustments-horizontal',
  },
  {
    key: 'hotkeys',
    group: 'common',
    labelKey: 'settingsTab.hotkeys',
    descriptionKey: 'settingsCenter.theme.hotkeys',
    icon: 'i-tabler-keyboard',
  },
  {
    key: 'input',
    group: 'automation',
    labelKey: 'settingsTab.input_calibration',
    descriptionKey: 'settingsCenter.theme.input',
    icon: 'i-tabler-mouse-2',
  },
  {
    key: 'launcher',
    group: 'common',
    labelKey: 'settingsTab.launcher',
    descriptionKey: 'settingsCenter.theme.launcher',
    icon: 'i-tabler-layout-grid-add',
  },
  {
    key: 'ai',
    group: 'connections',
    labelKey: 'settingsTab.ai',
    descriptionKey: 'settingsCenter.theme.ai',
    icon: 'i-tabler-sparkles',
  },
  {
    key: 'mcp',
    group: 'connections',
    labelKey: 'settingsTab.mcp',
    descriptionKey: 'settingsCenter.theme.mcp',
    icon: 'i-tabler-plug-connected',
  },
  {
    key: 'network',
    group: 'advanced',
    labelKey: 'settingsTab.network',
    descriptionKey: 'settingsCenter.theme.network',
    icon: 'i-tabler-world-www',
  },
  {
    key: 'applications',
    group: 'connections',
    labelKey: 'settingsTab.applications',
    descriptionKey: 'settingsCenter.theme.applications',
    icon: 'i-tabler-apps',
  },
  {
    key: 'automation',
    group: 'automation',
    labelKey: 'settingsTab.automation',
    descriptionKey: 'settingsCenter.theme.automation',
    icon: 'i-tabler-pointer-cog',
  },
]

export const SETTINGS_THEME_GROUPS: ReadonlyArray<{
  key: SettingsThemeGroupKey
  labelKey: string
}> = [
  { key: 'common', labelKey: 'settingsCenter.group.common' },
  { key: 'connections', labelKey: 'settingsCenter.group.connections' },
  { key: 'automation', labelKey: 'settingsCenter.group.automation' },
  { key: 'advanced', labelKey: 'settingsCenter.group.advanced' },
]

export function groupSettingsThemes(themes: readonly SettingsThemeDefinition[]): Array<{
  key: SettingsThemeGroupKey
  labelKey: string
  themes: SettingsThemeDefinition[]
}> {
  return SETTINGS_THEME_GROUPS.flatMap((group) => {
    const grouped = themes.filter((theme) => theme.group === group.key)
    return grouped.length ? [{ ...group, themes: grouped }] : []
  })
}

export function isSettingsThemeKey(value: unknown): value is SettingsThemeKey {
  return typeof value === 'string' && SETTINGS_THEME_KEYS.includes(value as SettingsThemeKey)
}
