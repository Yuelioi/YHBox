// useGraphLayout 自动布局 (ELK) + 选中节点对齐.
// 操作目标始终是 activeGraph (主图 / 当前子图层级), 与其他 composable 一致.
import type { Ref, ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphNode as VueFlowNode } from '@vue-flow/core'
import type { Container, Graph } from '@/lib/backend'
import { useElkLayout } from './useElkLayout'

export type AlignMode =
  | 'left'
  | 'right'
  | 'top'
  | 'bottom'
  | 'center-h'
  | 'center-v'
  | 'h-equal'
  | 'v-equal'

// sizeLookup 从 VueFlow 量好的 dimensions 建 id→尺寸 查表 (兜底 fallback, 含 0/缺失).
function sizeLookup(
  nodes: VueFlowNode[],
  dim: 'width' | 'height',
  fallback: number,
): (id: string) => number {
  const byID = new Map<string, number>()
  for (const n of nodes) byID.set(n.id, n.dimensions?.[dim] || fallback)
  return (id) => byID.get(id) ?? fallback
}

export function useGraphLayout(opts: {
  activeGraph: ComputedRef<Graph | null>
  getSelectedNodes: Ref<VueFlowNode[]>
  syncFlowFromDraft: () => void
  dirty: Ref<boolean>
  toast: { add: (o: Record<string, unknown>) => unknown }
  applyDraftMutation: (m: (d: Container) => void) => void
}) {
  const { activeGraph, getSelectedNodes, syncFlowFromDraft, dirty, toast, applyDraftMutation } =
    opts
  const { t } = useI18n()

  const { autoLayout } = useElkLayout({ activeGraph, applyDraftMutation, toast, t })

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
        // 按"宽度"留等宽空隙 (edge-to-edge), 而非按原点等距 —— 否则窄跨度里塞宽节点会重叠.
        // 首尾节点保持不动, 中间节点摊成相邻边距相等. 尺寸取自 VueFlow 量好的 dimensions (兜底 220).
        const w = sizeLookup(sel, 'width', 220)
        const sorted = [...targets].sort((a, b) => a.x - b.x)
        const first = sorted[0],
          last = sorted[sorted.length - 1]
        const span = last.x + w(last.id) - first.x
        const totalW = sorted.reduce((s, n) => s + w(n.id), 0)
        const gap = (span - totalW) / (sorted.length - 1)
        let cursor = first.x
        for (const n of sorted) {
          n.x = cursor
          cursor += w(n.id) + gap
        }
        break
      }
      case 'v-equal': {
        const h = sizeLookup(sel, 'height', 90)
        const sorted = [...targets].sort((a, b) => a.y - b.y)
        const first = sorted[0],
          last = sorted[sorted.length - 1]
        const span = last.y + h(last.id) - first.y
        const totalH = sorted.reduce((s, n) => s + h(n.id), 0)
        const gap = (span - totalH) / (sorted.length - 1)
        let cursor = first.y
        for (const n of sorted) {
          n.y = cursor
          cursor += h(n.id) + gap
        }
        break
      }
    }
    dirty.value = true
    syncFlowFromDraft()
  }

  return { autoLayout, alignSelected }
}
