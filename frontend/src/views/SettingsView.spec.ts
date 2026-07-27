import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsView.vue'), 'utf8')

describe('SettingsView navigation', () => {
  it('exposes a searchable, deep-linkable tab interface', () => {
    expect(source).toContain('role="tablist"')
    expect(source).toContain('aria-orientation="vertical"')
    expect(source).toContain('role="tab"')
    expect(source).toContain(':aria-selected="activeKey === theme.key"')
    expect(source).toContain(':tabindex="activeKey === theme.key ? 0 : -1"')
    expect(source).toContain('role="tabpanel"')
    expect(source).toContain('route.query.section')
    expect(source).toContain('v-model="searchQuery"')
    expect(source).toContain('v-for="group in filteredThemeGroups"')
    expect(source).toContain('groupSettingsThemes(filteredThemes.value)')
  })

  it('uses the shared commercial settings shell and preserves theme drafts', () => {
    expect(source).toContain('class="settings-shell"')
    expect(source).toContain('<SettingsPageHeader')
    expect(source).toContain('<KeepAlive>')
  })
})
