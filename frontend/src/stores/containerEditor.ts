import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Subgraph } from '@/lib/backend'

// 子图已全局化 (2026-06-12): 池里就是后端 Subgraph 全量, 不再有"摘要"裁剪。
// 旧名保留作别名, 消费方不用全改 import。
export type SubgraphSummary = Subgraph

// 编辑器层级 + 全局子图池（NodeInspector / ContainerFlowNode / 子图库 用）。
//
// 数据 vs 视图态的隔离边界 (复发#5 教训, 不回退):
//   - 子图数据**全局一份** (subgraphsById) — 2026-06-12 全局化后子图本来就是全 app 共享资产,
//     多个 keep-alive 编辑器看同一池是正确语义 (改一处全更新)。
//   - **视图态仍按容器隔离** (editorPathByContainer / activeContainerID / touchedByContainer):
//     层级导航、前台指针、"本容器动过哪些子图"绝不跨容器串。
// 主图在视口缓存里的层级 key (子图层级用 sgID)。
const MAIN_GRAPH_KEY = '__main__'

export const useContainerEditorStore = defineStore('containerEditor', () => {
  const activeContainerID = ref<string>('')
  // 全局子图池 byID — backend.subgraphs.list() 全量灌入.
  const subgraphsById = ref<Record<string, Subgraph>>({})
  const editorPathByContainer = ref<Record<string, string[]>>({}) // 主图时为空；进 sub-A 时 = ['sub-A']
  // 本容器编辑会话动过 (待保存) 的子图 ID — 保存流只写这些 (带 rev 乐观锁), 归属清晰不跨容器代保.
  const touchedByContainer = ref<Record<string, string[]>>({})

  // 侧栏 '容器' 跳回正在编辑的容器 — 离开列表/进入编辑器时 set, ContainersView mount 时 clear.
  const lastEditingContainerID = ref<string>('')
  function setLastEditing(id: string) {
    lastEditingContainerID.value = id
  }
  function clearLastEditing() {
    lastEditingContainerID.value = ''
  }

  // ── 全局池访问 ──
  const subgraphList = computed<Subgraph[]>(() => Object.values(subgraphsById.value))
  function subgraphById(id: string): Subgraph | undefined {
    return subgraphsById.value[id]
  }
  // 用户可见子图 = 排除匿名 (CollapsedNode 后备体)。子图库/插入候选用。
  const visibleSubgraphs = computed<Subgraph[]>(() =>
    subgraphList.value.filter((s) => !s.isAnonymous),
  )

  // setPool 整体替换全局池 (首次加载).
  function setPool(list: Subgraph[]) {
    const next: Record<string, Subgraph> = {}
    for (const sg of list) next[sg.id] = sg
    subgraphsById.value = next
  }
  // mergePool 后端快照同步, 但同 id 保内存版 — 正在编辑未落盘的修改不被磁盘版盖掉
  // (录制雪崩根因③同款防御)。后端列表为基准: 池里多出的 (后端已删) 移除。
  function mergePool(fresh: Subgraph[]) {
    const next: Record<string, Subgraph> = {}
    for (const sg of fresh) next[sg.id] = subgraphsById.value[sg.id] ?? sg
    subgraphsById.value = next
  }
  // replaceSubgraph 单条覆盖 (保存成功后回灌带新 rev 的版本 / 重载放弃本地修改).
  function replaceSubgraph(sg: Subgraph) {
    subgraphsById.value[sg.id] = sg
  }

  // ── 视图态: 按容器精确访问 (per-instance composable 用自己的 containerID 调, 不受 active 漂移影响) ──
  function editorPathFor(cid: string): string[] {
    return editorPathByContainer.value[cid] ?? []
  }
  // 只读 computed —— 写走 setPath / 导航 action。
  const editorPath = computed<string[]>(
    () => editorPathByContainer.value[activeContainerID.value] ?? [],
  )

  function setPath(path: string[]) {
    if (activeContainerID.value) editorPathByContainer.value[activeContainerID.value] = path
  }

  // setActiveContainer 设前台 + 初始化该容器视图态 (load / reload 用)。
  function setActiveContainer(id: string) {
    activeContainerID.value = id
    if (!(id in editorPathByContainer.value)) editorPathByContainer.value[id] = []
  }

  // markActive 仅把前台指针指向某容器 (keep-alive onActivated 用)。
  function markActive(id: string) {
    if (id) activeContainerID.value = id
  }

  // 编辑器实例销毁时释放该容器视图态 (全局池不动 — 数据是共享的)。
  function dropContainer(id: string) {
    delete editorPathByContainer.value[id]
    delete touchedByContainer.value[id]
    delete viewportByContainer.value[id]
    if (activeContainerID.value === id) activeContainerID.value = ''
  }

  // ── 子图改动归属 (保存流) ──
  function touchSubgraph(cid: string, sgID: string) {
    const cur = touchedByContainer.value[cid] ?? []
    if (!cur.includes(sgID)) touchedByContainer.value[cid] = [...cur, sgID]
  }
  function touchedFor(cid: string): string[] {
    return touchedByContainer.value[cid] ?? []
  }
  function clearTouched(cid: string, ids?: string[]) {
    if (!ids) {
      delete touchedByContainer.value[cid]
      return
    }
    const cur = touchedByContainer.value[cid] ?? []
    touchedByContainer.value[cid] = cur.filter((x) => !ids.includes(x))
  }

  // 层级导航 — 作用在前台容器视图态上。
  function pushPath(sgID: string) {
    setPath([...editorPath.value, sgID])
  }
  function popPath() {
    setPath(editorPath.value.slice(0, -1))
  }
  function gotoPathIndex(idx: number) {
    setPath(editorPath.value.slice(0, idx + 1))
  }
  function resetPath() {
    setPath([])
  }

  // 1:1 模型 helpers：扫全池子图内 Subgraph 节点引用计数（主图引用由 view 层另算）。
  function countSubgraphRefs(sgID: string): number {
    let count = 0
    for (const sg of subgraphList.value) {
      for (const n of sg.graph?.nodes ?? []) {
        if (n.kind === 'Subgraph' && (n as any).config?.SubgraphID === sgID) count++
      }
    }
    return count
  }

  // ── 视口缓存 (视图态, 按容器隔离) ──
  // 每 (容器, 图层级) 记最后一次相机 viewport。主图↔子图本是同一个 vue-flow 相机, 切层级不存/取
  // 就会"跑飞"(在内容很远的子图 pan 到 19000, 切回主图相机还停在那)。graphID = sgID 或 MAIN_GRAPH_KEY。
  type Viewport = { x: number; y: number; zoom: number }
  const viewportByContainer = ref<Record<string, Record<string, Viewport>>>({})
  function saveViewport(cid: string, graphID: string, vp: Viewport) {
    if (!cid) return
    const m = viewportByContainer.value[cid] ?? (viewportByContainer.value[cid] = {})
    m[graphID] = vp
  }
  function getSavedViewport(cid: string, graphID: string): Viewport | undefined {
    return viewportByContainer.value[cid]?.[graphID]
  }

  // clipboard 暂存 — 跨容器共享是有意的 (容器间 copy/paste; 全局化后引用即语义).
  const clipboard = ref<{
    nodes: any[]
    edges: any[]
    subgraphsDeepCopy: Record<string, any>
  } | null>(null)

  return {
    MAIN_GRAPH_KEY,
    saveViewport,
    getSavedViewport,
    activeContainerID,
    subgraphsById,
    subgraphList,
    subgraphById,
    visibleSubgraphs,
    setPool,
    mergePool,
    replaceSubgraph,
    editorPath,
    editorPathFor,
    lastEditingContainerID,
    setLastEditing,
    clearLastEditing,
    setActiveContainer,
    markActive,
    dropContainer,
    touchSubgraph,
    touchedFor,
    clearTouched,
    setPath,
    pushPath,
    popPath,
    gotoPathIndex,
    resetPath,
    countSubgraphRefs,
    clipboard,
  }
})
