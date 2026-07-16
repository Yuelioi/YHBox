import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsAutomation.vue'), 'utf8')

describe('SettingsAutomation', () => {
  it('binds targets through installed application slots and explicit consent', () => {
    expect(source).toContain('applicationSlot')
    expect(source).toContain('backend.automation.grantWorkflowConsent')
    expect(source).toContain('backend.automation.revokeWorkflowConsent')
    expect(source).toContain('<SettingsRestartBadge')
    expect(source).not.toMatch(/\b(?:HWND|processId|executable)\b/)
  })

  it('keeps expandable targets and form controls keyboard accessible', () => {
    expect(source).toContain(':aria-expanded="expandedSlot === target.slot"')
    expect(source).toContain(':aria-controls="`automation-target-${target.slot}`"')
    expect(source).toContain('<UFormField')
    expect(source).toContain(':loading="busy[target.slot]"')
  })
})
