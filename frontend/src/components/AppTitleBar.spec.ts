import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/components/AppTitleBar.vue'), 'utf8')

describe('AppTitleBar accessibility', () => {
  it('labels primary navigation and current destinations', () => {
    expect(source).toContain(':aria-label="t(\'sidebar.primary_navigation\')"')
    expect(source).toContain(':aria-current="item.active ? \'page\' : undefined"')
    expect(source).toContain('buildAppNavigation')
  })

  it('uses the Yotta product mark instead of a generic app icon', () => {
    expect(source).toContain('YottaMark')
    expect(source).not.toContain('i-tabler-device-gamepad-2')
  })

  it('names utility destinations and native window controls', () => {
    expect(source).toContain(':aria-label="t(\'sidebar.settings\')"')
    expect(source).toContain(':aria-label="t(\'sidebar.about\')"')
    expect(source).toContain(':aria-label="t(\'sidebar.open_launcher\')"')
    expect(source).toContain('data-testid="open-launcher"')
    expect(source).toContain('backend.tools.openLauncher()')
    expect(source).toContain(':aria-label="t(\'editor.window.minimize\')"')
    expect(source).toContain(':aria-label="t(\'editor.window.close\')"')
  })

  it('routes close requests through the workflow leave guard', () => {
    expect(source).toContain('requestMainWindowClose')
    expect(source).not.toContain('closeImmediate: onClose')
  })
})
