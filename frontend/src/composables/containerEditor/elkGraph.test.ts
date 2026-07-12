import { describe, it, expect } from 'vitest'
import {
  estimateNodeSize,
  buildElkGraph,
  anchorOffset,
  placeDetached,
  subgraphMarkerNodes,
  writeMarkerPositions,
  type BuildOpts,
} from './elkGraph'
import type { GraphNode, GraphEdge } from '@/lib/backend'

const fakeGetSpec: BuildOpts['getSpec'] = (kind) =>
  ({
    If: { execIn: ['in'], execOut: ['then', 'else'], dataIn: { cond: 'bool' }, dataOut: {} },
    Switch: {
      execIn: ['in'],
      execOut: [],
      execOutFn: (c: any) => (c?.cases ?? []).map((_: any, i: number) => `case.${i}`),
      dataIn: {},
      dataOut: {},
    },
    GetVar: { execIn: [], execOut: [], dataIn: {}, dataOut: { value: 'any' } },
    Comment: { execIn: [], execOut: [], dataIn: {}, dataOut: {} },
  })[kind] as any

const dims = {
  If: { width: 200, height: 100 },
  GetVar: { width: 160, height: 50 },
  Switch: { width: 240, height: 150 },
}
const opts = (): BuildOpts => ({
  getSpec: fakeGetSpec,
  getDims: (id, kind) => (dims as any)[kind] ?? null,
  direction: 'RIGHT',
})
const node = (id: string, kind: string, config: any = {}): GraphNode => ({
  id,
  kind,
  x: 0,
  y: 0,
  config,
})

describe('estimateNodeSize', () => {
  it('未知 kind 回退默认 220x90', () => {
    expect(estimateNodeSize('Nope', {})).toEqual({ width: 220, height: 90 })
  })
  it('CommentBox 用 cfg 宽度', () => {
    expect(estimateNodeSize('CommentBox', { width: 600 }).width).toBe(600)
  })
  it('高度随 pin 数增长', () => {
    // pinCount 由 buildElkGraph 从 registry 派生传入；估高随它增长
    expect(estimateNodeSize('Switch', {}, 6).height).toBeGreaterThan(
      estimateNodeSize('Switch', {}, 2).height,
    )
  })
})

describe('buildElkGraph', () => {
  it('边连源/目标节点 (连节点中心, 不连端口 — 防楼梯)', () => {
    const g = buildElkGraph(
      [node('n1', 'If'), node('g1', 'GetVar')],
      [{ from: 'g1.value', to: 'n1.cond' }],
      opts(),
    )
    expect(g.edges![0].sources).toEqual(['g1'])
    expect(g.edges![0].targets).toEqual(['n1'])
  })
  it('节点不带 ELK 端口', () => {
    const g = buildElkGraph(
      [node('n1', 'If'), node('g', 'GetVar')],
      [{ from: 'g.value', to: 'n1.cond' }],
      opts(),
    )
    expect(g.children!.find((c) => c.id === 'n1')!.ports).toBeUndefined()
  })
  it('exec 边比 data 边 priority 高 (分类按 from 的 pin 名, 与去端口无关)', () => {
    const edges: GraphEdge[] = [
      { from: 'a.then', to: 'b.in' },
      { from: 'g1.value', to: 'b.cond' },
    ]
    const g = buildElkGraph([node('a', 'If'), node('b', 'If'), node('g1', 'GetVar')], edges, opts())
    const execEdge = g.edges!.find((e) => e.sources[0] === 'a')!
    const dataEdge = g.edges!.find((e) => e.sources[0] === 'g1')!
    expect(Number(execEdge.layoutOptions!.__priority)).toBeGreaterThan(
      Number(dataEdge.layoutOptions!.__priority),
    )
  })
  it('无边节点(游离)被排除', () => {
    const g = buildElkGraph(
      [node('c', 'Comment'), node('n1', 'If'), node('x', 'GetVar')],
      [{ from: 'x.value', to: 'n1.cond' }],
      opts(),
    )
    const ids = g.children!.map((c) => c.id)
    expect(ids).not.toContain('c')
    expect(ids).toContain('n1')
    expect(ids).toContain('x')
  })
  it('getDims 测不到时走 estimateNodeSize 兜底', () => {
    const noDims = (): BuildOpts => ({ ...opts(), getDims: () => null })
    const g = buildElkGraph(
      [node('n1', 'If'), node('s', 'Switch', { cases: ['a', 'b'] })],
      [{ from: 'n1.then', to: 's.in' }],
      noDims(),
    )
    const sNode = g.children!.find((c) => c.id === 's')!
    // Switch 's' 的 pin: execIn 'in'(1) + execOutFn cases→case.0/case.1(2) = 3
    expect(sNode.height).toBe(estimateNodeSize('Switch', {}, 3).height)
  })
})

describe('subgraphMarkerNodes (入口/出口 marker 合成进布局)', () => {
  it('entry + 多 output → pseudo GraphNode (id=nodeID, 正确 kind, 沿用 marker 自己的 x/y)', () => {
    const ns = subgraphMarkerNodes({ nodeID: 'in1', x: 10, y: 20 }, [
      { nodeID: 'o1', x: 300, y: 40 },
      { nodeID: 'o2', x: 300, y: 120 },
    ])
    expect(ns.map((n) => [n.id, n.kind, n.x, n.y])).toEqual([
      ['in1', 'SubgraphInput', 10, 20],
      ['o1', 'SubgraphOutput', 300, 40],
      ['o2', 'SubgraphOutput', 300, 120],
    ])
  })
  it('缺坐标走默认 (entry 80,160 / output 420,160)', () => {
    const ns = subgraphMarkerNodes({ nodeID: 'in1' }, [{ nodeID: 'o1' }])
    expect([ns[0].x, ns[0].y]).toEqual([80, 160])
    expect([ns[1].x, ns[1].y]).toEqual([420, 160])
  })
  it('无 nodeID 的项跳过; entry 缺失只出 output', () => {
    expect(subgraphMarkerNodes(null, [{ x: 1, y: 2 }])).toEqual([])
    expect(subgraphMarkerNodes(null, [{ nodeID: 'o1' }]).map((n) => n.id)).toEqual(['o1'])
  })
})

describe('writeMarkerPositions (布局后写回 marker metadata)', () => {
  it('按 nodeID 写回 entry + output 新坐标', () => {
    const sg = { entry: { nodeID: 'in1', x: 0, y: 0 }, outputPins: [{ nodeID: 'o1', x: 0, y: 0 }] }
    const changed = writeMarkerPositions(sg, { in1: { x: 11, y: 22 }, o1: { x: 33, y: 44 } })
    expect(changed).toBe(true)
    expect([sg.entry.x, sg.entry.y]).toEqual([11, 22])
    expect([sg.outputPins[0].x, sg.outputPins[0].y]).toEqual([33, 44])
  })
  it('posById 里没有的 marker 不动; 全无命中返 false', () => {
    const sg = { entry: { nodeID: 'in1', x: 5, y: 5 }, outputPins: [] }
    expect(writeMarkerPositions(sg, { other: { x: 9, y: 9 } })).toBe(false)
    expect([sg.entry.x, sg.entry.y]).toEqual([5, 5])
  })
})

describe('anchorOffset', () => {
  it('使新布局重心对齐旧重心', () => {
    const oldP = { a: { x: 0, y: 0 }, b: { x: 100, y: 0 } } // 旧中心 (50,0)
    const newP = { a: { x: 0, y: 0 }, b: { x: 200, y: 0 } } // 新中心 (100,0)
    expect(anchorOffset(oldP, newP)).toEqual({ dx: -50, dy: 0 })
  })
})

describe('placeDetached', () => {
  it('RIGHT 布局：重叠的游离节点停到包围盒下方、按序堆叠', () => {
    const bbox = { minX: 0, minY: 0, maxX: 200, maxY: 100 }
    const detached = [
      { id: 'c1', x: 10, y: 10, width: 80, height: 40 },
      { id: 'c2', x: 20, y: 20, width: 80, height: 40 },
    ]
    const out = placeDetached(detached, bbox, 'RIGHT')
    expect(out.c1.y).toBeGreaterThan(100)
    expect(out.c2.y).toBeGreaterThan(out.c1.y)
  })
  it('不与 bbox 重叠的游离节点保持原位', () => {
    const bbox = { minX: 0, minY: 0, maxX: 200, maxY: 100 }
    const detached = [{ id: 'c', x: 999, y: 999, width: 80, height: 40 }]
    expect(placeDetached(detached, bbox, 'RIGHT').c).toEqual({ x: 999, y: 999 })
  })
})
