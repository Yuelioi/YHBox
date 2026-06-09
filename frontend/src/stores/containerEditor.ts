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
//
// 状态按容器隔离 (subgraphsByContainer / editorPathByContainer keyed by containerID)。
// 起因: <keep-alive include="ContainerEditorView"> 同时缓存多个容器编辑器, 旧实现用单一
// 全局 ref 存"当前容器"的子图/层级, 切到别的容器会把它整个覆盖 → 切回来时本容器独有的
// Subgraph 节点渲染成 "(子图未找到)", 且未落盘子图编辑/导航层级在切换时丢失。
// 现在每个容器有独立 slot, 互不干扰; activeContainerID 只是"哪个编辑器在前台"的指针,
// 由各编辑器实例在 mount / onActivated 时 setActiveContainer / markActive 指向自己。
// 对外 API (subgraphsForCurrentContainer / editorPath / pushPath ... ) 形态不变 —— 它们
// 作用在 activeContainerID 对应的 slot 上; descendant 组件 (只在 active 编辑器里渲染) 照旧读。
export const useContainerEditorStore = defineStore('containerEditor', () => {
  const activeContainerID = ref<string>('')
  const subgraphsByContainer = ref<Record<string, SubgraphSummary[]>>({})
  const editorPathByContainer = ref<Record<string, string[]>>({}) // 主图时为空；进 sub-A 时 = ['sub-A']

  // 侧栏 '容器' 跳回正在编辑的容器 — 离开列表/进入编辑器时 set, ContainersView mount 时 clear.
  // keep-alive cache 还在, 用户切走 (settings/help) 再点 '容器' 直接回原编辑器 + draft 不丢.
  const lastEditingContainerID = ref<string>('')
  function setLastEditing(id: string) { lastEditingContainerID.value = id }
  function clearLastEditing() { lastEditingContainerID.value = '' }

  // ── 按容器精确访问 (per-instance composable 用自己的 containerID 调, 不受 active 漂移影响) ──
  function subgraphsFor(cid: string): SubgraphSummary[] {
    return subgraphsByContainer.value[cid] ?? []
  }
  function editorPathFor(cid: string): string[] {
    return editorPathByContainer.value[cid] ?? []
  }

  // ── active 视图 (descendant 组件读: 它们只在前台编辑器子树里渲染) ──
  const subgraphsForCurrentContainer = computed<SubgraphSummary[]>(
    () => subgraphsByContainer.value[activeContainerID.value] ?? [],
  )
  // 只读 computed (Pinia setup store 把 computed 当 getter, 外部不能直接赋值) —— 写走 setPath / 导航 action。
  const editorPath = computed<string[]>(() => editorPathByContainer.value[activeContainerID.value] ?? [])

  // setPath 整体设前台容器层级 (替代原先的 editorStore.editorPath = [...] 直接赋值)。
  function setPath(path: string[]) {
    if (activeContainerID.value) editorPathByContainer.value[activeContainerID.value] = path
  }

  // setActiveContainer 整体替换某容器的子图列表 + 设为前台 (load / reload 用)。
  function setActiveContainer(id: string, subgraphs: SubgraphSummary[]) {
    activeContainerID.value = id
    subgraphsByContainer.value[id] = subgraphs
    if (!(id in editorPathByContainer.value)) editorPathByContainer.value[id] = []
  }

  // markActive 仅把前台指针指向某容器 (keep-alive onActivated 用) —— 数据已在 slot 里, 不动。
  function markActive(id: string) {
    if (id) activeContainerID.value = id
  }

  // mergeSubgraphs: 后端子图快照同步进该容器 slot, 但不整体覆盖 —— 两边都有的子图保留内存版,
  // 否则正在编辑、还没落盘的子图修改会被磁盘旧版盖掉、引用丢失 (录制雪崩根因③)。
  // 后端列表为基准: 不在列表里的 (后端已删) 自然移除。按 id 比对只在同一容器 slot 内, 不跨容器串。
  function mergeSubgraphs(id: string, fresh: SubgraphSummary[]) {
    activeContainerID.value = id
    const existing = new Map((subgraphsByContainer.value[id] ?? []).map((s) => [s.id, s]))
    subgraphsByContainer.value[id] = fresh.map((f) => existing.get(f.id) ?? f)
  }

  // 容器编辑器实例销毁 (keep-alive 淘汰 / 离开不缓存) 时释放该容器 slot, 防内存堆积。
  function dropContainer(id: string) {
    delete subgraphsByContainer.value[id]
    delete editorPathByContainer.value[id]
    if (activeContainerID.value === id) activeContainerID.value = ''
  }

  // 层级导航 — 作用在前台容器 slot 上 (只有 active 编辑器在响应用户操作)。
  function pushPath(sgID: string) {
    setPath([...editorPath.value, sgID])
  }
  function popPath() {
    setPath(editorPath.value.slice(0, -1))
  }
  function gotoPathIndex(idx: number) {
    setPath(editorPath.value.slice(0, idx + 1))
  }
  // 回主图根 —— 编辑器「重载」后用 (子图可能已被外部改/删, 停旧子图视图会出错)。
  function resetPath() {
    setPath([])
  }

  // 1:1 模型 helpers：扫前台容器子图内 Subgraph 节点引用计数（主图引用由 view 层另算）
  function countSubgraphRefs(sgID: string): number {
    let count = 0
    for (const sg of subgraphsForCurrentContainer.value) {
      for (const n of (sg.graph?.nodes ?? [])) {
        if (n.kind === 'Subgraph' && (n as any).config?.SubgraphID === sgID) count++
      }
    }
    return count
  }

  // clipboard 暂存（Task A.4 / B.8 用，先放骨架）— 跨容器共享是有意的 (容器间 copy/paste)。
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
    subgraphsFor,
    editorPathFor,
    lastEditingContainerID,
    setLastEditing,
    clearLastEditing,
    setActiveContainer,
    markActive,
    mergeSubgraphs,
    dropContainer,
    setPath,
    pushPath,
    popPath,
    gotoPathIndex,
    resetPath,
    countSubgraphRefs,
    clipboard,
  }
})
