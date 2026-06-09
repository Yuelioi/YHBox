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
import { edgeKind, isSentinelPin } from '@/components/containers/pinSpec'
import { useContainerEditorStore } from '@/stores/containerEditor'

export function useGraphMutations(opts: {
  activeGraph: ComputedRef<Graph | null>
  flowEdges: Ref<FlowEdge[]>
  syncFlowFromDraft: () => void
  findNodeAcrossGraphs: (id: string) => GraphNode | null
  deleteSubgraphCascade: (sgID: string) => Promise<void>
}) {
  const { activeGraph, flowEdges, syncFlowFromDraft, findNodeAcrossGraphs, deleteSubgraphCascade } =
    opts
  const editorStore = useContainerEditorStore()

  type EdgeDblClickEvent = { edge: { id: string } }

  // B2: virtual entry/output marker — 找当前 subgraph 里这个 id 对应的 slot.
  // 返回 'entry' / 'output' / null. null = 不是 marker (普通节点走老路径).
  function virtualMarkerSlot(id: string): { kind: 'entry' | 'output'; sgID: string; declID?: string } | null {
    if (editorStore.editorPath.length === 0) return null
    const sgID = editorStore.editorPath[editorStore.editorPath.length - 1]
    const sg = editorStore.subgraphsForCurrentContainer.find((s) => s.id === sgID)
    if (!sg) return null
    if (sg.entry?.nodeID === id) return { kind: 'entry', sgID }
    const p = (sg.outputPins ?? []).find((p) => p.nodeID === id)
    if (p) return { kind: 'output', sgID, declID: p.id }
    return null
  }

  function onNodesChange(changes: NodeChange[]) {
    const g = activeGraph.value
    if (!g) return
    for (const ch of changes) {
      if (ch.type === 'position' && ch.position) {
        // B2: virtual marker → 写 sg.entry/outputPins metadata, 不写 g.nodes.
        const slot = virtualMarkerSlot(ch.id)
        if (slot) {
          const sg = editorStore.subgraphsForCurrentContainer.find((s) => s.id === slot.sgID)
          if (sg) {
            if (slot.kind === 'entry') {
              sg.entry = { ...sg.entry, x: ch.position.x, y: ch.position.y }
            } else if (slot.declID) {
              const p = (sg.outputPins ?? []).find((p) => p.id === slot.declID)
              if (p) {
                p.x = ch.position.x
                p.y = ch.position.y
              }
            }
          }
          continue
        }
        const node = g.nodes.find((n) => n.id === ch.id)
        if (node) {
          node.x = ch.position.x
          node.y = ch.position.y
        }
      }
      if (ch.type === 'remove') {
        // B2: virtual marker 不能删
        if (virtualMarkerSlot(ch.id)) continue
        // Subgraph 节点删前 snapshot, 用于级联删子图
        const removedNode = findNodeAcrossGraphs(ch.id)
        const removedSubgraphID =
          removedNode?.kind === 'Subgraph'
            ? (removedNode.config?.SubgraphID as string | undefined)
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
    const srcPin = c.sourceHandle ?? 'out'
    const tgtPin = c.targetHandle ?? 'in'
    // 防火墙: 哨兵 pin (子图未解析的渲染兜底 __missing__ / 无出口 __empty__) 绝不连成持久化边。
    // 否则 FE 子图 store 一旦滞后, 连出的边带 __missing__ → 主图保存被后端拒 → 反复出现的"子图未找到"存盘失败。
    if (isSentinelPin(srcPin) || isSentinelPin(tgtPin)) return
    const srcNode = g.nodes.find((n) => n.id === c.source)
    const from = `${c.source}.${srcPin}`
    const to = `${c.target}.${tgtPin}`
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
