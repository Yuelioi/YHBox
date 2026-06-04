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

export function useGraphLayout(opts: {
  activeGraph: ComputedRef<Graph | null>
  getSelectedNodes: Ref<VueFlowNode[]>
  syncFlowFromDraft: () => void
  dirty: Ref<boolean>
  toast: { add: (o: Record<string, unknown>) => unknown }
  applyDraftMutation: (m: (d: Container) => void) => void
}) {
  const { activeGraph, getSelectedNodes, syncFlowFromDraft, dirty, toast, applyDraftMutation } = opts
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
