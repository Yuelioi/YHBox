import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export interface SubgraphSummary {
  id: string
  label: string
  // B2: outputPins 含 virtual marker nodeID + 位置 (Editor 渲染用)
  outputPins: { id: string; name: string; nodeID?: string; x?: number; y?: number }[]
  // B2: 子图入口 virtual marker
  entry: { nodeID: string; x?: number; y?: number }
  // v2 修复 Bug：activeGraph computed 需要完整 graph，否则双击进子图后画布空白
  graph: {
    id: string
    version: number
    nodes: any[]
    edges: any[]
  }
  description?: string
  recordingContext?: {
    mouseCounts360: number
    resolution: [number, number]
    recordedAt: string
  }
  tags?: string[]
}

// 编辑器层级 + 当前容器的子图列表（NodeInspector / ContainerFlowNode 用）。
// Task 2.7 / 2.8 在 ContainerEditorView 里 setActiveContainer + setEditorPath 来填充。
export const useContainerEditorStore = defineStore('containerEditor', () => {
  const activeContainerID = ref<string>('')
  const subgraphsForCurrentContainer = ref<SubgraphSummary[]>([])
  const editorPath = ref<string[]>([]) // 主图时为空；进 sub-A 时 = ['sub-A']

  // 侧栏 '容器' 跳回正在编辑的容器 — 离开列表/进入编辑器时 set, ContainersView mount 时 clear.
  // keep-alive cache 还在, 用户切走 (settings/help) 再点 '容器' 直接回原编辑器 + draft 不丢.
  const lastEditingContainerID = ref<string>('')
  function setLastEditing(id: string) { lastEditingContainerID.value = id }
  function clearLastEditing() { lastEditingContainerID.value = '' }

  function setActiveContainer(id: string, subgraphs: SubgraphSummary[]) {
    activeContainerID.value = id
    subgraphsForCurrentContainer.value = subgraphs
  }

  function pushPath(sgID: string) {
    editorPath.value = [...editorPath.value, sgID]
  }
  function popPath() {
    editorPath.value = editorPath.value.slice(0, -1)
  }
  function gotoPathIndex(idx: number) {
    editorPath.value = editorPath.value.slice(0, idx + 1)
  }

  // 1:1 模型 helpers：扫子图内 Subgraph 节点引用计数（主图引用由 view 层另算）
  function countSubgraphRefs(sgID: string): number {
    let count = 0
    for (const sg of subgraphsForCurrentContainer.value) {
      for (const n of (sg.graph?.nodes ?? [])) {
        if (n.kind === 'Subgraph' && (n as any).config?.SubgraphID === sgID) count++
      }
    }
    return count
  }

  // clipboard 暂存（Task A.4 / B.8 用，先放骨架）
  const clipboard = ref<{ nodes: any[]; edges: any[]; subgraphsDeepCopy: Record<string, any> } | null>(null)

  // 用户可见子图列表 = 排除 isAnonymous (这些是 CollapsedNode 后备子图, 不允许直接编辑).
  // 用于 Library 候选 / Subgraph 节点 SubgraphID 下拉. find/navigation 仍走原 subgraphsForCurrentContainer.
  const visibleSubgraphs = computed<SubgraphSummary[]>(() =>
    subgraphsForCurrentContainer.value.filter((s) => !(s as any).isAnonymous),
  )

  return {
    activeContainerID,
    subgraphsForCurrentContainer,
    visibleSubgraphs,
    editorPath,
    lastEditingContainerID,
    setLastEditing,
    clearLastEditing,
    setActiveContainer,
    pushPath,
    popPath,
    gotoPathIndex,
    countSubgraphRefs,
    clipboard,
  }
})
