import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsGeneral.vue'), 'utf8')

describe('SettingsGeneral editor display', () => {
  it('groups appearance, behavior, capture, and diagnostics with shared settings primitives', () => {
    expect(source).toContain('useSidebarPrefs')
    expect(source).toContain('<SettingsSection')
    expect(source).toContain('<SettingsRow')
    expect(source).toContain('<SettingsRestartBadge')
    expect(source).toContain(':model-value="sidebarPrefs.experienceMode"')
    expect(source).toContain('@update:model-value="onEditorDetailChange"')
    expect(source).toContain("patchLogger('enabled', value)")
  })
})
