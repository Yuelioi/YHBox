export const SETTINGS_THEME_KEYS = [
  'general',
  'hotkeys',
  'input',
  'launcher',
  'ai',
  'network',
  'applications',
  'automation',
] as const

export type SettingsThemeKey = (typeof SETTINGS_THEME_KEYS)[number]

export interface SettingsThemeDefinition {
  key: SettingsThemeKey
  labelKey: string
  descriptionKey: string
  icon: string
}

export const SETTINGS_THEMES: readonly SettingsThemeDefinition[] = [
  {
    key: 'general',
    labelKey: 'settingsTab.general',
    descriptionKey: 'settingsCenter.theme.general',
    icon: 'i-tabler-adjustments-horizontal',
  },
  {
    key: 'hotkeys',
    labelKey: 'settingsTab.hotkeys',
    descriptionKey: 'settingsCenter.theme.hotkeys',
    icon: 'i-tabler-keyboard',
  },
  {
    key: 'input',
    labelKey: 'settingsTab.input_calibration',
    descriptionKey: 'settingsCenter.theme.input',
    icon: 'i-tabler-mouse-2',
  },
  {
    key: 'launcher',
    labelKey: 'settingsTab.launcher',
    descriptionKey: 'settingsCenter.theme.launcher',
    icon: 'i-tabler-layout-grid-add',
  },
  {
    key: 'ai',
    labelKey: 'settingsTab.ai',
    descriptionKey: 'settingsCenter.theme.ai',
    icon: 'i-tabler-sparkles',
  },
  {
    key: 'network',
    labelKey: 'settingsTab.network',
    descriptionKey: 'settingsCenter.theme.network',
    icon: 'i-tabler-shield-lock',
  },
  {
    key: 'applications',
    labelKey: 'settingsTab.applications',
    descriptionKey: 'settingsCenter.theme.applications',
    icon: 'i-tabler-apps',
  },
  {
    key: 'automation',
    labelKey: 'settingsTab.automation',
    descriptionKey: 'settingsCenter.theme.automation',
    icon: 'i-tabler-pointer-cog',
  },
]

export function isSettingsThemeKey(value: unknown): value is SettingsThemeKey {
  return typeof value === 'string' && SETTINGS_THEME_KEYS.includes(value as SettingsThemeKey)
}
