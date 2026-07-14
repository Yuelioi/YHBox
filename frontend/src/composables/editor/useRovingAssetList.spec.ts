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

  it('moves focus across category listboxes in the same browser', () => {
    document.body.innerHTML = `
      <div data-asset-browser-list>
        <div data-asset-list><button data-asset-option data-asset-id="a"></button></div>
        <div data-asset-list><button data-asset-option data-asset-id="b"></button></div>
      </div>
    `
    const first = document.querySelector<HTMLElement>('[data-asset-id="a"]')!
    const second = document.querySelector<HTMLElement>('[data-asset-id="b"]')!
    const list = useRovingAssetList(ref(['a', 'b']))

    list.move('a', {
      key: 'ArrowDown',
      preventDefault: vi.fn(),
      currentTarget: first,
    } as unknown as KeyboardEvent)

    expect(list.activeId.value).toBe('b')
    expect(document.activeElement).toBe(second)
  })
})
