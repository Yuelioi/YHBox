import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (path: string) => readFileSync(join(process.cwd(), path), 'utf8')

describe('settings visual system', () => {
  it('keeps section and field descriptions subordinate to their titles', () => {
    const styles = readSource('src/style.css')
    const section = readSource('src/components/settings/SettingsSection.vue')
    const row = readSource('src/components/settings/SettingsRow.vue')

    expect(section).toContain('settings-section__title')
    expect(section).toContain('settings-section__description')
    expect(row).toContain('settings-row__label')
    expect(row).toContain('settings-row__hint')
    expect(styles).toContain("> [data-slot='labelWrapper']")
    expect(styles).toContain("> [data-slot='hint']")
    expect(styles).toMatch(/\.settings-row__label\s*\{[\s\S]*?font-size:\s*13px/)
    expect(styles).toMatch(/\.settings-row__hint\s*\{[\s\S]*?font-size:\s*11px/)
  })

  it('uses one card surface with flat content across settings tabs', () => {
    const styles = readSource('src/style.css')
    const view = readSource('src/views/SettingsView.vue')
    const applications = readSource('src/views/SettingsApplications.vue')
    const network = readSource('src/views/SettingsNetwork.vue')
    const automation = readSource('src/views/SettingsAutomation.vue')

    expect(view).toContain(':data-settings-theme="activeKey"')
    expect(styles).toContain('--ui-surface:')
    expect(styles).toContain('--settings-section-bg:')
    expect(styles).not.toContain('--settings-inset-bg:')
    expect(styles).not.toContain('--settings-item-bg:')
    expect(styles).toContain('.settings-section {')
    expect(styles).toContain('.settings-collection {')
    expect(styles).toContain('.settings-inset {')
    expect(styles).toMatch(/\.settings-collection\s*\{[\s\S]*?background:\s*transparent/)
    expect(styles).toMatch(/\.launcher-health-card\s*\{[\s\S]*?background:\s*transparent/)
    expect(styles).toMatch(/\.launcher-block\s*\{[\s\S]*?background:\s*transparent/)
    expect(applications).toContain('class="settings-collection"')
    expect(network).toContain('class="settings-collection"')
    expect(automation).toContain('class="settings-collection"')
  })

  it('keeps double-click editing as an accelerator with a visible edit command', () => {
    const applications = readSource('src/views/SettingsApplications.vue')
    const automation = readSource('src/views/SettingsAutomation.vue')

    for (const source of [applications, automation]) {
      expect(source).toContain('class="settings-entity-summary"')
      expect(source).toContain('@dblclick.prevent=')
      expect(source).toContain("'common.close' : 'common.edit'")
      expect(source).toContain(':aria-expanded=')
      expect(source).toContain(':aria-controls=')
    }
  })

  it('does not expose removed implementation disclaimers in application settings', () => {
    const zh = readSource('src/i18n/zh.ts')
    const en = readSource('src/i18n/en.ts')

    expect(zh).not.toContain('不做哈希或身份校验')
    expect(en).not.toContain('No hash or identity check is performed')
  })
})
