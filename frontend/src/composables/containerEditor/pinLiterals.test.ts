import { describe, it, expect, beforeAll } from 'vitest'
import { unconnectedDataInPins } from './pinLiterals'
import { register, __resetForTests } from '@/components/containers/nodeRegistry/registry'
import type { PinType } from '@/components/containers/nodeRegistry/index'

const KEYPRESS_DATAIN: Record<string, PinType> = { VK: 'string', Dur: 'number' }

// 动态输入分支是标志驱动 (getSpec(kind)?.dynamicInputs) — 测试注册最小 Expr spec.
beforeAll(() => {
  __resetForTests()
  register({
    kind: 'Expr',
    group: 'purefunc',
    labelZh: 'node.Expr.label',
    description: 'node.Expr.description',
    example: 'node.Expr.example',
    visual: { icon: '', bg: '', border: '' },
    execIn: [],
    execOut: [],
    dataIn: {},
    dataOut: { value: 'any' },
    fields: [],
    defaults: {},
    isPureData: true,
    dynamicInputs: true,
  })
})

describe('unconnectedDataInPins', () => {
  it('无边时返回全部 data-in pin', () => {
    const out = unconnectedDataInPins('KeyPress', KEYPRESS_DATAIN, null, [], 'n1')
    expect(out).toEqual([
      { name: 'VK', type: 'string' },
      { name: 'Dur', type: 'number' },
    ])
  })

  it('一条 data 边连入某 pin → 该 pin 不返', () => {
    const edges = [{ from: 'src.out', to: 'n1.VK' }]
    const out = unconnectedDataInPins('KeyPress', KEYPRESS_DATAIN, null, edges, 'n1')
    expect(out.map((p) => p.name)).toEqual(['Dur'])
  })

  it('exec 边连入 (落 exec-in pin 名) 不影响 data pin 判定', () => {
    const edges = [{ from: 'src.Done', to: 'n1.In' }]
    const out = unconnectedDataInPins('KeyPress', KEYPRESS_DATAIN, null, edges, 'n1')
    expect(out.map((p) => p.name)).toEqual(['VK', 'Dur'])
  })

  it('同一 pin 多条边连入 → 仍只排除 (防御性)', () => {
    const edges = [
      { from: 'a.out', to: 'n1.VK' },
      { from: 'b.out', to: 'n1.VK' },
    ]
    const out = unconnectedDataInPins('KeyPress', KEYPRESS_DATAIN, null, edges, 'n1')
    expect(out.map((p) => p.name)).toEqual(['Dur'])
  })

  it('point 类型原样返回 (scalar 过滤是调用方的事)', () => {
    const out = unconnectedDataInPins('ClickAt', { Pos: 'point' }, null, [], 'n1')
    expect(out).toEqual([{ name: 'Pos', type: 'point' }])
  })

  it('dynamicInputs 节点 (Expr) 并入 config.Inputs[] 未连线动态输入', () => {
    const config = { Inputs: [{ Name: 'a', Type: 'number' }, { Name: 'b', Type: 'number' }] }
    const edges = [{ from: 's.out', to: 'n1.a' }]
    const out = unconnectedDataInPins('Expr', {}, config, edges, 'n1')
    expect(out.map((p) => p.name)).toEqual(['b'])
  })

  it('有专属 section 的 kind (BESPOKE_EDITOR_KINDS) 不暴露通用 literal pin', () => {
    // 这些节点 input 由各自专属 Inspector section / 画布也不出内联框。与存储位置正交。
    const wtDataIn: Record<string, PinType> = { Title: 'string', Class: 'string' }
    expect(unconnectedDataInPins('Win32WindowTarget', wtDataIn, null, [], 'n1')).toEqual([])
    expect(unconnectedDataInPins('MouseCalibration', { Counts360: 'number' }, null, [], 'n2')).toEqual([])
    expect(unconnectedDataInPins('Subgraph', { SubgraphID: 'string' }, null, [], 'n3')).toEqual([])
    expect(unconnectedDataInPins('PlayClip', { ClipID: 'string' }, null, [], 'n4')).toEqual([])
    expect(unconnectedDataInPins('Switch', { Value: 'string' }, null, [], 'n5')).toEqual([])
  })
})
