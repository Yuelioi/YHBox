import { describe, expect, it } from 'vitest'
import { addCreatedCategory, uniqueCategoryOptions } from './categoryOptions'

describe('category options', () => {
  it('adds a trimmed category for immediate selection', () => {
    expect(addCreatedCategory(['daily'], ' fishing ')).toEqual({
      categories: ['daily', 'fishing'],
      value: 'fishing',
    })
  })

  it('merges category sources without blanks or duplicates', () => {
    expect(uniqueCategoryOptions(['daily', ''], ['combat'], ['daily'], [' fishing '])).toEqual([
      'daily',
      'combat',
      'fishing',
    ])
  })
})
