import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/components/LogPanel.vue'), 'utf8')

describe('LogPanel shell', () => {
  it('uses a semantic disclosure without nesting the panel actions inside it', () => {
    expect(source).toContain(':aria-expanded="!collapsed"')
    expect(source).toContain('aria-controls="app-log-panel-body"')
    expect(source).toContain('id="app-log-panel-body"')
  })

  it('exposes filter state and accessible names for icon actions', () => {
    expect(source).toContain(':aria-pressed="filter === opt"')
    expect(source).toContain(':aria-label="t(\'log.settings\')"')
    expect(source).toContain(':aria-label="t(\'log.clear\')"')
  })

  it('adapts expanded height to the available viewport', () => {
    expect(source).toContain('clamp(180px, 28vh, 320px)')
  })

  it('offers source-level logging, live transport, and minimum-level controls', () => {
    expect(source).toContain("toggleField('enabled', !enabled)")
    expect(source).toContain("toggleField('liveView'")
    expect(source).toContain("toggleField('level'")
    expect(source).toContain("t('log.disabled')")
  })
})
