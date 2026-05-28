// 画布节点搜索 (Ctrl+F). 从 ContainerEditorView 抽 (backlog C1).
// open ref 跟 useEditorHotkeys 共享 — hotkey toggle, modal pick 后关.

import { computed, nextTick, ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVueFlow } from '@vue-flow/core'
import type { Container, Graph } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { centerOnNode } from './constants'
import { walkAllGraphs } from './graphWalk'
import { KIND_LABEL_ZH } from '@/components/containers/pinSpec'
import type { NodeSearchResult } from '@/components/containers/NodeSearchModal.vue'

interface UseNodeSearchOpts {
  open: Ref<boolean>
  draft: Ref<Container | null>
  activeGraph: Ref<Graph | null>
  selectedID: Ref<string | null>
}

export function useNodeSearch(opts: UseNodeSearchOpts) {
  const { t } = useI18n()
  const { open, draft, activeGraph, selectedID } = opts
  const editorStore = useContainerEditorStore()
  const { setCenter } = useVueFlow()

  const query = ref('')

  const results = computed<NodeSearchResult[]>(() => {
    if (!draft.value) return []
    const q = query.value.toLowerCase().trim()
    if (!q) return []
    const out: NodeSearchResult[] = []
    walkAllGraphs(draft.value, (n, { location, sgID }) => {
      // KIND_LABEL_ZH[k] 值是 i18n key. 搜索 hay 含本地化 label, 跨 locale work.
      const labelKey = KIND_LABEL_ZH[n.kind]
      const localizedKind = labelKey ? t(labelKey) : ''
      const hay = `${n.id} ${n.kind} ${localizedKind} ${n.label ?? ''}`.toLowerCase()
      if (hay.includes(q)) {
        out.push({ id: n.id, kind: n.kind, label: n.label, location, sgID })
      }
    })
    return out
  })

  async function onPick(r: NodeSearchResult) {
    const inCurrent = (!r.sgID && editorStore.editorPath.length === 0)
      || (!!r.sgID && editorStore.editorPath[editorStore.editorPath.length - 1] === r.sgID)
    if (!inCurrent) {
      editorStore.editorPath = r.sgID ? [r.sgID] : []
      await nextTick()
    }
    selectedID.value = r.id
    const target = activeGraph.value?.nodes.find((n: any) => n.id === r.id)
    if (target) {
      try {
        centerOnNode(setCenter, target as { x: number; y: number })
      } catch {}
    }
    open.value = false
  }

  return { query, results, onPick }
}
