import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/components/AppTitleBar.vue'), 'utf8')

describe('AppTitleBar accessibility', () => {
  it('labels primary navigation and current destinations', () => {
    expect(source).toContain(':aria-label="t(\'sidebar.primary_navigation\')"')
    expect(source).toContain(':aria-current="item.active ? \'page\' : undefined"')
  })

  it('names utility destinations and native window controls', () => {
    expect(source).toContain(':aria-label="t(\'sidebar.settings\')"')
    expect(source).toContain(':aria-label="t(\'sidebar.about\')"')
    expect(source).toContain(':aria-label="t(\'editor.window.minimize\')"')
    expect(source).toContain(':aria-label="t(\'editor.window.close\')"')
  })
})
