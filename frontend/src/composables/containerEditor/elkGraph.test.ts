import { describe, it, expect } from 'vitest'
import { estimateNodeSize, buildElkGraph, type BuildOpts } from './elkGraph'
import type { GraphNode, GraphEdge } from '@/lib/backend'

const fakeGetSpec: BuildOpts['getSpec'] = (kind) => ({
  If:     { execIn: ['in'], execOut: ['then', 'else'], dataIn: { cond: 'bool' }, dataOut: {} },
  Switch: { execIn: ['in'], execOut: [], execOutFn: (c: any) => (c?.cases ?? []).map((_: any, i: number) => `case.${i}`), dataIn: {}, dataOut: {} },
  GetVar: { execIn: [], execOut: [], dataIn: {}, dataOut: { value: 'any' } },
  Comment:{ execIn: [], execOut: [], dataIn: {}, dataOut: {} },
}[kind] as any)

const dims = { If: { width: 200, height: 100 }, GetVar: { width: 160, height: 50 }, Switch: { width: 240, height: 150 } }
const opts = (): BuildOpts => ({
  getSpec: fakeGetSpec,
  getDims: (id, kind) => (dims as any)[kind] ?? null,
  direction: 'RIGHT',
})
const node = (id: string, kind: string, config: any = {}): GraphNode => ({ id, kind, x: 0, y: 0, config })

describe('estimateNodeSize', () => {
  it('未知 kind 回退默认 220x90', () => {
    expect(estimateNodeSize('Nope', {})).toEqual({ width: 220, height: 90 })
  })
  it('CommentBox 用 cfg 宽度', () => {
    expect(estimateNodeSize('CommentBox', { width: 600 }).width).toBe(600)
  })
  it('Switch 高度随 cases 数增长', () => {
    const few = estimateNodeSize('Switch', { cases: ['a'] }).height
    const many = estimateNodeSize('Switch', { cases: ['a', 'b', 'c', 'd'] }).height
    expect(many).toBeGreaterThan(few)
  })
})

describe('buildElkGraph', () => {
  it('输入 pin 落 WEST、输出 pin 落 EAST', () => {
    const g = buildElkGraph([node('n1', 'If'), node('g', 'GetVar')], [{ from: 'g.value', to: 'n1.cond' }], opts())
    const ports = g.children!.find((c) => c.id === 'n1')!.ports!
    expect(ports.find((p) => p.id === 'n1::cond')!.layoutOptions!['elk.port.side']).toBe('WEST')
    expect(ports.find((p) => p.id === 'n1::then')!.layoutOptions!['elk.port.side']).toBe('EAST')
  })
  it('边映射到正确 port id', () => {
    const g = buildElkGraph([node('n1', 'If'), node('g1', 'GetVar')], [{ from: 'g1.value', to: 'n1.cond' }], opts())
    expect(g.edges![0].sources).toEqual(['g1::value'])
    expect(g.edges![0].targets).toEqual(['n1::cond'])
  })
  it('exec 边比 data 边 priority 高', () => {
    const edges: GraphEdge[] = [{ from: 'a.then', to: 'b.in' }, { from: 'g1.value', to: 'b.cond' }]
    const g = buildElkGraph([node('a', 'If'), node('b', 'If'), node('g1', 'GetVar')], edges, opts())
    const execEdge = g.edges!.find((e) => e.sources[0] === 'a::then')!
    const dataEdge = g.edges!.find((e) => e.sources[0] === 'g1::value')!
    expect(Number(execEdge.layoutOptions!.__priority)).toBeGreaterThan(Number(dataEdge.layoutOptions!.__priority))
  })
  it('动态 pin(Switch.cases)派生为 port', () => {
    const g = buildElkGraph([node('s', 'Switch', { cases: ['x', 'y'] }), node('g', 'GetVar')], [{ from: 'g.value', to: 's.in' }], opts())
    const ids = g.children!.find((c) => c.id === 's')!.ports!.map((p) => p.id)
    expect(ids).toContain('s::case.0'); expect(ids).toContain('s::case.1')
  })
  it('无边节点(游离)被排除', () => {
    const g = buildElkGraph([node('c', 'Comment'), node('n1', 'If'), node('x', 'GetVar')], [{ from: 'x.value', to: 'n1.cond' }], opts())
    const ids = g.children!.map((c) => c.id)
    expect(ids).not.toContain('c'); expect(ids).toContain('n1'); expect(ids).toContain('x')
  })
  it('getDims 测不到时走 estimateNodeSize 兜底', () => {
    const noDims = (): BuildOpts => ({ ...opts(), getDims: () => null })
    const g = buildElkGraph([node('n1', 'If'), node('s', 'Switch', { cases: ['a', 'b'] })], [{ from: 'n1.then', to: 's.in' }], noDims())
    const sNode = g.children!.find((c) => c.id === 's')!
    expect(sNode.height).toBe(estimateNodeSize('Switch', { cases: ['a', 'b'] }).height)
  })
})
