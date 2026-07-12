import { computed, nextTick, onActivated, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MarkerType } from '@vue-flow/core'
import {
  backend,
  type Container,
  type Graph,
  type GraphNode,
  type GraphEdge,
  type Subgraph,
} from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { edgeKind } from '@/components/containers/pinSpec'
import { useSidebarPrefs } from '@/composables/editor/useSidebarPrefs'
import { SUBGRAPH_ENTRY_DEFAULT, SUBGRAPH_OUTPUT_DEFAULT } from './constants'
import * as hist from './historyEngine'

// FlowNode / FlowEdge: vue-flow 渲染数据形态 (与后端 GraphNode/GraphEdge 区分)
export interface FlowNode {
  id: string
  type: string
  position: { x: number; y: number }
  // B2: _virtual=true 标识 entry/output marker (来源 sg.entry / sg.outputPins, 不在 graph.nodes).
  // useGraphMutations / onDeleteSelected 用 _virtual 拒删/拒复制; 拖动时 patch metadata 不 patch nodes.
  data: {
    kind: string
    config?: Record<string, any>
    disabled?: boolean
    label?: string
    _virtual?: boolean
    _markerRole?: 'entry' | 'output'
    _declID?: string
  }
}
export interface FlowEdge {
  id: string
  source: string
  target: string
  sourceHandle?: string
  targetHandle?: string
  type?: string
  animated?: boolean
  style?: Record<string, any>
  markerEnd?: any
}

/**
 * useContainerDraft 容器编辑器数据中心。
 *  - draft: 本地编辑副本
 *  - dirty: 未保存改动标记
 *  - activeGraph: 跟随 editorPath 切换的当前层级 graph
 *  - flowNodes / flowEdges: vue-flow 渲染数据
 *  - syncFlowFromDraft(): draft → flow 渲染数据
 *  - refreshSubgraphStore(): 后端 listSubgraphs 同步到 store
 */
export function useContainerDraft(containerID: string) {
  const editorStore = useContainerEditorStore()
  const { t } = useI18n()

  const draft = ref<Container | null>(null)
  const dirty = ref(false)
  const flowNodes = ref<FlowNode[]>([])
  const flowEdges = ref<FlowEdge[]>([])
  // 连线样式偏好 (曲线/直角/折线) — 单例 pref, 切换时 watch 重刷 flowEdges.
  const { prefs: sidebarPrefs } = useSidebarPrefs()

  // ---- Undo/redo history (纯引擎在 historyEngine.ts) ----
  const histState = ref<hist.HistoryState>({ entries: [], idx: -1 })
  const canUndo = computed(() => hist.canUndo(histState.value))
  const canRedo = computed(() => hist.canRedo(histState.value))

  // 取指定子图当前状态 (给批量撤销条目带上); 引擎内部再深拷贝, 这里给引用即可。
  function touchedSubgraphSnapshot(ids: string[]): Record<string, Subgraph> {
    const out: Record<string, Subgraph> = {}
    for (const id of ids) {
      const sg = editorStore.subgraphById(id)
      if (sg) out[id] = sg as Subgraph
    }
    return out
  }

  // 视图态按本容器隔离 (editorPathFor 自己的 containerID 读), 不受前台 activeContainerID
  // 漂移影响; 子图数据走全局池 (全 app 共享资产, 多编辑器看同一份是全局化后的正确语义)。
  const myPath = computed(() => editorStore.editorPathFor(containerID))

  // activeGraph: 跟随 editorPath 切换. 空 path → 主图; 否则 = 当前 path 末尾对应子图的 graph
  // (全局池 byID lookup)。返回 Graph | null, 不带 any: 下游 mutation 必须显式 null check.
  const activeGraph = computed<Graph | null>(() => {
    if (!draft.value) return null
    const path = myPath.value
    if (path.length === 0) return draft.value.graph
    const cur = editorStore.subgraphById(path[path.length - 1])
    return (cur?.graph as Graph) ?? null
  })

  function syncFlowFromDraft() {
    if (!draft.value) return
    const g = activeGraph.value
    if (!g) {
      flowNodes.value = []
      flowEdges.value = []
      return
    }
    const realNodes: FlowNode[] = g.nodes.map((n: GraphNode) => ({
      id: n.id,
      type: n.kind,
      position: { x: n.x, y: n.y },
      data: { kind: n.kind, config: n.config, disabled: n.disabled === true, label: n.label ?? '' },
    }))
    // B2: 在 subgraph 上下文里, 注入 entry/output virtual marker flowNodes.
    // 这些 node ID 跟 sg.entry.nodeID / sg.outputPins[].nodeID 对齐, edges 引用它们.
    const virtualNodes: FlowNode[] = []
    if (myPath.value.length > 0) {
      const curSgID = myPath.value[myPath.value.length - 1]
      const sg = editorStore.subgraphById(curSgID)
      if (sg) {
        if (sg.entry && sg.entry.nodeID) {
          virtualNodes.push({
            id: sg.entry.nodeID,
            type: 'SubgraphInput',
            position: {
              x: sg.entry.x ?? SUBGRAPH_ENTRY_DEFAULT.x,
              y: sg.entry.y ?? SUBGRAPH_ENTRY_DEFAULT.y,
            },
            data: {
              kind: 'SubgraphInput',
              config: {},
              disabled: false,
              label: t('editorAux.entry_pin'),
              _virtual: true,
              _markerRole: 'entry',
            },
          })
        }
        for (const p of sg.outputPins ?? []) {
          if (!p.nodeID) continue
          virtualNodes.push({
            id: p.nodeID,
            type: 'SubgraphOutput',
            position: { x: p.x ?? SUBGRAPH_OUTPUT_DEFAULT.x, y: p.y ?? SUBGRAPH_OUTPUT_DEFAULT.y },
            data: {
              kind: 'SubgraphOutput',
              config: { DeclID: p.id },
              disabled: false,
              label: p.name ?? t('editorAux.output_pin'),
              _virtual: true,
              _markerRole: 'output',
              _declID: p.id,
            },
          })
        }
      }
    }
    flowNodes.value = [...virtualNodes, ...realNodes]
    flowEdges.value = g.edges.map((e: GraphEdge, i: number) => {
      const dot = e.from.indexOf('.')
      const src = e.from.slice(0, dot)
      const srcPin = e.from.slice(dot + 1)
      const dot2 = e.to.indexOf('.')
      const tgt = e.to.slice(0, dot2)
      const tgtPin = e.to.slice(dot2 + 1)
      const fromKind = g.nodes.find((n) => n.id === src)?.kind ?? ''
      const isData = edgeKind(fromKind, srcPin) === 'data'
      return {
        id: 'e' + i,
        source: src,
        target: tgt,
        sourceHandle: srcPin,
        targetHandle: tgtPin,
        type: sidebarPrefs.value.edgeStyle,
        animated: isData,
        style: isData
          ? { stroke: '#60a5fa', strokeWidth: 1.5, strokeDasharray: '4 4' } // data edge: dashed blue
          : { stroke: '#a1a1aa', strokeWidth: 1.5 }, // exec edge: solid zinc
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: isData ? '#60a5fa' : '#a1a1aa',
        },
      }
    })
  }

  // editorPath 变化时重新 sync（进/出子图层级）— 不参与 dirty 判定，可立即装。
  // watch 本容器 slot 的 path, 不watch 全局 active (别的容器切前台不该触发本实例 re-sync)。
  watch(
    () => myPath.value,
    () => {
      syncFlowFromDraft()
    },
    { deep: true },
  )

  // 连线样式切换 → 重建 flowEdges (type 变) — 不动 draft, 不标 dirty.
  watch(
    () => sidebarPrefs.value.edgeStyle,
    () => syncFlowFromDraft(),
  )

  // dirty watcher 推迟到 onMounted load 完成后才装 (见末尾 installDirtyWatchers)
  // 原方案: watch 在 setup 阶段立即装 + setTimeout(0) 重置初始 fire — 依赖 microtask 顺序, brittle.
  // 新方案: load 完毕显式装 watcher, 初始化期间根本不触发, 不需要重置 hack.

  /**
   * loadIntoEditor — 把一份 Container 装进编辑器: 重置 draft / 子图 store / flow 渲染 / undo 基线。
   * onMounted 首次加载 + reload 重载共用。不装 watcher、不动 editorPath。
   */
  async function loadIntoEditor(c: Container) {
    draft.value = JSON.parse(JSON.stringify(c))
    // 全局子图池同步 (NodeInspector / ContainerFlowNode + activeGraph computed 用)。
    // mergePool: 同 id 保内存版 — 别的编辑器正在编辑未落盘的修改不被盖掉。
    editorStore.setActiveContainer(c.id)
    try {
      const pool = (await backend.subgraphs.list()) ?? []
      editorStore.mergePool(pool)
    } catch (e) {
      console.warn('subgraphs.list failed', e)
    }
    syncFlowFromDraft()
    // 撤销基线: 刚加载的状态 = idx 0
    if (draft.value) histState.value = hist.initHistory(draft.value)
  }

  onMounted(async () => {
    if (!containerID) return
    const r = await backend.containers.get(containerID)
    if (r === undefined) {
      console.error('container not found:', containerID)
      return
    }
    await loadIntoEditor(r as unknown as Container)
    // load 完毕后才装 dirty watcher; 初始化阶段不触发 → 不再需要 setTimeout 重置 hack。
    // 子图 dirty: 只 watch 本实例当前正在编辑的那张图 (activeGraph), 不 watch 全局池 ——
    // 否则别的容器编辑器改池里任意子图会误标本实例 dirty (复发#5 同类误触发)。
    // 改动归属: path 非空时记 touch(本容器, 当前子图) — 保存流只写本容器动过的子图。
    watch(
      draft,
      () => {
        dirty.value = true
      },
      { deep: true },
    )
    watch(
      () => activeGraph.value,
      (g, prev) => {
        const path = myPath.value
        if (path.length === 0) return // 主图编辑 → draft watcher 已覆盖
        // deep watch 触发分两种: ① 当前图深层内容被改 (真编辑 → g === prev, 同一对象被 mutate);
        // ② activeGraph 指向变了 (进/出子图层级、或子图被 replace → g !== prev)。后者只是导航/换引用,
        // 不是编辑 —— 不标 dirty (否则"只是进子图什么都没改"也会变未保存)。
        if (g !== prev) return
        dirty.value = true
        editorStore.touchSubgraph(containerID, path[path.length - 1])
      },
      { deep: true },
    )
  })

  // keep-alive 缓存命中重新激活: 先把"前台容器"指针指回本实例。
  // 再**重拉本容器子图**: 后台期间本容器可能被 import/分享/外部写入新子图 (写盘但没刷本 keep-alive
  // 编辑器的 store slot) → 不重拉则 subgraphsByContainer 滞后 → Subgraph 节点解析不到 → __missing__
  // → 主图保存被拒 (反复出现的"子图未找到"根因之一)。mergeSubgraphs 保留内存未落盘的编辑、仅补盘上新子图,
  // 且只动本容器 slot (按容器隔离), 安全。
  onActivated(() => {
    if (containerID) editorStore.markActive(containerID)
    void refreshSubgraphStore()
  })

  // 实例销毁 (keep-alive 淘汰 / 离开不缓存) → 释放本容器 store slot, 防多容器编辑后内存堆积。
  onBeforeUnmount(() => {
    if (containerID) editorStore.dropContainer(containerID)
  })

  /**
   * reload — 从磁盘重读这个容器, 覆盖编辑器当前内容 (配合 MCP / 外部改盘)。
   * 先调后端 Reload RPC 让 byID 缓存刷新、拿到最新容器, 再 loadIntoEditor。
   * 强制回主图根 (子图可能已被外部改/删)。重载后清 dirty —— loadIntoEditor 替换 draft
   * 会触发已装的 dirty watcher, 故 await nextTick() 等其 fire 完再强制 false。
   * 失败 (容器被删等) 向上抛, 由调用方 toast。
   */
  async function reload() {
    if (!containerID) return
    const c = (await backend.containers.reload(containerID)) as unknown as Container
    editorStore.resetPath()
    // 放弃本容器动过的子图: 用盘上版本覆盖内存版, 再清归属 — 别的编辑器的未落盘修改不受波及。
    try {
      const fresh = (await backend.subgraphs.list()) ?? []
      const freshById = new Map(fresh.map((s) => [s.id, s]))
      for (const id of editorStore.touchedFor(containerID)) {
        const f = freshById.get(id)
        if (f) editorStore.replaceSubgraph(f)
      }
      editorStore.clearTouched(containerID)
      editorStore.mergePool(fresh)
    } catch (e) {
      console.warn('reload subgraph pool failed', e)
    }
    await loadIntoEditor(c)
    await nextTick()
    dirty.value = false
  }

  /** 重新从后端拉全局子图池并合并 (保留内存未落盘编辑) */
  async function refreshSubgraphStore() {
    try {
      const fresh = (await backend.subgraphs.list()) ?? []
      editorStore.mergePool(fresh)
    } catch (e) {
      console.warn('refreshSubgraphStore failed', e)
    }
  }

  /**
   * applyDraftMutation — 单向数据流 mutation 入口.
   * mutator 收到 draft 直接 mutate, wrapper 自动 dirty + sync + history snapshot.
   * 所有 var / node / edge mutation 都走这, flowNodes 永远 derive 自 draft.
   *
   * Coalescing: vue-flow fires separate @nodes-change events per deleted node,
   * so deleting 2 nodes triggers 2 calls within a few ms. Without coalescing
   * each call pushes a snapshot → Ctrl+Z requires N presses for N nodes.
   * Fix: within COALESCE_MS of the last snapshot, mutate the last history slot
   * in-place instead of pushing a new one. User sees 1 undo step per "burst".
   */
  const COALESCE_MS = 200
  let lastSnapshotAt = 0

  function applyDraftMutation(mutator: (draft: Container) => void): void {
    if (!draft.value) return
    // 在子图里编辑时, 当前活动子图 (path 末尾) 也要进历史快照, 否则 undo 只回退主图、子图改动退不回。
    const sgID = myPath.value.length > 0 ? myPath.value[myPath.value.length - 1] : null
    const now = Date.now()
    const coalescing = histState.value.idx >= 0 && now - lastSnapshotAt < COALESCE_MS
    // 改前 (每段 burst 第一次): 把活动子图"改前"状态 merge 进当前条目, 让 undo 能回到 burst 前的子图。
    if (sgID && !coalescing) {
      histState.value = hist.augmentTop(histState.value, touchedSubgraphSnapshot([sgID]))
    }
    mutator(draft.value)
    dirty.value = true
    syncFlowFromDraft()
    // 主图改 → 条目无 sgState (行为同旧); 子图改 → 带活动子图"改后"状态。
    const entry = hist.makeEntry(draft.value, sgID ? touchedSubgraphSnapshot([sgID]) : undefined)
    // burst 窗口内 → 原地替换 top (一步撤销整段 burst); 否则推新条目 (截断 redo + cap)。
    if (coalescing) {
      histState.value = hist.replaceTop(histState.value, entry)
    } else {
      histState.value = hist.pushEntry(histState.value, entry)
    }
    lastSnapshotAt = now
  }

  /**
   * applyBulkMutation — yt 控制台批量改专用: 主图 + 触及子图一次性改, 落成**一条**可撤销条目。
   * mutator 改主图 draft (主图节点) + 经 editorStore 改子图节点; touchedSgIDs = 被改子图 id。
   * undo/redo 时主图 + 这些子图一起还原。详见 specs/2026-06-17-yt-scripting-console.md §撤销机制。
   */
  function applyBulkMutation(touchedSgIDs: string[], mutator: (draft: Container) => void): void {
    if (!draft.value) return
    // 改前: 把触及子图的改前状态 merge 进当前条目, 让 undo 能回到改前子图。
    histState.value = hist.augmentTop(histState.value, touchedSubgraphSnapshot(touchedSgIDs))
    mutator(draft.value)
    for (const id of touchedSgIDs) editorStore.touchSubgraph(containerID, id)
    dirty.value = true
    syncFlowFromDraft()
    // 改后: 推新条目带改后子图状态。
    histState.value = hist.pushEntry(
      histState.value,
      hist.makeEntry(draft.value, touchedSubgraphSnapshot(touchedSgIDs)),
    )
    lastSnapshotAt = Date.now()
  }

  // 把一条历史条目写回: 主图 draft + (若有) 子图回灌 editorStore。
  function applyRestore(entry: hist.HistoryEntry | null): void {
    if (!entry || !draft.value) return
    const { draft: d, subgraphs } = hist.restore(entry)
    draft.value = d
    for (const sg of subgraphs) editorStore.replaceSubgraph(sg)
    dirty.value = true
    syncFlowFromDraft()
  }
  function undo() {
    const { state: ns, entry } = hist.undo(histState.value)
    if (!entry) return
    histState.value = ns
    applyRestore(entry)
  }
  function redo() {
    const { state: ns, entry } = hist.redo(histState.value)
    if (!entry) return
    histState.value = ns
    applyRestore(entry)
  }

  return {
    draft,
    dirty,
    activeGraph,
    applyDraftMutation,
    applyBulkMutation,
    undo,
    redo,
    canUndo,
    canRedo,
    flowNodes,
    flowEdges,
    syncFlowFromDraft,
    refreshSubgraphStore,
    reload,
  }
}
