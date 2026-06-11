import { ref, type ComputedRef } from 'vue'
import { useVueFlow } from '@vue-flow/core'
import type { Container, Graph } from '@/lib/backend'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { createElk, baseLayoutOptions, PRIORITY_KEY, type ElkNode } from './elkConfig'
import { buildElkGraph, anchorOffset, placeDetached, type Pos, type BBox } from './elkGraph'

export function useElkLayout(opts: {
  activeGraph: ComputedRef<Graph | null>
  applyDraftMutation: (m: (d: Container) => void) => void
  toast: { add: (o: Record<string, unknown>) => unknown }
  t: (k: string, p?: Record<string, unknown>) => string
}) {
  const { activeGraph, applyDraftMutation, toast, t } = opts
  const { findNode, fitView } = useVueFlow()
  const elk = createElk()
  const isLayouting = ref(false)

  async function autoLayout(direction: 'LR' | 'TB' = 'LR', o: { fitView?: boolean } = {}) {
    if (isLayouting.value) return
    const g = activeGraph.value
    if (!g || (g.nodes ?? []).length === 0) return
    const dir = direction === 'LR' ? 'RIGHT' : 'DOWN'
    isLayouting.value = true
    try {
      // 旧位置（锚定用）
      const oldP: Record<string, Pos> = {}
      for (const n of g.nodes) oldP[n.id] = { x: n.x, y: n.y }

      const elkGraph = buildElkGraph(g.nodes, g.edges, {
        getSpec: (kind) => getSpec(kind) as any,
        getDims: (id, kind) => {
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
      for (const k of Object.keys(newP)) { newP[k].x += dx; newP[k].y += dy }

      // 游离节点安置（基于锚定后的连通簇总包围盒）
      const bbox = bboxOf(res.children ?? [], dx, dy)
      const detached = g.nodes
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
          if (newP[n.id]) { n.x = newP[n.id].x; n.y = newP[n.id].y }
          else if (detP[n.id]) { n.x = detP[n.id].x; n.y = detP[n.id].y }
        }
      })
      if (o.fitView) fitView()
    } catch (e) {
      toast.add({ title: t('graphLayout.layout_failed'), description: String((e as Error)?.message ?? e), color: 'error' })
    } finally {
      isLayouting.value = false
    }
  }

  return { autoLayout, isLayouting }
}

// 连通簇布局后的总包围盒（dx/dy 是锚定平移量）。
function bboxOf(children: ElkNode[], dx: number, dy: number): BBox {
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity
  for (const c of children) {
    const x = (c.x ?? 0) + dx, y = (c.y ?? 0) + dy
    minX = Math.min(minX, x); minY = Math.min(minY, y)
    maxX = Math.max(maxX, x + (c.width ?? 0)); maxY = Math.max(maxY, y + (c.height ?? 0))
  }
  if (minX === Infinity) return { minX: 0, minY: 0, maxX: 0, maxY: 0 }
  return { minX, minY, maxX, maxY }
}
