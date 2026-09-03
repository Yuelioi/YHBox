import { describe, expect, it } from 'vitest'
import { moveListSelection, numberedSelectionIndex } from './listKeyboardSelection'

const event = (key: string, overrides: Partial<KeyboardEvent> = {}) =>
  ({
    key,
    altKey: false,
    ctrlKey: false,
    metaKey: false,
    shiftKey: false,
    isComposing: false,
    ...overrides,
  }) as KeyboardEvent

describe('numbered list keyboard selection', () => {
  it('wraps arrow selection through the visible results', () => {
    expect(moveListSelection(0, -1, 3)).toBe(2)
    expect(moveListSelection(2, 1, 3)).toBe(0)
    expect(moveListSelection(0, 1, 0)).toBe(0)
  })

  it('maps unmodified 1–9 keys to visible result indexes', () => {
    expect(numberedSelectionIndex(event('1'), 9)).toBe(0)
    expect(numberedSelectionIndex(event('9'), 9)).toBe(8)
    expect(numberedSelectionIndex(event('4'), 3)).toBeUndefined()
    expect(numberedSelectionIndex(event('0'), 10)).toBeUndefined()
  })

  it('does not intercept shortcuts or IME composition', () => {
    expect(numberedSelectionIndex(event('1', { ctrlKey: true }), 9)).toBeUndefined()
    expect(numberedSelectionIndex(event('1', { shiftKey: true }), 9)).toBeUndefined()
    expect(numberedSelectionIndex(event('1', { isComposing: true }), 9)).toBeUndefined()
  })
})
