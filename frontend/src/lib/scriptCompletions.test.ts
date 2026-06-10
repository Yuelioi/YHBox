// scriptCompletions.test.ts — 节点函数补全签名推导 (非 exec pin 进签名, exec pin 排除)。
import { describe, it, expect } from 'vitest'
import type { Spec } from '@bindings/yotta/internal/node'
import { nodeFnCompletions, SUGAR_COMPLETIONS } from './scriptCompletions'

function fakeSpec(kind: string, inputs: { name: string; type: string }[]): Spec {
  return { kind, inputs } as unknown as Spec
}

describe('nodeFnCompletions', () => {
  const specs = new Map<string, Spec>([
    [
      'ClickAt',
      fakeSpec('ClickAt', [
        { name: 'In', type: 'Exec' },
        { name: 'XRatio', type: 'Number' },
        { name: 'YRatio', type: 'Number' },
      ]),
    ],
    ['Random', fakeSpec('Random', [{ name: 'Min', type: 'Number' }])],
  ])

  it('detail 含非 exec pin, 不含 exec pin In', () => {
    const items = nodeFnCompletions(['ClickAt'], specs)
    expect(items).toHaveLength(1)
    expect(items[0].label).toBe('ClickAt')
    expect(items[0].detail).toContain('XRatio')
    expect(items[0].detail).toContain('YRatio')
    expect(items[0].detail).not.toContain('In,')
    expect(items[0].detail).toBe('ClickAt({XRatio, YRatio})')
  })

  it('labelOf 提供时拼进 detail', () => {
    const items = nodeFnCompletions(['Random'], specs, () => '随机数')
    expect(items[0].detail).toBe('Random({Min}) · 随机数')
  })

  it('specs 缺 kind 时签名为空括号, 不崩', () => {
    const items = nodeFnCompletions(['Unknown'], specs)
    expect(items[0].detail).toBe('Unknown({})')
  })

  it('apply 是回调 (光标定位用), 不是纯字符串', () => {
    const items = nodeFnCompletions(['ClickAt'], specs)
    expect(typeof items[0].apply).toBe('function')
  })
})

describe('SUGAR_COMPLETIONS', () => {
  it('糖函数全集齐且都是 function 类型', () => {
    const labels = SUGAR_COMPLETIONS.map((c) => c.label)
    expect(labels).toEqual(
      expect.arrayContaining(['vars.get', 'vars.set', 'vars.inc', 'params.get', 'sleep', 'log.info']),
    )
    expect(SUGAR_COMPLETIONS.every((c) => c.type === 'function')).toBe(true)
  })
})
