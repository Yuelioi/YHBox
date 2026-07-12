import { ref, type ComputedRef } from 'vue'
import { useVueFlow } from '@vue-flow/core'
import type { Container, Graph, GraphNode } from '@/lib/backend'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { loadElk, baseLayoutOptions, PRIORITY_KEY, type ElkNode } from './elkConfig'
import {
  buildElkGraph,
  anchorOffset,
  placeDetached,
  subgraphMarkerNodes,
  writeMarkerPositions,
  type Pos,
  type BBox,
} from './elkGraph'

export function useElkLayout(opts: {
  activeGraph: ComputedRef<Graph | null>
  applyDraftMutation: (m: (d: Container) => void) => void
  toast: { add: (o: Record<string, unknown>) => unknown }
  t: (k: string, p?: Record<string, unknown>) => string
}) {
  const { activeGraph, applyDraftMutation, toast, t } = opts
  const { findNode, fitView } = useVueFlow()
  const editorStore = useContainerEditorStore()
  const isLayouting = ref(false)

  // 当前在子图层级 → 返回该子图 + 入口/出口 virtual marker 的 pseudo 节点 (合进布局);
  // 主图层级返回 null。marker 不在 activeGraph.nodes 里, 必须单独取 (否则布局漏排 + 丢边)。
  function currentMarkers(): { sgID: string; nodes: GraphNode[] } | null {
    const path = editorStore.editorPath
    if (path.length === 0) return null
    const sgID = path[path.length - 1]
    const sg = editorStore.subgraphById(sgID) as { entry?: any; outputPins?: any[] } | undefined
    if (!sg) return null
    return { sgID, nodes: subgraphMarkerNodes(sg.entry, sg.outputPins) }
  }

  async function autoLayout(direction: 'LR' | 'TB' = 'LR', o: { fitView?: boolean } = {}) {
    if (isLayouting.value) return
    const g = activeGraph.value
    if (!g || (g.nodes ?? []).length === 0) return
    const dir = direction === 'LR' ? 'RIGHT' : 'DOWN'
    isLayouting.value = true
    try {
      const elk = await loadElk()
      // 子图层级: 把入口/出口 virtual marker 合进布局节点集 (marker 不在 g.nodes 里,
      // 但 g.edges 引用它们 — 不合进来 marker 不被重排、连 marker 的边还会被过滤丢掉)。
      const markerCtx = currentMarkers()
      const markerNodes = markerCtx?.nodes ?? []
      const markerIDs = new Set(markerNodes.map((m) => m.id))
      const allNodes = markerNodes.length ? [...g.nodes, ...markerNodes] : g.nodes

      // 旧位置（锚定用）
      const oldP: Record<string, Pos> = {}
      for (const n of allNodes) oldP[n.id] = { x: n.x, y: n.y }

      const elkGraph = buildElkGraph(allNodes, g.edges, {
        getSpec: (kind) => getSpec(kind) as any,
        getDims: (id) => {
          const d = findNode(id)?.dimensions
          return d && d.width ? { width: d.width, height: d.height } : null
        },
        direction: dir,
      })
      elkGraph.layoutOptions = baseLayoutOptions(dir)
      // 把占位 __priority 换成真实 ELK key
      for (const e of elkGraph.edges ?? []) {
        const p = (e.layoutOptions as Record<string, string> | undefined)?.__priority
        e.layoutOptions = p ? { [PRIORITY_KEY]: p } : {}
      }

      const res = await elk.layout(elkGraph)

      // 连通节点新坐标（ELK 左上角）
      const newP: Record<string, Pos> = {}
      for (const c of res.children ?? []) newP[c.id!] = { x: c.x ?? 0, y: c.y ?? 0 }

      // 锚定：整体平移使连通节点重心不变
      const oldConnected: Record<string, Pos> = {}
      for (const k of Object.keys(newP)) if (oldP[k]) oldConnected[k] = oldP[k]
      const { dx, dy } = anchorOffset(oldConnected, newP)
      for (const k of Object.keys(newP)) {
        newP[k].x += dx
        newP[k].y += dy
      }

      // 游离节点安置（基于锚定后的连通簇总包围盒）
      const bbox = bboxOf(res.children ?? [], dx, dy)
      const detached = allNodes
        .filter((n) => !(n.id in newP))
        .map((n) => {
          const d = findNode(n.id)?.dimensions
          return { id: n.id, x: n.x, y: n.y, width: d?.width ?? 220, height: d?.height ?? 90 }
        })
      const detP = placeDetached(detached, bbox, dir)

      // 原子写回（连通 + 游离）：直接 mutate activeGraph.value（codebase 既有模式，
      // applyDraftMutation 自动 dirty + sync + 历史快照 → 一次撤销复原）。
      applyDraftMutation(() => {
        const graph = activeGraph.value
        if (!graph) return
        for (const n of graph.nodes) {
          if (markerIDs.has(n.id)) continue // marker 不在 graph.nodes, 这里只动真实 body 节点
          if (newP[n.id]) {
            n.x = newP[n.id].x
            n.y = newP[n.id].y
          } else if (detP[n.id]) {
            n.x = detP[n.id].x
            n.y = detP[n.id].y
          }
        }
        // marker 新坐标写回 sg.entry / outputPins (不在 graph.nodes) + 标脏归属本容器。
        // 跟 onNodesChange 的 marker 拖动写回同路径; syncFlowFromDraft (applyDraftMutation 内) 重渲染。
        if (markerCtx && markerIDs.size) {
          const sg = editorStore.subgraphById(markerCtx.sgID) as
            | { entry?: any; outputPins?: any[] }
            | undefined
          if (sg) {
            const markerPos: Record<string, Pos> = {}
            for (const id of markerIDs) {
              const p = newP[id] ?? detP[id]
              if (p) markerPos[id] = p
            }
            if (writeMarkerPositions(sg, markerPos)) {
              editorStore.touchSubgraph(editorStore.activeContainerID, markerCtx.sgID)
            }
          }
        }
      })
      if (o.fitView) fitView()
    } catch (e) {
      toast.add({
        title: t('graphLayout.layout_failed'),
        description: String((e as Error)?.message ?? e),
        color: 'error',
      })
    } finally {
      isLayouting.value = false
    }
  }

  return { autoLayout, isLayouting }
}

// 连通簇布局后的总包围盒（dx/dy 是锚定平移量）。
function bboxOf(children: ElkNode[], dx: number, dy: number): BBox {
  let minX = Infinity,
    minY = Infinity,
    maxX = -Infinity,
    maxY = -Infinity
  for (const c of children) {
    const x = (c.x ?? 0) + dx,
      y = (c.y ?? 0) + dy
    minX = Math.min(minX, x)
    minY = Math.min(minY, y)
    maxX = Math.max(maxX, x + (c.width ?? 0))
    maxY = Math.max(maxY, y + (c.height ?? 0))
  }
  if (minX === Infinity) return { minX: 0, minY: 0, maxX: 0, maxY: 0 }
  return { minX, minY, maxX, maxY }
}
