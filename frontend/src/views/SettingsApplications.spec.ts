import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsApplications.vue'), 'utf8')

describe('SettingsApplications', () => {
  it('treats a cancelled file picker as a clean no-op', () => {
    expect(source).toContain('if (!path)')
    expect(source).toContain('pickerCancelled.value = true')
    expect(source).toContain('if (!inspection) return')
    expect(source).toContain('finally {')
    expect(source).toContain('picking.value = false')
    expect(source).toContain("'settingsApplications.profiles.cancelled'")
  })

  it('prevents overlapping picker requests and reports inspection failures inline', () => {
    expect(source).toContain('if (picking.value) return')
    expect(source).toContain('if (busy[profile.slot]) return')
    expect(source).toContain('pickerFailure.value =')
    expect(source).toContain("t('settingsApplications.picker.inspect_failed')")
    expect(source).not.toContain('toast.add')
  })
})
