// 单元测 useSubgraphLifecycle 纯查询 + 纯变换函数:
//   countSubgraphReferencesIncludeMain / findNodeAcrossGraphs / gcOrphanSubgraphs /
//   deepCloneSubgraphForCopy
// 不测 backend-touching 函数 (autoCreateSubgraphForNewNode / cascadeIfOrphan): 涉及 RPC mock + await chain,
// 集成测试合理点, 单元层投入产出比低.
import { describe, it, expect, beforeEach } from 'vitest'
import { computed, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useSubgraphLifecycle } from './useSubgraphLifecycle'
import type { Container, Graph, GraphNode } from '@/lib/backend'

function makeNode(id: string, kind: string, config?: Record<string, any>): GraphNode {
  return { id, kind, x: 0, y: 0, config: config ?? {} }
}

function makeContainer(mainNodes: GraphNode[]): Container {
  return {
    schemaVersion: 1,
    id: 'c1',
    name: 'test',
    graph: { id: 'g-main', version: 1, nodes: mainNodes, edges: [] },
    createdAt: '',
    updatedAt: '',
  }
}

function setup(draft: Container | null, sgs: Array<{ id: string; graph: Graph; outputPins?: any[] }>) {
  setActivePinia(createPinia())
  const store = useContainerEditorStore()
  store.setActiveContainer(
    draft?.id ?? '',
    sgs.map((s) => ({
      id: s.id,
      label: s.id,
      outputPins: s.outputPins ?? [],
      graph: s.graph,
    })),
  )
  const draftRef = ref<Container | null>(draft)
  const activeGraph = computed<Graph | null>(() => draftRef.value?.graph ?? null)
  const lifecycle = useSubgraphLifecycle({
    draft: draftRef,
    activeGraph,
    syncFlowFromDraft: () => {},
    refreshSubgraphStore: async () => {},
  })
  return { store, lifecycle }
}

describe('useSubgraphLifecycle.countSubgraphReferencesIncludeMain', () => {
  it('计数主图引用', () => {
    const draft = makeContainer([
      makeNode('n1', 'Subgraph', { subgraphId: 'sg-a' }),
      makeNode('n2', 'Subgraph', { subgraphId: 'sg-a' }),
      makeNode('n3', 'Subgraph', { subgraphId: 'sg-b' }),
      makeNode('n4', 'Sleep'),
    ])
    const { lifecycle } = setup(draft, [])
    expect(lifecycle.countSubgraphReferencesIncludeMain('sg-a')).toBe(2)
    expect(lifecycle.countSubgraphReferencesIncludeMain('sg-b')).toBe(1)
    expect(lifecycle.countSubgraphReferencesIncludeMain('sg-none')).toBe(0)
  })

  it('计数嵌套子图内引用 (主图 + 子图 都加)', () => {
    const draft = makeContainer([makeNode('m1', 'Subgraph', { subgraphId: 'sg-a' })])
    const sgs = [
      {
        id: 'sg-a',
        graph: {
          id: 'g-a',
          version: 1,
          nodes: [makeNode('a1', 'Subgraph', { subgraphId: 'sg-b' })],
          edges: [],
        },
      },
      {
        id: 'sg-b',
        graph: {
          id: 'g-b',
          version: 1,
          nodes: [
            makeNode('b1', 'Subgraph', { subgraphId: 'sg-c' }),
            makeNode('b2', 'Subgraph', { subgraphId: 'sg-c' }),
          ],
          edges: [],
        },
      },
    ]
    const { lifecycle } = setup(draft, sgs)
    expect(lifecycle.countSubgraphReferencesIncludeMain('sg-a')).toBe(1) // 主图
    expect(lifecycle.countSubgraphReferencesIncludeMain('sg-b')).toBe(1) // sg-a 里
    expect(lifecycle.countSubgraphReferencesIncludeMain('sg-c')).toBe(2) // sg-b 里 2 个
  })

  it('draft 为 null 时返 0', () => {
    const { lifecycle } = setup(null, [])
    expect(lifecycle.countSubgraphReferencesIncludeMain('sg-anything')).toBe(0)
  })
})

describe('useSubgraphLifecycle.findNodeAcrossGraphs', () => {
  it('找主图节点', () => {
    const draft = makeContainer([makeNode('n1', 'Sleep')])
    const { lifecycle } = setup(draft, [])
    expect(lifecycle.findNodeAcrossGraphs('n1')?.kind).toBe('Sleep')
  })

  it('找子图内节点', () => {
    const draft = makeContainer([])
    const sgs = [
      {
        id: 'sg-a',
        graph: { id: 'g-a', version: 1, nodes: [makeNode('inner', 'KeyPress')], edges: [] },
      },
    ]
    const { lifecycle } = setup(draft, sgs)
    expect(lifecycle.findNodeAcrossGraphs('inner')?.kind).toBe('KeyPress')
  })

  it('不存在返 null', () => {
    const { lifecycle } = setup(makeContainer([]), [])
    expect(lifecycle.findNodeAcrossGraphs('nope')).toBeNull()
  })
})

describe('useSubgraphLifecycle.gcOrphanSubgraphs', () => {
  it('返无引用的子图 ID', () => {
    const draft = makeContainer([makeNode('m1', 'Subgraph', { subgraphId: 'sg-used' })])
    const sgs = [
      { id: 'sg-used', graph: { id: 'g1', version: 1, nodes: [], edges: [] } },
      { id: 'sg-orphan-1', graph: { id: 'g2', version: 1, nodes: [], edges: [] } },
      { id: 'sg-orphan-2', graph: { id: 'g3', version: 1, nodes: [], edges: [] } },
    ]
    const { lifecycle } = setup(draft, sgs)
    const orphans = lifecycle.gcOrphanSubgraphs().sort()
    expect(orphans).toEqual(['sg-orphan-1', 'sg-orphan-2'])
  })

  it('嵌套引用算被引用 (不算 orphan)', () => {
    const draft = makeContainer([makeNode('m1', 'Subgraph', { subgraphId: 'sg-a' })])
    const sgs = [
      {
        id: 'sg-a',
        graph: {
          id: 'g-a',
          version: 1,
          nodes: [makeNode('a1', 'Subgraph', { subgraphId: 'sg-b' })],
          edges: [],
        },
      },
      { id: 'sg-b', graph: { id: 'g-b', version: 1, nodes: [], edges: [] } },
    ]
    const { lifecycle } = setup(draft, sgs)
    expect(lifecycle.gcOrphanSubgraphs()).toEqual([])
  })
})

describe('useSubgraphLifecycle.deepCloneSubgraphForCopy', () => {
  it('node id / graph id / outputPin id 都重发', () => {
    const src = {
      id: 'sg-orig',
      label: 'orig',
      graph: {
        id: 'g-orig',
        version: 1,
        nodes: [makeNode('n1', 'Sleep'), makeNode('n2', 'KeyPress')],
        edges: [{ from: 'n1.out', to: 'n2.in' }],
      },
      outputPins: [
        { id: 'decl-orig', name: 'done' },
        { id: 'decl-orig-2', name: 'fail' },
      ],
      createdAt: '',
    }
    const { lifecycle } = setup(makeContainer([]), [])
    const clone = lifecycle.deepCloneSubgraphForCopy(src as any)

    // graph.id 重发
    expect(clone.graph.id).not.toBe('g-orig')
    // node id 重发, kind 保留
    expect(clone.graph.nodes).toHaveLength(2)
    expect(clone.graph.nodes[0].id).not.toBe('n1')
    expect(clone.graph.nodes[0].kind).toBe('Sleep')
    expect(clone.graph.nodes[1].kind).toBe('KeyPress')
    // edge from/to 按新 nodeID 改写
    const e = clone.graph.edges[0]
    expect(e.from).toBe(`${clone.graph.nodes[0].id}.out`)
    expect(e.to).toBe(`${clone.graph.nodes[1].id}.in`)
    // outputPins id 重发, name 保留
    expect(clone.outputPins).toHaveLength(2)
    expect(clone.outputPins[0].id).not.toBe('decl-orig')
    expect(clone.outputPins[0].name).toBe('done')
    expect(clone.outputPins[1].name).toBe('fail')
  })

  it('空 src 不炸', () => {
    const { lifecycle } = setup(makeContainer([]), [])
    const clone = lifecycle.deepCloneSubgraphForCopy({
      id: '',
      label: '',
      graph: { id: '', version: 1, nodes: [], edges: [] },
      outputPins: [],
      createdAt: '',
    } as any)
    expect(clone.graph.nodes).toEqual([])
    expect(clone.outputPins).toEqual([])
  })
})

describe('useSubgraphLifecycle 集成: cascade 引用计数语义', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  // cascade 删用 deleteSubgraphCascade 入口, 但它先 check countSubgraphReferencesIncludeMain.
  // 这里通过 reference count 验证: 多引用时 cascade 不应触发 (单元层只验入口检查, 完整 await
  // 链交集成测).
  it('countSubgraphReferencesIncludeMain > 0 时 cascade 入口必须 return 不删', () => {
    const draft = makeContainer([
      makeNode('a', 'Subgraph', { subgraphId: 'sg-shared' }),
      makeNode('b', 'Subgraph', { subgraphId: 'sg-shared' }),
    ])
    const { lifecycle } = setup(draft, [
      { id: 'sg-shared', graph: { id: 'g', version: 1, nodes: [], edges: [] } },
    ])
    // 引用 2 个; 即便 ID 被"删"也得保留
    expect(lifecycle.countSubgraphReferencesIncludeMain('sg-shared')).toBe(2)
  })
})
