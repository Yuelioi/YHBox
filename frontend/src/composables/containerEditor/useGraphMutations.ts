// useGraphMutations 把所有 graph 边/节点 mutation 收纳进来 (单一写入点).
//
// 原因: ContainerEditorView orchestrator 里 6 个 handler 曾各自直接写
// draft.value.graph 而不是 activeGraph.value, 导致子图层级 mutation 失效.
// review 已修复 inline, 但下次再加 handler 又会漏. 把它们抽成 composable,
// 内部 const g = activeGraph.value 是唯一写入点, 结构上消除整类 bug.
//
// onAddNode 含 Start 唯一性 / Subgraph 自动建子图等业务逻辑, 仍留在 view.
import type { Ref, ComputedRef } from 'vue'
import type { NodeChange, EdgeChange, Connection } from '@vue-flow/core'
import type { Graph, GraphNode, GraphEdge } from '@/lib/backend'
import type { FlowEdge } from './useContainerDraft'
import { edgeKind } from '@/components/containers/pinSpec'

export function useGraphMutations(opts: {
  activeGraph: ComputedRef<Graph | null>
  flowEdges: Ref<FlowEdge[]>
  syncFlowFromDraft: () => void
  findNodeAcrossGraphs: (id: string) => GraphNode | null
  deleteSubgraphCascade: (sgID: string) => Promise<void>
}) {
  const { activeGraph, flowEdges, syncFlowFromDraft, findNodeAcrossGraphs, deleteSubgraphCascade } =
    opts

  type EdgeDblClickEvent = { edge: { id: string } }

  function onNodesChange(changes: NodeChange[]) {
    const g = activeGraph.value
    if (!g) return
    for (const ch of changes) {
      if (ch.type === 'position' && ch.position) {
        const node = g.nodes.find((n) => n.id === ch.id)
        if (node) {
          node.x = ch.position.x
          node.y = ch.position.y
        }
      }
      if (ch.type === 'remove') {
        // Subgraph 节点删前 snapshot, 用于级联删子图
        const removedNode = findNodeAcrossGraphs(ch.id)
        const removedSubgraphID =
          removedNode?.kind === 'Subgraph'
            ? (removedNode.config?.subgraphId as string | undefined)
            : undefined

        g.nodes = g.nodes.filter((n) => n.id !== ch.id)
        g.edges = g.edges.filter(
          (e) => !e.from.startsWith(ch.id + '.') && !e.to.startsWith(ch.id + '.'),
        )

        if (removedSubgraphID) {
          void deleteSubgraphCascade(removedSubgraphID)
        }
      }
    }
  }

  function onEdgeDoubleClick(evt: EdgeDblClickEvent) {
    if (!evt?.edge?.id) return
    const g = activeGraph.value
    if (!g) return
    const idx = flowEdges.value.findIndex((e) => e.id === evt.edge.id)
    if (idx < 0) return
    g.edges.splice(idx, 1)
    syncFlowFromDraft()
  }

  function onEdgesChange(changes: EdgeChange[]) {
    const g = activeGraph.value
    if (!g) return
    for (const ch of changes) {
      if (ch.type === 'remove') {
        // ch.id 是 vue-flow 内部 id ('e0' 等); flowEdges 和 g.edges 一一映射
        const idx = flowEdges.value.findIndex((e) => e.id === ch.id)
        if (idx >= 0) g.edges.splice(idx, 1)
      }
    }
  }

  function onConnect(c: Connection) {
    const g = activeGraph.value
    if (!g) return
    const srcNode = g.nodes.find((n) => n.id === c.source)
    const srcPin = c.sourceHandle ?? 'out'
    const from = `${c.source}.${srcPin}`
    const to = `${c.target}.${c.targetHandle ?? 'in'}`
    // Derive edge type for dedup policy. data: single-source (replace same to);
    // exec: exec-out single target + exec-in single source (replace same from or to).
    const isData = srcNode ? edgeKind(srcNode.kind, srcPin) === 'data' : false
    if (isData) {
      g.edges = g.edges.filter((e: GraphEdge) => e.to !== to)
    } else {
      g.edges = g.edges.filter((e: GraphEdge) => e.from !== from && e.to !== to)
    }
    g.edges.push({ from, to }) // no kind field — derived at render/validate time
    syncFlowFromDraft()
  }

  return { onNodesChange, onEdgeDoubleClick, onEdgesChange, onConnect }
}
