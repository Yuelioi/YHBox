import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsAutomation.vue'), 'utf8')

describe('SettingsAutomation', () => {
  it('binds adapter-specific exact targets behind slots and explicit consent', () => {
    expect(source).toContain('applicationSlot')
    expect(source).toContain('backend.automation.grantWorkflowConsent')
    expect(source).toContain('backend.automation.revokeWorkflowConsent')
    expect(source).toContain('captureBackend')
    expect(source).toContain('backend.automation.listTargetTypes()')
    expect(source).toContain("captureBackend: (type.captureBackends[0] ?? '')")
    expect(source).toContain('targetKind: type.targetKind')
    expect(source).toContain('<SettingsRestartBadge')
    expect(source).not.toMatch(/\b(?:HWND|processId)\b/)
    expect(source).toContain("applicationSlot: ''")
    expect(source).not.toContain('applications.value[0]')
    expect(source).toContain('matchingInstalledApplications')
    expect(source).toContain('startWin32WindowTargetCapture')
    expect(source).toContain('cancelWin32WindowTargetCapture')
    expect(source).toContain("useWailsEvent<unknown>('win32windowtarget:captured'")
    expect(source).toContain('target.windowTitle.trim()')
    expect(source).toContain('target.windowClass.trim()')
    expect(source).toContain('@click="duplicateTarget(target)"')
    expect(source).toContain("addTarget('android-device')")
    expect(source).toContain('backend.automation.listADBDevices()')
    expect(source).toContain('backend.automation.checkTargetHealth(target.slot)')
    expect(source).toContain('target.adbProduct = selected.product')
    expect(source).toContain('target.androidPackage?.trim()')
    expect(source).toContain("addTarget('browser-page')")
    expect(source).toContain('backend.automation.listBrowserTargets')
    expect(source).toContain('target.browserWebSocketUrl = selected.webSocketDebuggerUrl')
    expect(source).toContain('target.browserTargetId?.trim()')
    expect(source).toContain('target.browserEndpoint?.trim()')
  })

  it('keeps expandable targets and form controls keyboard accessible', () => {
    expect(source).toContain(':aria-expanded="expandedSlot === target.slot"')
    expect(source).toContain(':aria-controls="`automation-target-${target.slot}`"')
    expect(source).toContain('<UFormField')
    expect(source).toContain(':loading="busy[target.slot]"')
  })
})
