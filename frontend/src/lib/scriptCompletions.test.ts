// scriptCompletions.test.ts — 节点函数补全签名推导 (非 exec pin 进签名, exec pin 排除)
// + 语法快速反馈与 $变量引用提取的纯函数。
import { describe, it, expect } from 'vitest'
import type { Spec } from '@bindings/yotta/internal/node'
import {
  nodeFnCompletions,
  scriptExitItemsForKind,
  scriptDollarRefs,
  scriptSyntaxErrors,
  SUGAR_COMPLETIONS,
} from './scriptCompletions'

function fakeSpec(
  kind: string,
  inputs: { name: string; type: string }[],
  outputs: { name: string; type: string }[] = [],
): Spec {
  return { kind, inputs, outputs } as unknown as Spec
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
      expect.arrayContaining(['params.get', 'sleep', 'log.info', 'Exit.Found']),
    )
    // 变量读写不走糖 (用 $hp / GetVar/SetVar 节点函数), vars.* 已删。
    expect(labels).not.toContain('vars.get')
    expect(SUGAR_COMPLETIONS.every((c) => c.type === 'function' || c.type === 'constant')).toBe(
      true,
    )
  })
})

describe('scriptExitItemsForKind', () => {
  it('uses node Spec exec outputs and maps standard exits to Exit constants', () => {
    const specs = new Map<string, Spec>([
      [
        'CheckTemplate',
        fakeSpec(
          'CheckTemplate',
          [],
          [
            { name: 'Found', type: 'Exec' },
            { name: 'NotFound', type: 'Exec' },
            { name: 'Point', type: 'Point' },
          ],
        ),
      ],
    ])

    expect(scriptExitItemsForKind('CheckTemplate', specs).map((i) => i.insert)).toEqual([
      'Exit.Found',
      'Exit.NotFound',
    ])
  })

  it('falls back to string literals for non-standard exit names', () => {
    const specs = new Map<string, Spec>([
      ['CustomNode', fakeSpec('CustomNode', [], [{ name: 'Skipped', type: 'Exec' }])],
    ])

    expect(scriptExitItemsForKind('CustomNode', specs).map((i) => i.insert)).toEqual(['"Skipped"'])
  })
})

describe('scriptSyntaxErrors', () => {
  it('合法脚本无诊断', () => {
    expect(scriptSyntaxErrors('let a = 1\nif (a > 0) {\n  sleep(100)\n}\n')).toEqual([])
  })

  it('空文档无诊断', () => {
    expect(scriptSyntaxErrors('')).toEqual([])
  })

  it('括号不闭合报错且行号对', () => {
    const errs = scriptSyntaxErrors('let a = 1\nif (a > 0 {\n}')
    expect(errs.length).toBeGreaterThan(0)
    expect(errs[0].line).toBe(2)
  })

  it('同一行级联错误只报一次', () => {
    const errs = scriptSyntaxErrors('if ((( {')
    const lines = errs.map((e) => e.line)
    expect(new Set(lines).size).toBe(lines.length)
  })
})

describe('scriptDollarRefs', () => {
  it('提取 $引用, 字符串与注释里的 $ 不命中', () => {
    const { refs } = scriptDollarRefs('let a = $hp\nlog.info("$fake")\n// $comment\n')
    expect(refs.map((r) => r.name)).toEqual(['hp'])
  })

  it('本地 let $x 定义记入 defined, 不算外部引用', () => {
    const { refs, defined } = scriptDollarRefs('let $x = 1\nlog.info($x + $hp)')
    expect(defined.has('x')).toBe(true)
    expect(refs.map((r) => r.name).sort()).toEqual(['hp', 'x'])
  })

  it('裸 $ 不命中', () => {
    const { refs } = scriptDollarRefs('let $ = 1')
    expect(refs).toEqual([])
  })
})
