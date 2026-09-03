import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = (name: string) => readFileSync(join(process.cwd(), `src/views/${name}.vue`), 'utf8')

describe('settings surface distillation', () => {
  it('keeps launcher cleanup beside command composition instead of a health section', () => {
    const launcher = source('SettingsLauncher')
    expect(launcher).toContain("t('settingsLauncher.layout_stats'")
    expect(launcher).toContain('@click="cleanupStale"')
    expect(launcher).toContain('cleanupStaleLauncherBlocks')
    expect(launcher).not.toContain("t('settingsLauncher.health_title')")
  })

  it('removes explanatory notice cards from AI and MCP settings', () => {
    expect(source('SettingsAI')).not.toContain('settingsAI.security')
    expect(source('SettingsMCP')).not.toContain('settingsMCP.security_')
  })
})
