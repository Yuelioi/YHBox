import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsGeneral.vue'), 'utf8')

describe('SettingsGeneral editor display', () => {
  it('owns the persisted node detail preference', () => {
    expect(source).toContain('useSidebarPrefs')
    expect(source).toContain("t('settings.editor_display.section_title')")
    expect(source).toContain(':model-value="sidebarPrefs.experienceMode"')
    expect(source).toContain('@update:model-value="onEditorDetailChange"')
  })
})
