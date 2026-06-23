import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

const updateMock = vi.fn().mockResolvedValue(true)
vi.mock('@/lib/backend', () => ({
  backend: { settings: { update: (...a: unknown[]) => updateMock(...a), get: vi.fn() } },
}))

import { useSettingsStore } from './settings'

const conn = { id: 'a', label: 'X', protocol: 'openai' as const, baseURL: '', apiKey: '' }

describe('settings store · patchAIConnections', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    updateMock.mockClear()
  })

  it('patches the connections array, omitting default when not given', async () => {
    const store = useSettingsStore()
    await store.patchAIConnections([conn])
    expect(updateMock).toHaveBeenCalledTimes(1)
    const patch = updateMock.mock.calls[0][0] as { ai: { connections: unknown[]; default?: string } }
    expect(patch.ai.connections).toEqual([conn])
    expect('default' in patch.ai).toBe(false)
  })

  it('includes an explicit empty default (delete-clears-default)', async () => {
    const store = useSettingsStore()
    await store.patchAIConnections([], '')
    const patch = updateMock.mock.calls[0][0] as { ai: { connections: unknown[]; default?: string } }
    expect(patch.ai).toEqual({ connections: [], default: '' })
  })
})
