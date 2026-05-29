import { describe, it, expect } from 'vitest'
import { unconnectedDataInPins } from './pinLiterals'
import type { PinType } from '@/components/containers/nodeRegistry/index'

const KEYPRESS_DATAIN: Record<string, PinType> = { VK: 'string', Dur: 'number' }

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

  it('Expr 节点并入 config.inputs[] 未连线动态输入', () => {
    const config = { inputs: [{ name: 'a', type: 'number' }, { name: 'b', type: 'number' }] }
    const edges = [{ from: 's.out', to: 'n1.a' }]
    const out = unconnectedDataInPins('Expr', {}, config, edges, 'n1')
    expect(out.map((p) => p.name)).toEqual(['b'])
  })
})
