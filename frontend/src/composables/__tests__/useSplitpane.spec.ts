import { describe, it, expect, beforeEach } from 'vitest'
import { useSplitpane } from '../useSplitpane'

describe('useSplitpane', () => {
  beforeEach(() => localStorage.clear())

  it('returns default when no localStorage value', () => {
    const sp = useSplitpane('test.left', { default: 280, min: 200, max: 480 })
    expect(sp.width.value).toBe(280)
  })

  it('restores value from localStorage', () => {
    localStorage.setItem('test.left', '350')
    const sp = useSplitpane('test.left', { default: 280, min: 200, max: 480 })
    expect(sp.width.value).toBe(350)
  })

  it('setWidth clamps to min/max and persists', () => {
    const sp = useSplitpane('test.left', { default: 280, min: 200, max: 480 })
    sp.setWidth(600)
    expect(sp.width.value).toBe(480)
    expect(localStorage.getItem('test.left')).toBe('480')
    sp.setWidth(100)
    expect(sp.width.value).toBe(200)
  })

  it('ignores corrupt localStorage value', () => {
    localStorage.setItem('test.left', 'not-a-number')
    const sp = useSplitpane('test.left', { default: 280, min: 200, max: 480 })
    expect(sp.width.value).toBe(280)
  })
})
