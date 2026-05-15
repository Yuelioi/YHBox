import { createI18n } from 'vue-i18n'
import zh from './zh'
import en from './en'

export type Locale = 'zh' | 'en'
export const LOCALES: Locale[] = ['zh', 'en']

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: 'zh' as Locale,
  fallbackLocale: 'zh' as Locale,
  // 静默 fallback：缺键时自动回 zh，控制台不刷屏
  fallbackWarn: false,
  missingWarn: false,
  messages: { zh, en },
})

// setLocale 切换 UI 文字（hot swap）。SettingsView 改 locale 后调一次。
export function setLocale(loc: Locale) {
  i18n.global.locale.value = loc
}
