import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { useListSelection } from './useListSelection'

function setup(ids: string[] = ['a', 'b', 'c', 'd', 'e']) {
  const visible = ref(ids)
  return { visible, sel: useListSelection(visible) }
}

describe('useListSelection', () => {
  it('单击 = 替换选中并设锚点', () => {
    const { sel } = setup()
    sel.click('b')
    sel.click('d')
    expect([...sel.selected.value]).toEqual(['d'])
  })
  it('ctrl 单击 = toggle 加选/减选', () => {
    const { sel } = setup()
    sel.click('a')
    sel.click('c', { ctrl: true })
    expect([...sel.selected.value].sort()).toEqual(['a', 'c'])
    sel.click('a', { ctrl: true })
    expect([...sel.selected.value]).toEqual(['c'])
  })
  it('shift 单击 = 锚点到当前的可见范围 (正反向)', () => {
    const { sel } = setup()
    sel.click('b')
    sel.click('d', { shift: true })
    expect([...sel.selected.value].sort()).toEqual(['b', 'c', 'd'])
    sel.click('a', { shift: true }) // 锚点仍是 b, 反向
    expect([...sel.selected.value].sort()).toEqual(['a', 'b'])
  })
  it('无锚点时 shift 退化为单选', () => {
    const { sel } = setup()
    sel.click('c', { shift: true })
    expect([...sel.selected.value]).toEqual(['c'])
  })
  it('single: 恰好 1 个时给 id, 否则 null', () => {
    const { sel } = setup()
    expect(sel.single.value).toBeNull()
    sel.click('a')
    expect(sel.single.value).toBe('a')
    sel.click('b', { ctrl: true })
    expect(sel.single.value).toBeNull()
  })
  it('可见列表收缩时剔除不可见选中项', async () => {
    const { visible, sel } = setup()
    sel.click('b')
    sel.click('d', { shift: true })
    visible.value = ['b', 'e'] // c、d 被过滤掉
    await nextTick()
    expect([...sel.selected.value]).toEqual(['b'])
  })
  it('clear 清空选中与锚点', () => {
    const { sel } = setup()
    sel.click('a')
    sel.clear()
    expect(sel.selected.value.size).toBe(0)
    sel.click('c', { shift: true }) // 锚点已清, shift 退化单选
    expect([...sel.selected.value]).toEqual(['c'])
  })
})
