// frontend/src/composables/editor/useSidebarPrefs.spec.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { nextTick } from 'vue'

describe('useSidebarPrefs', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.resetModules()
  })

  it('loads defaults when localStorage empty', async () => {
    const { useSidebarPrefs } = await import('./useSidebarPrefs')
    const { prefs } = useSidebarPrefs()
    expect(prefs.value.leftDrawer).toBe(null)
    expect(prefs.value.varsExpanded).toBe(true)
  })

  it('persists changes to localStorage', async () => {
    const { useSidebarPrefs, SIDEBAR_PREFS_KEY } = await import('./useSidebarPrefs')
    const { prefs } = useSidebarPrefs()
    prefs.value.varsExpanded = false
    await nextTick()
    const raw = localStorage.getItem(SIDEBAR_PREFS_KEY)
    expect(raw).toBeTruthy()
    expect(JSON.parse(raw!).varsExpanded).toBe(false)
  })

  it('load restores from localStorage', async () => {
    const { SIDEBAR_PREFS_KEY } = await import('./useSidebarPrefs')
    localStorage.setItem(SIDEBAR_PREFS_KEY, JSON.stringify({ inspectorCollapsed: true }))
    vi.resetModules()
    const { useSidebarPrefs } = await import('./useSidebarPrefs')
    const { prefs } = useSidebarPrefs()
    expect(prefs.value.inspectorCollapsed).toBe(true)
    expect(prefs.value.varsExpanded).toBe(true)
  })
})
