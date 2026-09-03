import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const toolViews = [
  'MouseHUDView.vue',
  'RecordingHUDView.vue',
  'CalibrationHUDView.vue',
  'FloatingLauncherView.vue',
  'ScreenPickerView.vue',
]

const readSource = (path: string) => readFileSync(join(process.cwd(), path), 'utf8')

describe('standalone window presentation contract', () => {
  it.each(toolViews)('%s uses the shared HUD shell and Nuxt UI buttons', (filename) => {
    const source = readSource(`src/views/tools/${filename}`)

    expect(source).toContain('<HudShell')
    expect(source).not.toMatch(/<button(?:\s|>)/)
  })

  it('keeps window dragging and close accessibility in the shared shell', () => {
    const source = readSource('src/components/tools/HudShell.vue')

    expect(source).toContain('--wails-draggable: drag')
    expect(source).toContain('--wails-draggable: no-drag')
    expect(source).toContain(':aria-label="resolvedCloseTitle"')
  })

  it('shares the semantic state panel between live HUDs', () => {
    for (const filename of ['RecordingHUDView.vue', 'CalibrationHUDView.vue']) {
      expect(readSource(`src/views/tools/${filename}`)).toContain('<HudStatePanel')
    }
  })

  it('keeps settings focused on editing while the real launcher provides live feedback', () => {
    const floatingSource = readSource('src/views/tools/FloatingLauncherView.vue')
    const settingsSource = readSource('src/views/SettingsLauncher.vue')

    expect(floatingSource).toContain('<LauncherSurface')
    expect(settingsSource).not.toContain('<LauncherSurface')
    expect(settingsSource).not.toContain('launcher-preview')
    expect(settingsSource).toContain('<div class="settings-page">')
    expect(settingsSource).not.toContain('settings-page--wide')
    expect(settingsSource).toContain('backend.tools.openLauncher()')
    expect(settingsSource).toContain('<USwitch')
    expect(settingsSource).toContain('launcherSlotHotkeysDisabled')
    expect(settingsSource).not.toContain('configure_hotkey')
    expect(settingsSource).not.toContain('i-tabler-arrow-up')
    expect(settingsSource).not.toContain('i-tabler-arrow-down')
    expect(floatingSource).toContain('backend.tools.openLauncherSettings()')
  })

  it('keeps the launcher keyboard and stale-state interaction contract', () => {
    const source = readSource('src/views/tools/FloatingLauncherView.vue')
    const surface = readSource('src/components/launcher/LauncherSurface.vue')

    expect(source).toContain("event.key === 'ArrowDown'")
    expect(source).toContain("event.key === 'Enter'")
    expect(source).toContain('query.value += event.key')
    expect(source).toContain('backend.tools.refreshLauncherHotkeys()')
    expect(source).not.toContain("event.key >= '1'")
    expect(surface).toContain('launcher-command--stale')
    expect(surface).toContain('launcher-surface--xlarge')
    expect(surface).toContain('.launcher-command--separator-before::before')
    expect(surface).toContain(':aria-disabled="item.stale')
  })

  it('refreshes launcher settings and Workflow sources after cross-window settings changes', () => {
    const source = readSource('src/views/tools/FloatingLauncherView.vue')

    expect(source).toMatch(
      /function refreshLauncherData[\s\S]*settingsStore\.load\(\)[\s\S]*workflowTransport\.listSources\(\)/,
    )
    expect(source).toMatch(/Events\.On\([\s\S]*'settings:changed'[\s\S]*refreshLauncherData/)
  })

  it('polls through a stale short-run snapshot instead of leaving the launcher running forever', () => {
    const source = readSource('src/views/tools/FloatingLauncherView.vue')

    expect(source).toContain('pollTerminalRunStatus(')
    expect(source).toContain('requestedRunId.value !== outcome.runId')
  })

  it('lets a running launcher command stop the exact Run without closing the launcher', () => {
    const source = readSource('src/views/tools/FloatingLauncherView.vue')
    const surface = readSource('src/components/launcher/LauncherSurface.vue')

    expect(source).toContain('workflowTransport.cancelRun(runId)')
    expect(source).toContain("settleRequest('cancelled')")
    expect(surface).toContain("statusFor(item.workflowId) === 'running'")
    expect(surface).toContain("case 'cancelled':")
  })

  it('restores active workflow Runs and tracks Runs started outside the launcher window', () => {
    const source = readSource('src/views/tools/FloatingLauncherView.vue')

    expect(source).toContain('workflowTransport.getActiveSourceRuns(')
    expect(source).toContain('event.workflowId')
    expect(source).toContain('activeRuns.value')
  })

  it('shows a bottom stop tip using the live configured slot shortcut', () => {
    const source = readSource('src/views/tools/FloatingLauncherView.vue')

    expect(source).toContain('class="launcher-stop-tip"')
    expect(source).toContain('stopTipItem.shortcut')
    expect(source).toContain("t('floatingLauncher.stop_tip_hotkey'")
  })

  it('lets live HUD state regions fill spare height instead of pushing it above actions', () => {
    for (const filename of ['RecordingHUDView.vue', 'CalibrationHUDView.vue']) {
      const source = readSource(`src/views/tools/${filename}`)

      expect(source).toContain('class="live-hud__stage"')
      expect(source).toMatch(/\.live-hud__stage\s*\{[^}]*flex:\s*1;/s)
      expect(source).not.toContain('class="mt-auto flex items-center')
    }
  })

  it('reconciles the recording HUD from a monotonic backend snapshot after late mount', () => {
    const source = readSource('src/views/tools/RecordingHUDView.vue')

    expect(source).toContain('applySnapshot(await backend.recording.getState())')
    expect(source).toContain('if (nextRevision < revision) return')
    expect(source).toContain("st?.mode === 'simple' || st?.mode === 'precise'")
  })

  it('keeps recording shortcuts small and live while removing redundant subtitle copy', () => {
    const source = readSource('src/views/tools/RecordingHUDView.vue')

    expect(source).not.toContain("t('recordingHud.subtitle')")
    expect(source).toContain('backend.events.onHotkeyChanged')
    expect(source).toContain("hotkeys.keyFor('recording.cancel', 'F7')")
    expect(source).toContain('class="recording-shortcut"')
    expect(source).toMatch(/\.recording-shortcut\s*\{[^}]*font-size:\s*9px;/s)
  })
})
