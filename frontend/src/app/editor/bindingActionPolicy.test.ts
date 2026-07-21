import { describe, expect, it } from 'vitest'
import { bindingActionPolicy } from './bindingActionPolicy'

describe('binding action policy', () => {
  it('uses one reset action when a bound input has a contract default', () => {
    expect(bindingActionPolicy({ required: true, hasDefault: true, bound: true })).toEqual({
      resetToDefault: true,
      clear: false,
    })
  })

  it('does not offer an impossible clear action for required inputs without defaults', () => {
    expect(bindingActionPolicy({ required: true, hasDefault: false, bound: true })).toEqual({
      resetToDefault: false,
      clear: false,
    })
  })

  it('only offers clear for a bound optional input without a default', () => {
    expect(bindingActionPolicy({ required: false, hasDefault: false, bound: true })).toEqual({
      resetToDefault: false,
      clear: true,
    })
  })
})
