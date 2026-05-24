import type { Ref, ComputedRef } from 'vue'
import { backend, type Container, type Graph, type GraphNode, type Subgraph } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import type { ClonedSubgraphForPaste } from './useNodeClipboard'

// 内部 ID 生成器（与 ContainerEditorView 保持同样格式，避免单独抽 idgen.ts）
function genID(): string {
  return 'n_' + Math.random().toString(36).slice(2, 10)
}

/**
 * useSubgraphLifecycle Subgraph 1:1 模型 lifecycle 操作集。
 * 依赖 draft + activeGraph 来自 useContainerDraft。
 */
export function useSubgraphLifecycle(opts: {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  syncFlowFromDraft: () => void
  refreshSubgraphStore: () => Promise<void>
}) {
  const { draft, refreshSubgraphStore } = opts
  const editorStore = useContainerEditorStore()

  /**
   * autoCreateSubgraphForNewNode 当用户拖入 Subgraph 节点：调 backend.createSubgraph 建一个空子图，
   * 把返回 id 写到 node.config.SubgraphID。失败返 false（caller 决定是否 push 节点）。
   */
  async function autoCreateSubgraphForNewNode(node: GraphNode): Promise<boolean> {
    if (node?.kind !== 'Subgraph') return true
    if (!draft.value) return false
    try {
      const created = (await backend.containers.createSubgraph(
        draft.value.id,
        '子图 ' + new Date().toLocaleTimeString().slice(0, 5),
      )) as Subgraph
      node.config = { ...(node.config ?? {}), SubgraphID: created.id }
      await refreshSubgraphStore()
      return true
    } catch (e) {
      console.error('autoCreateSubgraphForNewNode failed:', e)
      return false
    }
  }

  /**
   * countSubgraphReferencesIncludeMain 扫主图 + 所有子图，统计指向 sgID 的 Subgraph 节点数。
   */
  function countSubgraphReferencesIncludeMain(sgID: string): number {
    if (!draft.value) return 0
    let count = 0
    for (const n of draft.value.graph.nodes) {
      if (n.kind === 'Subgraph' && n.config?.SubgraphID === sgID) count++
    }
    for (const sg of editorStore.subgraphsForCurrentContainer) {
      for (const n of sg.graph?.nodes ?? []) {
        if (n.kind === 'Subgraph' && n.config?.SubgraphID === sgID) count++
      }
    }
    return count
  }

  function findNodeAcrossGraphs(nodeID: string): GraphNode | null {
    if (!draft.value) return null
    const main = draft.value.graph.nodes.find((n) => n.id === nodeID)
    if (main) return main as GraphNode
    for (const sg of editorStore.subgraphsForCurrentContainer) {
      const n = sg.graph?.nodes?.find((n) => n.id === nodeID)
      if (n) return n as GraphNode
    }
    return null
  }

  async function cascadeIfOrphan(sgID: string | undefined, visited: Set<string>) {
    if (!sgID || visited.has(sgID) || !draft.value) return
    visited.add(sgID)
    if (countSubgraphReferencesIncludeMain(sgID) > 0) return
    // snapshot nested children BEFORE await — refreshSubgraphStore 等 await 之间
    // store 会被替换，原 sg 对象引用失效，遍历时拿到 stale 数据会漏删
    const sg = editorStore.subgraphsForCurrentContainer.find((s) => s.id === sgID)
    const nestedIDs: string[] = (sg?.graph?.nodes ?? [])
      .filter((n) => n.kind === 'Subgraph')
      .map((n) => n.config?.SubgraphID as string | undefined)
      .filter((id): id is string => !!id)
    try {
      await backend.containers.deleteSubgraph(draft.value.id, sgID)
    } catch (e) {
      console.warn('cascade deleteSubgraph failed', sgID, e)
    }
    for (const nID of nestedIDs) {
      await cascadeIfOrphan(nID, visited)
    }
  }

  /**
   * deleteSubgraphCascade 给 onNodesChange remove 用：已知被删 Subgraph 节点的 subgraphID，
   * 检查引用计数，0 时联动删子图 + 递归处理嵌套孙子图。
   */
  async function deleteSubgraphCascade(removedSubgraphID: string) {
    if (!draft.value || !removedSubgraphID) return
    if (countSubgraphReferencesIncludeMain(removedSubgraphID) > 0) return
    const visited = new Set<string>()
    await cascadeIfOrphan(removedSubgraphID, visited)
    await refreshSubgraphStore()
  }

  /**
   * deepCloneSubgraphForCopy paste Subgraph 节点时给副本子图用。
   * 重发内部 nodeID + outputPin decl ID + graph.id。
   */
  function deepCloneSubgraphForCopy(src: Subgraph): ClonedSubgraphForPaste {
    const idMap: Record<string, string> = {}
    const nodes: GraphNode[] = (src.graph?.nodes ?? []).map((n) => {
      const newID = genID()
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
      id: genID(),
      name: p.name,
    }))
    return { graph: { id: genID(), version: 1, nodes, edges }, outputPins }
  }

  /**
   * gcOrphanSubgraphs 扫无引用的子图，返 sgID 列表（caller 决定何时实际 backend.deleteSubgraph）。
   */
  function gcOrphanSubgraphs(): string[] {
    if (!draft.value) return []
    const referenced = new Set<string>()
    for (const n of draft.value.graph.nodes) {
      if (n.kind === 'Subgraph' && n.config?.SubgraphID) {
        referenced.add(String(n.config.SubgraphID))
      }
    }
    for (const sg of editorStore.subgraphsForCurrentContainer) {
      for (const n of sg.graph?.nodes ?? []) {
        if (n.kind === 'Subgraph' && n.config?.SubgraphID) {
          referenced.add(String(n.config.SubgraphID))
        }
      }
    }
    return editorStore.subgraphsForCurrentContainer
      .filter((s) => !referenced.has(s.id))
      .map((s) => s.id)
  }

  return {
    autoCreateSubgraphForNewNode,
    countSubgraphReferencesIncludeMain,
    findNodeAcrossGraphs,
    deleteSubgraphCascade,
    deepCloneSubgraphForCopy,
    gcOrphanSubgraphs,
  }
}
