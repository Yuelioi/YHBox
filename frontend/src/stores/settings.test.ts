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

const conn = { id: 'a', label: 'X', protocol: 'openai' as const, baseURL: '', apiKey: '' }

describe('settings store · patchAIConnections', () => {
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
      ai: { connections: [], default: '' },
      mcp: { armed: false },
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
      ai: { connections: [], default: '' },
      mcp: { armed: false },
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

  it('patches the connections array, omitting default when not given', async () => {
    const store = useSettingsStore()
    await store.patchAIConnections([conn])
    expect(updateMock).toHaveBeenCalledTimes(1)
    const patch = updateMock.mock.calls[0][0] as {
      ai: { connections: unknown[]; default?: string }
    }
    expect(patch.ai.connections).toEqual([conn])
    expect('default' in patch.ai).toBe(false)
  })

  it('includes an explicit empty default (delete-clears-default)', async () => {
    const store = useSettingsStore()
    await store.patchAIConnections([], '')
    const patch = updateMock.mock.calls[0][0] as {
      ai: { connections: unknown[]; default?: string }
    }
    expect(patch.ai).toEqual({ connections: [], default: '' })
  })
})
