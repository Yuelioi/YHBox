import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsView.vue'), 'utf8')

describe('SettingsView navigation', () => {
  it('exposes a labelled vertical tab interface', () => {
    expect(source).toContain('role="tablist"')
    expect(source).toContain('aria-orientation="vertical"')
    expect(source).toContain('role="tab"')
    expect(source).toContain(':aria-selected="activeTab === tab.key"')
    expect(source).toContain('role="tabpanel"')
    expect(source).toContain('@keydown="onTabKeydown"')
  })

  it('keeps the settings content independently scrollable', () => {
    expect(source).toContain('class="min-w-0 flex-1 overflow-auto"')
  })
})
