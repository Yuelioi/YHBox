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

type MarkerOwner = {
  entry?: { nodeID?: string; x?: number; y?: number }
  outputPins?: Array<{ nodeID?: string; x?: number; y?: number }>
}

type LayoutContext = {
  graph: Graph
  containerID: string
  editorPath: string[]
  subgraph: MarkerOwner | undefined
  markerCtx: { sgID: string; owner: MarkerOwner; nodes: GraphNode[] } | null
}

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

  // 布局输入必须绑定到发起时的 graph + editor 层级。ELK 引擎和 layout 都是异步的，期间
  // keep-alive 编辑器可能切容器/子图；只比较 graph 不足以保护复用同一 graph 的 path 切换。
  function captureLayoutContext(graph: Graph): LayoutContext {
    const editorPath = [...editorStore.editorPath]
    const sgID = editorPath.at(-1)
    const subgraph = sgID ? (editorStore.subgraphById(sgID) as MarkerOwner | undefined) : undefined
    return {
      graph,
      containerID: editorStore.activeContainerID,
      editorPath,
      subgraph,
      markerCtx:
        sgID && subgraph
          ? {
              sgID,
              owner: subgraph,
              nodes: subgraphMarkerNodes(subgraph.entry, subgraph.outputPins),
            }
          : null,
    }
  }

  function isLayoutContextCurrent(context: LayoutContext): boolean {
    if (
      activeGraph.value !== context.graph ||
      editorStore.activeContainerID !== context.containerID
    ) {
      return false
    }
    const path = editorStore.editorPath
    if (
      path.length !== context.editorPath.length ||
      path.some((segment, index) => segment !== context.editorPath[index])
    ) {
      return false
    }
    const sgID = path.at(-1)
    const subgraph = sgID ? (editorStore.subgraphById(sgID) as MarkerOwner | undefined) : undefined
    return subgraph === context.subgraph
  }

  async function autoLayout(direction: 'LR' | 'TB' = 'LR', o: { fitView?: boolean } = {}) {
    if (isLayouting.value) return
    const g = activeGraph.value
    if (!g || (g.nodes ?? []).length === 0) return
    const context = captureLayoutContext(g)
    const dir = direction === 'LR' ? 'RIGHT' : 'DOWN'
    isLayouting.value = true
    try {
      const elk = await loadElk()
      if (!isLayoutContextCurrent(context)) return
      // 子图层级: 把入口/出口 virtual marker 合进布局节点集 (marker 不在 g.nodes 里,
      // 但 g.edges 引用它们 — 不合进来 marker 不被重排、连 marker 的边还会被过滤丢掉)。
      const markerCtx = context.markerCtx
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
      if (!isLayoutContextCurrent(context)) return

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
      let applied = false
      applyDraftMutation(() => {
        // applyDraftMutation 当前同步执行；闭包内仍复核一次，避免未来 wrapper 行为变化后
        // 把旧 layout 写入另一 editor。写回只用捕获对象，不重新解析 activeGraph/marker。
        if (!isLayoutContextCurrent(context)) return
        applied = true
        for (const n of context.graph.nodes) {
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
          const markerPos: Record<string, Pos> = {}
          for (const id of markerIDs) {
            const p = newP[id] ?? detP[id]
            if (p) markerPos[id] = p
          }
          if (writeMarkerPositions(markerCtx.owner, markerPos)) {
            editorStore.touchSubgraph(context.containerID, markerCtx.sgID)
          }
        }
      })
      if (applied && o.fitView) fitView()
    } catch (e) {
      if (!isLayoutContextCurrent(context)) return
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
