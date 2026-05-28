// useNodeClipboard 节点剪贴板（Ctrl+C/V）+ Subgraph 1:1 复制独立子图副本。
// clipboard 在 activeGraph 层级生效（主图 / 子图层级都能 copy/paste）。
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Ref, ComputedRef } from 'vue'
import type { GraphNode as VueFlowNode } from '@vue-flow/core'
import { backend, type Container, type Graph, type GraphNode, type Subgraph } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import type { FlowNode } from './useContainerDraft'

// deepCloneSubgraphForCopy 返回的形态: 重发了 id / outputPin id 的浅 wrap
// 跟后端 Subgraph 不一致 (只保留 graph + outputPins), 局部窄声明
export interface ClonedSubgraphForPaste {
  graph: Graph
  outputPins: Array<{ id: string; name: string }>
}

export function useNodeClipboard(opts: {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  flowNodes: Ref<FlowNode[]>
  syncFlowFromDraft: () => void
  refreshSubgraphStore: () => Promise<void>
  deepCloneSubgraphForCopy: (src: Subgraph) => ClonedSubgraphForPaste
  getSelectedNodes: Ref<VueFlowNode[]>
  genID: () => string
  toast: { add: (o: Record<string, unknown>) => unknown }
}) {
  const {
    draft, activeGraph, flowNodes, syncFlowFromDraft, refreshSubgraphStore,
    deepCloneSubgraphForCopy, getSelectedNodes, genID, toast,
  } = opts
  const editorStore = useContainerEditorStore()
  const { t } = useI18n()

  const clipboard = ref<{
    nodes: any[]
    edges: any[]
    subgraphsDeepCopy: Record<string, any>
  } | null>(null)

  function onCopySelection() {
    if (!draft.value || !activeGraph.value) return
    const sel = getSelectedNodes.value ?? []
    if (sel.length === 0) return
    const ids = new Set(sel.map((n: any) => n.id))
    const nodes = (activeGraph.value.nodes as any[])
      .filter((n: any) => ids.has(n.id) && n.kind !== 'Start')
      .map((n: any) => JSON.parse(JSON.stringify(n)))
    if (nodes.length === 0) return
    const copiedIDs = new Set(nodes.map((n: any) => n.id))
    const edges = (activeGraph.value.edges as any[])
      .filter((e: any) => {
        const fromID = e.from.split('.')[0]
        const toID = e.to.split('.')[0]
        return copiedIDs.has(fromID) && copiedIDs.has(toID)
      })
      .map((e: any) => ({ ...e }))
    const subgraphsDeepCopy: Record<string, any> = {}
    for (const n of nodes) {
      if (n.kind === 'Subgraph' && n.config?.SubgraphID) {
        const sgID = String(n.config.SubgraphID)
        const sg = editorStore.subgraphsForCurrentContainer.find((s) => s.id === sgID)
        if (sg) subgraphsDeepCopy[sgID] = JSON.parse(JSON.stringify(sg))
      }
    }
    clipboard.value = { nodes, edges, subgraphsDeepCopy }
    toast.add({ title: t('editorAux.copied_nodes', { n: nodes.length }), color: 'success', duration: 1500 })
  }

  async function onPasteSelection() {
    if (!draft.value || !activeGraph.value || !clipboard.value) return
    const cb = clipboard.value
    const idMap: Record<string, string> = {}
    const cloned: any[] = []

    for (const n of cb.nodes) {
      const newID = genID()
      idMap[n.id] = newID
      const newCfg = JSON.parse(JSON.stringify(n.config ?? {}))
      if (n.kind === 'Subgraph' && newCfg.SubgraphID) {
        const sourceSg = cb.subgraphsDeepCopy[String(newCfg.SubgraphID)]
        if (sourceSg) {
          try {
            const created = (await backend.containers.createSubgraph(
              draft.value.id,
              (sourceSg.label ?? t('nodeClipboard.subgraph_word')) + t('editorAux.copy_suffix'),
            )) as any
            const newSgID = created.id
            const cloneInner = deepCloneSubgraphForCopy(sourceSg)
            await backend.containers.updateSubgraph(
              draft.value.id,
              newSgID,
              JSON.stringify({
                graph: cloneInner.graph,
                outputPins: cloneInner.outputPins,
                tags: sourceSg.tags,
                recordingContext: sourceSg.recordingContext,
              }),
            )
            newCfg.SubgraphID = newSgID
          } catch (e) {
            console.warn('paste: Subgraph clone failed', e)
          }
        }
      }
      cloned.push({
        ...JSON.parse(JSON.stringify(n)),
        id: newID,
        x: n.x + 40,
        y: n.y + 40,
        config: newCfg,
      })
    }

    const clonedEdges = cb.edges.map((e: any) => {
      const [fromID, fromPin] = e.from.split('.')
      const [toID, toPin] = e.to.split('.')
      return {
        from: `${idMap[fromID] ?? fromID}.${fromPin}`,
        to: `${idMap[toID] ?? toID}.${toPin}`,
      }
    })
    activeGraph.value.nodes.push(...cloned)
    activeGraph.value.edges.push(...clonedEdges)

    if (cb.nodes.some((n: any) => n.kind === 'Subgraph')) {
      await refreshSubgraphStore()
    }

    syncFlowFromDraft()
    setTimeout(() => {
      flowNodes.value = flowNodes.value.map((n: any) =>
        cloned.some((c) => c.id === n.id) ? { ...n, selected: true } : { ...n, selected: false },
      )
    }, 0)
  }

  function onEditorKeydown(e: KeyboardEvent) {
    const t = e.target as HTMLElement | null
    if (t && /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName)) return
    if (t?.isContentEditable) return
    if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey) {
      if (e.key === 'c' || e.key === 'C') {
        onCopySelection()
        e.preventDefault()
      } else if (e.key === 'v' || e.key === 'V') {
        void onPasteSelection()
        e.preventDefault()
      }
    }
  }

  onMounted(() => window.addEventListener('keydown', onEditorKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onEditorKeydown))

  return { clipboard, onCopySelection, onPasteSelection }
}
