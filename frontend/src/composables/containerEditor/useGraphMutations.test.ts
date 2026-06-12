// useGraphMutations.onConnect 哨兵 pin 防火墙单测。
// 根因: 子图未解析时节点出口 pin 渲染成 __missing__ 哨兵; 若连成边并存盘, 后端校验拒
// "不存在 out pin __missing__" → 主图保存失败 (反复出现的子图 bug)。onConnect 必须拦掉哨兵。
import { describe, it, expect } from 'vitest'
import { computed, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useGraphMutations } from './useGraphMutations'
import type { Graph, GraphNode } from '@/lib/backend'
import type { FlowEdge } from './useContainerDraft'

function node(id: string, kind: string): GraphNode {
  return { id, kind, x: 0, y: 0, config: {} } as GraphNode
}

function setup(nodes: GraphNode[]) {
  setActivePinia(createPinia())
  const graph: Graph = { id: 'g', version: 1, nodes, edges: [] } as Graph
  const activeGraph = computed<Graph | null>(() => graph)
  const flowEdges = ref<FlowEdge[]>([])
  const m = useGraphMutations({
    activeGraph,
    flowEdges,
    syncFlowFromDraft: () => {},
    findNodeAcrossGraphs: () => null,
  })
  return { m, graph }
}

describe('useGraphMutations.onConnect 哨兵防火墙', () => {
  it('正常 pin → 连成边', () => {
    const { m, graph } = setup([node('a', 'Subgraph'), node('b', 'Log')])
    m.onConnect({ source: 'a', sourceHandle: 'Done', target: 'b', targetHandle: 'In' } as any)
    expect(graph.edges).toEqual([{ from: 'a.Done', to: 'b.In' }])
  })

  it('source 是 __missing__ 哨兵 → 不连边 (杜绝存盘失败)', () => {
    const { m, graph } = setup([node('a', 'Subgraph'), node('b', 'Log')])
    m.onConnect({ source: 'a', sourceHandle: '__missing__', target: 'b', targetHandle: 'In' } as any)
    expect(graph.edges).toEqual([])
  })

  it('target 是 __empty__ 哨兵 → 不连边', () => {
    const { m, graph } = setup([node('a', 'Log'), node('b', 'Subgraph')])
    m.onConnect({ source: 'a', sourceHandle: 'Done', target: 'b', targetHandle: '__empty__' } as any)
    expect(graph.edges).toEqual([])
  })
})
