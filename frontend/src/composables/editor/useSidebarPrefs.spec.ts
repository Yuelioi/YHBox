import { describe, it, expect, beforeEach } from 'vitest'
import { useSidebarPrefs, SIDEBAR_PREFS_KEY } from './useSidebarPrefs'

describe('useSidebarPrefs', () => {
  beforeEach(() => localStorage.clear())

  it('loads defaults when localStorage empty', () => {
    const { prefs, load } = useSidebarPrefs()
    load()
    expect(prefs.value.leftSidebarCollapsed).toBe(false)
    expect(prefs.value.varsExpanded).toBe(true)
  })

  it('persists changes to localStorage', async () => {
    const { prefs } = useSidebarPrefs()
    prefs.value.varsExpanded = false
    await new Promise(r => setTimeout(r, 0))
    const raw = localStorage.getItem(SIDEBAR_PREFS_KEY)
    expect(raw).toBeTruthy()
    expect(JSON.parse(raw!).varsExpanded).toBe(false)
  })

  it('load restores from localStorage', () => {
    localStorage.setItem(SIDEBAR_PREFS_KEY, JSON.stringify({ inspectorCollapsed: true }))
    const { prefs, load } = useSidebarPrefs()
    load()
    expect(prefs.value.inspectorCollapsed).toBe(true)
    expect(prefs.value.varsExpanded).toBe(true)  // default preserved
  })
})
