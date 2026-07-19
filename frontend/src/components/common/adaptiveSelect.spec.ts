import { describe, expect, it } from 'vitest'
import { adaptiveSelectWidth, selectLabelWidth } from './adaptiveSelect'

describe('adaptiveSelectWidth', () => {
  it('uses the longest visible option including wide characters', () => {
    const items = [
      { label: '通用', value: 'generic' },
      { label: '浏览器自动化', value: 'browser' },
    ]

    expect(selectLabelWidth(items)).toBe(12)
    expect(adaptiveSelectWidth(items)).toBe(17)
  })

  it('supports custom label keys and nested groups', () => {
    const items = [[{ name: 'Windows automation', value: 'windows' }], [{ type: 'separator' }]]

    expect(selectLabelWidth(items, 'name')).toBe(18)
  })

  it('respects minimum and maximum widths', () => {
    expect(adaptiveSelectWidth(['A'], 'label', 12, 40)).toBe(12)
    expect(adaptiveSelectWidth(['A'.repeat(100)], 'label', 12, 40)).toBe(40)
  })
})
