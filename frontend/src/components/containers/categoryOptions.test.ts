import { describe, expect, it } from 'vitest'
import { addCreatedCategory, uniqueCategoryOptions } from './categoryOptions'

describe('category options', () => {
  it('adds a created category so InputMenu can select it immediately', () => {
    const result = addCreatedCategory(['daily'], ' fishing ')

    expect(result.categories).toEqual(['daily', 'fishing'])
    expect(result.value).toBe('fishing')
  })

  it('merges existing, created and current category values without blanks', () => {
    expect(uniqueCategoryOptions(['daily', ''], ['combat'], ['daily'], [' fishing '])).toEqual([
      'daily',
      'combat',
      'fishing',
    ])
  })
})
