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
    expect(source).toContain("profileFieldOptions(desktopTargetType.value, 'captureBackend')")
    expect(source).toContain("profileFieldOptions(desktopTargetType.value, 'inputBackend')")
    expect(source).toContain(
      "captureBackend: (profileFieldOptions(type, 'captureBackend')[0] ?? '')",
    )
    expect(source).toContain('targetKind: type.targetKind')
    expect(source).not.toContain('<SettingsRestartBadge')
    expect(source).not.toMatch(/\b(?:HWND|processId)\b/)
    expect(source).toContain("applicationSlot: ''")
    expect(source).not.toContain('applications.value[0]')
    expect(source).toContain('matchingInstalledApplications')
    expect(source).toContain('applications: { profiles: [...applications.value, application] }')
    expect(source).toContain('settingsAutomation.capture.install_confirm')
    expect(source).not.toContain(':disabled="applications.length === 0 || !desktopTargetType"')
    expect(source).toContain('startWin32WindowTargetCapture')
    expect(source).toContain('cancelWin32WindowTargetCapture')
    expect(source).toContain('backend.automation.grantAllWorkflowConsents()')
    expect(source).toContain('backend.automation.revokeAllWorkflowConsents()')
    expect(source).toContain("useWailsEvent<unknown>('win32windowtarget:captured'")
    expect(source).toContain('target.windowTitle.trim()')
    expect(source).toContain('target.windowClass.trim()')
    expect(source).toContain('@click="duplicateTarget(target)"')
    expect(source).toContain("addTarget('android-device')")
    expect(source).toContain('backend.automation.listADBDevices()')
    expect(source).toContain('backend.automation.listAndroidApps(serial)')
    expect(source).toContain(':virtualize="androidAppItems(target.adbSerial).length > 40"')
    expect(source).toContain('backend.automation.checkTargetHealth(target.slot)')
    expect(source).toContain('target.adbProduct = selected.product')
    expect(source).toContain('target.androidPackage?.trim()')
    expect(source).toContain("addTarget('browser-page')")
    expect(source).toContain('backend.automation.listBrowserTargets')
    expect(source).toContain('target.browserWebSocketUrl = selected.webSocketDebuggerUrl')
    expect(source).toContain('target.browserTargetId?.trim()')
    expect(source).toContain('target.browserEndpoint?.trim()')
    expect(source).toContain('genericTargetTypes')
    expect(source).toContain('v-for="field in targetFields(target)"')
    expect(source).toContain('profile: { ...target.profile }')
  })

  it('preserves the exact captured Win32 title and class identity', () => {
    expect(source).toContain('windowTitle: target.windowTitle,')
    expect(source).toContain('windowClass: target.windowClass,')
    expect(source).not.toContain('windowTitle: target.windowTitle.trim(),')
    expect(source).not.toContain('windowClass: target.windowClass.trim(),')
    expect(source).toContain("target.windowTitleMatch = 'exact'")
    expect(source).toContain('v-model="target.windowTitleMatch"')
    expect(source).toContain("profileFieldOptions(desktopTargetType.value, 'windowTitleMatch')")
    expect(source).toContain('v-model="target.windowSelection"')
    expect(source).toContain("profileFieldOptions(desktopTargetType.value, 'windowSelection')")
    expect(source).toContain('settingsAutomation.targets.preview_matches')
  })

  it('keeps expandable targets and form controls keyboard accessible', () => {
    expect(source).toContain(':aria-expanded="expandedSlot === target.slot"')
    expect(source).toContain(':aria-controls="`automation-target-${target.slot}`"')
    expect(source).toContain('<UFormField')
    expect(source).toContain(':loading="busy[target.slot]"')
  })

  it('keeps dense target identity and window selectors in a single-column flow', () => {
    expect(source).toContain('data-testid="automation-target-core-fields" class="space-y-4"')
    expect(source).toContain('data-testid="automation-target-window-fields" class="space-y-4"')
  })

  it('activates and authorizes a captured target in the same process', () => {
    expect(source).toContain('await backend.automation.grantWorkflowConsent(target.slot)')
    expect(source).not.toContain('captureFeedback.activationRequired')
    expect(source).not.toContain('settingsAutomation.capture.runtime_activation_required')
  })

  it('makes desktop targets follow the active mouse calibration unless explicitly overridden', () => {
    expect(source).toContain('data-testid="automation-target-calibration-inheritance"')
    expect(source).toContain('store.activeMouseCounts360')
    expect(source).toContain('store.data?.ui.activeMouseProfile')
    expect(source).toContain("mouseCalibrationMode: 'active' | 'custom'")
    expect(source).toContain("target.mouseCalibrationMode === 'active' ? 0 : target.mouseCounts360")
    expect(source).toContain("query: { section: 'input' }")
  })
})
