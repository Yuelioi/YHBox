import { nextTick, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useRovingAssetList } from './useRovingAssetList'

describe('useRovingAssetList', () => {
  it('keeps exactly one visible item in the tab order', async () => {
    const ids = ref(['a', 'b', 'c'])
    const list = useRovingAssetList(ids)

    expect(list.isTabStop('a')).toBe(true)
    expect(list.isTabStop('b')).toBe(false)

    ids.value = ['b', 'c']
    await nextTick()
    expect(list.isTabStop('b')).toBe(true)
  })

  it('moves with arrow, home, and end keys', () => {
    const ids = ref(['a', 'b', 'c'])
    const list = useRovingAssetList(ids)
    const preventDefault = vi.fn()

    expect(
      list.move('a', {
        key: 'End',
        preventDefault,
        currentTarget: null,
      } as unknown as KeyboardEvent),
    ).toBe(true)
    expect(list.activeId.value).toBe('c')

    expect(
      list.move('c', {
        key: 'ArrowUp',
        preventDefault,
        currentTarget: null,
      } as unknown as KeyboardEvent),
    ).toBe(true)
    expect(list.activeId.value).toBe('b')
    expect(preventDefault).toHaveBeenCalledTimes(2)
  })
})
