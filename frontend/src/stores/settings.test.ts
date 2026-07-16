import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

const updateMock = vi.fn().mockResolvedValue(true)
const getMock = vi.fn()
const onSettingsChangedMock = vi.fn()
vi.mock('@/lib/backend', () => ({
  backend: {
    settings: {
      update: (...a: unknown[]) => updateMock(...a),
      get: (...a: unknown[]) => getMock(...a),
    },
    events: { onSettingsChanged: (...a: unknown[]) => onSettingsChangedMock(...a) },
  },
}))

import { useSettingsStore, type Settings } from './settings'

const profile = {
  slot: 'primary',
  label: 'Primary',
  provider: 'openai-responses' as const,
  model: 'gpt-test',
  maxOutputTokens: 4096,
  capabilities: {
    structuredOutput: true,
    toolCalling: false,
    parallelTools: false,
    background: false,
    zeroRetention: false,
  },
  evaluation: 'unverified' as const,
}

describe('settings store · patchAIProfiles', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    updateMock.mockClear()
    updateMock.mockResolvedValue(true)
    getMock.mockReset()
    onSettingsChangedMock.mockReset()
  })

  it('merges a successful void-backed update and exposes save state', async () => {
    const store = useSettingsStore()
    store.data = {
      ui: { autostart: false },
      locale: 'zh',
      capture: {},
      ai: { profiles: [] },
    } as unknown as Settings

    await expect(store.patch({ ui: { autostart: true } })).resolves.toBe(true)
    expect(store.data.ui.autostart).toBe(true)
    expect(store.saveState).toBe('saved')
    expect(store.lastSavedAt).toEqual(expect.any(Number))
  })

  it('serializes rapid patches in call order', async () => {
    const releases: Array<() => void> = []
    updateMock.mockImplementation(
      () => new Promise<boolean>((resolve) => releases.push(() => resolve(true))),
    )
    const store = useSettingsStore()
    store.data = {
      ui: { launcherDisplay: 'both' },
      locale: 'zh',
      capture: {},
      ai: { profiles: [] },
    } as unknown as Settings

    const first = store.patch({ ui: { launcherDisplay: 'icon' } })
    const second = store.patch({ ui: { launcherDisplay: 'text' } })
    await vi.waitFor(() => expect(releases).toHaveLength(1))
    releases.shift()?.()
    await vi.waitFor(() => expect(releases).toHaveLength(1))
    releases.shift()?.()
    await Promise.all([first, second])

    expect(updateMock.mock.calls.map((call) => call[0])).toEqual([
      { ui: { launcherDisplay: 'icon' } },
      { ui: { launcherDisplay: 'text' } },
    ])
    expect(store.data.ui.launcherDisplay).toBe('text')
  })

  it('patches the complete profile array', async () => {
    const store = useSettingsStore()
    await store.patchAIProfiles([profile])
    expect(updateMock).toHaveBeenCalledTimes(1)
    const patch = updateMock.mock.calls[0][0] as { ai: { profiles: unknown[] } }
    expect(patch.ai.profiles).toEqual([profile])
  })

  it('patches exact automation targets as one installation snapshot', async () => {
    const store = useSettingsStore()
    const target = {
      slot: 'editor-input',
      label: 'Editor input',
      applicationSlot: 'editor',
      windowTitle: 'Editor',
      windowClass: 'EditorWindow',
      inputBackend: 'sendinput' as const,
      captureBackend: 'gdi' as const,
      resolveTimeoutMilliseconds: 3000,
    }
    await store.patchAutomationTargets([target])
    expect(updateMock).toHaveBeenCalledTimes(1)
    expect(updateMock.mock.calls[0][0]).toEqual({ automation: { win32Targets: [target] } })
  })
})
