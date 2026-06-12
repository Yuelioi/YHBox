import type { Ref, ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type Container, type Graph, type GraphNode, type Subgraph } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import type { ClonedSubgraphForPaste } from './useNodeClipboard'
import { genNodeID } from './ids'

/**
 * useSubgraphLifecycle Subgraph lifecycle 操作集 (2026-06-12 全局化后瘦身):
 * 子图是全局共享资产 — 删除 Subgraph 节点不再级联删子图 (匿名孤儿由后端 GC 回收,
 * 具名子图是用户资产留在库里); 旧 deleteSubgraphCascade/cascadeIfOrphan 整删。
 */
export function useSubgraphLifecycle(opts: {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  syncFlowFromDraft: () => void
  refreshSubgraphStore: () => Promise<void>
}) {
  const { draft } = opts
  const editorStore = useContainerEditorStore()
  const { t } = useI18n()

  /**
   * autoCreateSubgraphForNewNode 当用户拖入 Subgraph 节点：建一个空子图入全局池，
   * 把返回 id 写到 node.config.SubgraphID。失败返 false（caller 决定是否 push 节点）。
   */
  async function autoCreateSubgraphForNewNode(node: GraphNode): Promise<boolean> {
    if (node?.kind !== 'Subgraph') return true
    if (!draft.value) return false
    try {
      const created = await backend.subgraphs.create(
        t('subgraphLifecycle.default_name_prefix') + ' ' + new Date().toLocaleTimeString().slice(0, 5),
      )
      if (!created) return false
      node.config = { ...node.config, SubgraphID: created.id }
      editorStore.replaceSubgraph(created)
      return true
    } catch (e) {
      console.error('autoCreateSubgraphForNewNode failed:', e)
      return false
    }
  }

  /**
   * countSubgraphReferencesIncludeMain 扫本容器主图 + 全局池全部子图体，统计指向 sgID 的引用数。
   */
  function countSubgraphReferencesIncludeMain(sgID: string): number {
    if (!draft.value) return 0
    let count = 0
    for (const n of draft.value.graph.nodes) {
      if (n.kind === 'Subgraph' && n.config?.SubgraphID === sgID) count++
    }
    count += editorStore.countSubgraphRefs(sgID)
    return count
  }

  function findNodeAcrossGraphs(nodeID: string): GraphNode | null {
    if (!draft.value) return null
    const main = draft.value.graph.nodes.find((n) => n.id === nodeID)
    if (main) return main as GraphNode
    for (const sg of editorStore.subgraphList) {
      const n = sg.graph?.nodes?.find((x) => x.id === nodeID)
      if (n) return n as GraphNode
    }
    return null
  }

  /**
   * deepCloneSubgraphForCopy paste Subgraph 节点要"复制一份"语义时用。
   * 重发内部 nodeID + outputPin decl ID + graph.id。
   */
  function deepCloneSubgraphForCopy(src: Subgraph): ClonedSubgraphForPaste {
    const idMap: Record<string, string> = {}
    const nodes: GraphNode[] = (src.graph?.nodes ?? []).map((n) => {
      const newID = genNodeID()
      idMap[n.id] = newID
      return { ...(JSON.parse(JSON.stringify(n)) as GraphNode), id: newID }
    })
    const edges = (src.graph?.edges ?? []).map((e) => {
      const [fID, fPin] = e.from.split('.')
      const [tID, tPin] = e.to.split('.')
      return {
        from: `${idMap[fID] ?? fID}.${fPin}`,
        to: `${idMap[tID] ?? tID}.${tPin}`,
      }
    })
    const outputPins = (src.outputPins ?? []).map((p) => ({
      id: genNodeID(),
      name: p.name,
    }))
    return { graph: { id: genNodeID(), version: 1, nodes, edges }, outputPins }
  }

  return {
    autoCreateSubgraphForNewNode,
    countSubgraphReferencesIncludeMain,
    findNodeAcrossGraphs,
    deepCloneSubgraphForCopy,
  }
}
