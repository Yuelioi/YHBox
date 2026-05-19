import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

describe('discoveryStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.resetModules()
  })

  it('starts empty when localStorage empty', async () => {
    const { useDiscoveryStore } = await import('./discovery')
    const s = useDiscoveryStore()
    s.loadFromLocalStorage()
    expect(s.favorites).toEqual([])
    expect(s.recent).toEqual([])
  })

  it('toggleFavorite adds kind not in list, persists', async () => {
    const { useDiscoveryStore } = await import('./discovery')
    const s = useDiscoveryStore()
    s.toggleFavorite('SetVar')
    expect(s.favorites).toEqual(['SetVar'])
    expect(JSON.parse(localStorage.getItem('yhfish.favorites')!)).toEqual(['SetVar'])
  })

  it('toggleFavorite removes kind if already in list', async () => {
    const { useDiscoveryStore } = await import('./discovery')
    const s = useDiscoveryStore()
    s.toggleFavorite('SetVar')
    s.toggleFavorite('SetVar')
    expect(s.favorites).toEqual([])
  })

  it('pushRecent prepends + LRU max 8 + dedup', async () => {
    const { useDiscoveryStore } = await import('./discovery')
    const s = useDiscoveryStore()
    for (const k of ['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I']) s.pushRecent(k)
    // I is newest, A pushed out
    expect(s.recent).toEqual(['I', 'H', 'G', 'F', 'E', 'D', 'C', 'B'])
    expect(s.recent.length).toBe(8)
  })

  it('pushRecent moves existing kind to front', async () => {
    const { useDiscoveryStore } = await import('./discovery')
    const s = useDiscoveryStore()
    s.pushRecent('A')
    s.pushRecent('B')
    s.pushRecent('A')
    expect(s.recent).toEqual(['A', 'B'])
  })

  it('loadFromLocalStorage restores from existing keys', async () => {
    localStorage.setItem('yhfish.favorites', JSON.stringify(['Loop', 'If']))
    localStorage.setItem('yhfish.recent', JSON.stringify(['Add', 'Log']))
    const { useDiscoveryStore } = await import('./discovery')
    const s = useDiscoveryStore()
    s.loadFromLocalStorage()
    expect(s.favorites).toEqual(['Loop', 'If'])
    expect(s.recent).toEqual(['Add', 'Log'])
  })
})
