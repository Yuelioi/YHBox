import { describe, expect, it } from 'vitest'
import { asyncOptionPayloadForValue, normalizeAsyncDropdownValue } from './asyncDropdown'

describe('asyncDropdown helpers', () => {
  it('normalizes Nuxt UI selected item objects to their value', () => {
    expect(normalizeAsyncDropdownValue({ value: 'adb-1', label: 'Pixel' })).toBe('adb-1')
    expect(normalizeAsyncDropdownValue('manual')).toBe('manual')
  })

  it('resolves selected option metadata by string-equivalent value', () => {
    expect(
      asyncOptionPayloadForValue(
        [{ value: 123, label: 'Tab', meta: { ws: 'ws://page' } }],
        '123',
      ),
    ).toEqual({ value: '123', meta: { ws: 'ws://page' } })
  })

  it('returns null when selected option has no metadata', () => {
    expect(asyncOptionPayloadForValue([{ value: 'manual' }], 'manual')).toBeNull()
  })
})
