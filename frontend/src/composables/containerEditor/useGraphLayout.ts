// useGraphLayout 自动布局 (dagre) + 选中节点对齐.
// 操作目标始终是 activeGraph (主图 / 当前子图层级), 与其他 composable 一致.
import type { Ref, ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphNode as VueFlowNode } from '@vue-flow/core'
import type { Graph } from '@/lib/backend'
import dagre from '@dagrejs/dagre'

const NODE_W = 220
const NODE_H = 90

export type AlignMode =
  | 'left'
  | 'right'
  | 'top'
  | 'bottom'
  | 'center-h'
  | 'center-v'
  | 'h-equal'
  | 'v-equal'

export function useGraphLayout(opts: {
  activeGraph: ComputedRef<Graph | null>
  getSelectedNodes: Ref<VueFlowNode[]>
  syncFlowFromDraft: () => void
  dirty: Ref<boolean>
  toast: { add: (o: Record<string, unknown>) => unknown }
}) {
  const { activeGraph, getSelectedNodes, syncFlowFromDraft, dirty, toast } = opts
  const { t } = useI18n()

  function autoLayout(direction: 'LR' | 'TB' = 'LR') {
    const g = activeGraph.value
    if (!g || (g.nodes ?? []).length === 0) return
    const dg = new dagre.graphlib.Graph()
    dg.setGraph({ rankdir: direction, nodesep: 60, ranksep: 120, marginx: 40, marginy: 40 })
    dg.setDefaultEdgeLabel(() => ({}))
    for (const n of g.nodes) dg.setNode(n.id, { width: NODE_W, height: NODE_H })
    for (const e of g.edges) {
      const src = e.from.split('.')[0]
      const tgt = e.to.split('.')[0]
      dg.setEdge(src, tgt)
    }
    dagre.layout(dg)
    for (const n of g.nodes) {
      const p = dg.node(n.id)
      if (p) {
        n.x = p.x - NODE_W / 2
        n.y = p.y - NODE_H / 2
      }
    }
    dirty.value = true
    syncFlowFromDraft()
    toast.add({
      title: t('graphLayout.auto_layout_done', { dir: direction === 'LR' ? t('graphLayout.horizontal') : t('graphLayout.vertical') }),
      color: 'success',
      icon: 'i-tabler-layout-grid',
    })
  }

  function alignSelected(mode: AlignMode) {
    const g = activeGraph.value
    if (!g) return
    const sel = getSelectedNodes.value
    if (sel.length < 2) {
      toast.add({ title: t('editorAux.warning_select_two'), color: 'warning' })
      return
    }
    const selIDs = new Set(sel.map((s) => s.id))
    const targets = g.nodes.filter((n) => selIDs.has(n.id))
    if (targets.length < 2) return

    switch (mode) {
      case 'left': {
        const x = Math.min(...targets.map((n) => n.x))
        for (const n of targets) n.x = x
        break
      }
      case 'right': {
        const x = Math.max(...targets.map((n) => n.x))
        for (const n of targets) n.x = x
        break
      }
      case 'top': {
        const y = Math.min(...targets.map((n) => n.y))
        for (const n of targets) n.y = y
        break
      }
      case 'bottom': {
        const y = Math.max(...targets.map((n) => n.y))
        for (const n of targets) n.y = y
        break
      }
      case 'center-h': {
        const avg = targets.reduce((s, n) => s + n.y, 0) / targets.length
        for (const n of targets) n.y = avg
        break
      }
      case 'center-v': {
        const avg = targets.reduce((s, n) => s + n.x, 0) / targets.length
        for (const n of targets) n.x = avg
        break
      }
      case 'h-equal': {
        const sorted = [...targets].sort((a, b) => a.x - b.x)
        const min = sorted[0].x
        const max = sorted[sorted.length - 1].x
        const gap = (max - min) / (sorted.length - 1)
        for (let i = 0; i < sorted.length; i++) sorted[i].x = min + i * gap
        break
      }
      case 'v-equal': {
        const sorted = [...targets].sort((a, b) => a.y - b.y)
        const min = sorted[0].y
        const max = sorted[sorted.length - 1].y
        const gap = (max - min) / (sorted.length - 1)
        for (let i = 0; i < sorted.length; i++) sorted[i].y = min + i * gap
        break
      }
    }
    dirty.value = true
    syncFlowFromDraft()
  }

  return { autoLayout, alignSelected }
}
