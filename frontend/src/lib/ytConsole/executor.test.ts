import { describe, it, expect } from 'vitest'
import { runConsoleScript, type NodeModel } from './executor'

// 造一份最小 model: 两个主图节点 + 一个子图节点。
function model(): NodeModel[] {
  return [
    {
      id: 'a',
      kind: 'Sleep',
      label: '',
      sgID: null,
      literal: { JitterPct: 0 },
      defaults: { Duration: 1000, JitterPct: 0 },
      specPins: { Duration: 'number', JitterPct: 'number' },
    },
    {
      id: 'b',
      kind: 'Sleep',
      label: '',
      sgID: null,
      literal: {},
      defaults: { Duration: 1000, JitterPct: 0 },
      specPins: { Duration: 'number', JitterPct: 'number' },
    },
    {
      id: 'c',
      kind: 'ClickTemplate',
      label: '',
      sgID: 'sub-1',
      literal: { TimeoutMs: 5000 },
      defaults: { TimeoutMs: 5000, Threshold: 0.85 },
      specPins: { TimeoutMs: 'number', Threshold: 'number', MatchMode: 'string' },
    },
  ]
}
const ctx = (nodes = model(), selectedIds: string[] = []) => ({
  nodes,
  selectedIds,
  container: { id: 'cont-1', name: 'C' },
})

describe('runConsoleScript', () => {
  it('filter + set 命中节点 → applied 正确, 计数对', () => {
    const r = runConsoleScript(
      `yt.nodes.filter(n => n.has('JitterPct')).forEach(n => n.set('JitterPct', 10))`,
      ctx(),
    )
    expect(r.error).toBeUndefined()
    expect(r.applied).toEqual([
      { nodeId: 'a', sgID: null, pin: 'JitterPct', value: 10 },
      { nodeId: 'b', sgID: null, pin: 'JitterPct', value: 10 },
    ])
    expect(r.nodeCount).toBe(2)
    expect(r.pinCount).toBe(2)
  })

  it('set spec 没有的 pin → rejected, 不进 applied', () => {
    const r = runConsoleScript(`yt.nodes[0].set('Nope', 1)`, ctx())
    expect(r.applied).toEqual([])
    expect(r.rejected).toHaveLength(1)
    expect(r.rejected[0]).toMatchObject({
      nodeId: 'a',
      pin: 'Nope',
      reason: expect.stringContaining('pin'),
    })
  })

  it('同 pin 多次 set → 只留最后值, 算 1 个改动', () => {
    const r = runConsoleScript(
      `yt.nodes[0].set('JitterPct', 1); yt.nodes[0].set('JitterPct', 2)`,
      ctx(),
    )
    expect(r.applied).toEqual([{ nodeId: 'a', sgID: null, pin: 'JitterPct', value: 2 }])
    expect(r.pinCount).toBe(1)
  })

  it('get: 已 set 取 set 的; 否则 literal; 否则默认; 无则 undefined', () => {
    const r = runConsoleScript(
      `yt.log(yt.nodes[1].get('JitterPct')); // literal 没填 → 默认 0
       yt.nodes[1].set('JitterPct', 7);
       yt.log(yt.nodes[1].get('JitterPct')); // 读自己刚 set 的 → 7
       yt.log(String(yt.nodes[0].get('NoSuch')))`, // 无 literal 无默认 → undefined
      ctx(),
    )
    expect(r.logs).toEqual(['0', '7', 'undefined'])
  })

  it('按原值算: JitterPct *1.1 (b 用默认 0)', () => {
    const r = runConsoleScript(
      `yt.nodes.filter(n => n.kind==='Sleep').forEach(n => n.set('JitterPct', n.get('JitterPct') + 5))`,
      ctx(),
    )
    expect(r.applied).toEqual([
      { nodeId: 'a', sgID: null, pin: 'JitterPct', value: 5 },
      { nodeId: 'b', sgID: null, pin: 'JitterPct', value: 5 },
    ])
  })

  it('归一: number set 字符串数字 → number; NaN/Infinity → rejected', () => {
    const r = runConsoleScript(
      `yt.nodes[0].set('Duration', '3000');
       yt.nodes[1].set('Duration', 1/0);
       yt.nodes[1].set('JitterPct', 'abc')`,
      ctx(),
    )
    expect(r.applied).toEqual([{ nodeId: 'a', sgID: null, pin: 'Duration', value: 3000 }])
    expect(r.rejected.map((x) => x.pin).sort()).toEqual(['Duration', 'JitterPct'])
  })

  it('bool 归一: "false" → false (不是 true), 2 → rejected', () => {
    const nodes = model()
    nodes[0].specPins = { Flag: 'bool' }
    nodes[0].literal = {}
    nodes[0].defaults = {}
    const r = runConsoleScript(
      `yt.nodes[0].set('Flag', 'false'); yt.nodes[0].set('Flag', 'false')`, // 后写覆盖, 仍 false
      ctx(nodes),
    )
    expect(r.applied).toEqual([{ nodeId: 'a', sgID: null, pin: 'Flag', value: false }])
    const r2 = runConsoleScript(`yt.nodes[0].set('Flag', 2)`, ctx(nodes))
    expect(r2.applied).toEqual([])
    expect(r2.rejected).toHaveLength(1)
  })

  it('set 被拒后 get → 读到改前值 (不是被拒值)', () => {
    const r = runConsoleScript(
      `yt.nodes[0].set('JitterPct', 'abc'); // 被拒, 不写 overlay
       yt.log(String(yt.nodes[0].get('JitterPct')))`, // 仍读 literal 的 0
      ctx(),
    )
    expect(r.logs).toEqual(['0'])
    expect(r.applied).toEqual([])
  })

  it('脚本抛异常 → applied 为空 (零变更), error 带信息', () => {
    const r = runConsoleScript(`yt.nodes[0].set('JitterPct', 1); throw new Error('boom')`, ctx())
    expect(r.applied).toEqual([])
    expect(r.error).toContain('boom')
  })

  it('yt.selected 仅选中子集', () => {
    const r = runConsoleScript(
      `yt.selected.forEach(n => n.set('JitterPct', 9))`,
      ctx(model(), ['b']),
    )
    expect(r.applied).toEqual([{ nodeId: 'b', sgID: null, pin: 'JitterPct', value: 9 }])
  })

  it('handle 真只读: 改 n.kind 不生效 (strict 下抛错被 catch 成 error)', () => {
    const r = runConsoleScript(`yt.nodes[0].kind = 'X'`, ctx())
    // strict mode 下写 getter-only 抛 TypeError → 视为脚本异常, 零变更
    expect(r.applied).toEqual([])
    expect(r.error).toBeTruthy()
  })

  it('0 改动如实报告', () => {
    const r = runConsoleScript(
      `yt.nodes.filter(n => n.kind === 'Nonexistent').forEach(n => n.set('X', 1))`,
      ctx(),
    )
    expect(r.applied).toEqual([])
    expect(r.nodeCount).toBe(0)
    expect(r.pinCount).toBe(0)
    expect(r.error).toBeUndefined()
  })
})
