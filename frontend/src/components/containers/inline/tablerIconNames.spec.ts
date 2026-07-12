import { describe, expect, it } from 'vitest'
import names from 'virtual:tabler-icon-names'

describe('Tabler icon name index', () => {
  it('contains only a sorted, unique name index', () => {
    expect(names.length).toBeGreaterThan(5_000)
    expect(names).toEqual([...names].sort())
    expect(new Set(names).size).toBe(names.length)
    expect(names.every((name) => /^[a-z0-9-]+$/.test(name))).toBe(true)
  })

  it('contains common picker icons', () => {
    expect(names).toContain('search')
    expect(names).toContain('settings')
  })
})
