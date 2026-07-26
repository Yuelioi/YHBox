import { describe, expect, it } from 'vitest'
import { collapseSelectionErrorReason } from './collapseSelectionError'

describe('collapse selection error presentation', () => {
  it('maps internal boundary failures to localizable reasons', () => {
    expect(
      collapseSelectionErrorReason(
        new Error('selection needs one execution entry and at least one signal exit'),
      ),
    ).toBe('missing_boundary')
    expect(
      collapseSelectionErrorReason(new Error('selection has multiple execution entries')),
    ).toBe('multiple_entry')
  })

  it('does not expose an unknown internal error message', () => {
    expect(collapseSelectionErrorReason(new Error('internal implementation detail'))).toBe(
      'unknown',
    )
  })
})
