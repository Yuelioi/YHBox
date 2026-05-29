// useRecording 录制流程 (v2 Subgraph-only):
//   1. 倒计时 → recordStore.start(filterMode, containerID)  (hook 已挂, F12 / HUD / toolbar 都能停)
//   2. 停录路径:
//       - toolbar 停: stopRecording() → recordStore.stop() 同步拿 payload
//       - F12 / HUD 停: 后端 emit 'recording:completed' {subgraphID, containerID, label, filterMode} | {error}
//   3. 拿到 subgraphID → refreshSubgraphStore() 让 editorStore.subgraphsForCurrentContainer
//      拿到新 subgraph (activeGraph computed 从这里读, 不读 draft.subgraphs!) →
//      在 activeGraph 加 Subgraph 引用节点 (config.SubgraphID = subgraphID) →
//      autoConnect Start → 自动 saveDraft.
//
// 关键陷阱: container.Subgraphs 在 Go 端是 json:"-", backend.containers.get(id) 拿不到 subgraphs.
// 子图列表走单独 RPC backend.containers.listSubgraphs(cid) → 灌进 editorStore. 录完不 refresh
// store 的话, 用户点新 Subgraph 节点进入会拿到 null → 画布空白.
//
// 精准 vs 简易差别:
//   - 录制层: simple 模式 drainLoop 丢 RawDelta/MouseMove (省内存)
//   - 转换层: simple → KeyPress/ClickAt/Sleep 线性节点链; precise → 单 PlayClip 节点
//   - 前端看到的都是 Subgraph 引用节点, 操作同构.
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Ref, ComputedRef } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Container, type Graph } from '@/lib/backend'
import { useRecordingStore, type RecordingStopPayload } from '@/stores/recording'
import { useSettingsStore } from '@/stores/settings'
import { randID } from './ids'

export interface RecordOpts {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  syncFlowFromDraft: () => void
  // refreshSubgraphStore: 必传 — 录完后调一次, 否则 editorStore 没新 subgraph, 双击进入显示空白.
  refreshSubgraphStore: () => Promise<void> | void
  saveDraft: () => Promise<unknown> | unknown
  toast: { add: (o: Record<string, unknown>) => unknown }
}

export interface StartRecordingOpts {
  // replaceNodeID: NodeInspector 点 "重新录制覆盖" 时传入. 录完不创建新节点,
  // 而是把该节点 (应为 Subgraph kind) config.SubgraphID 改成新 subgraphID.
  // 旧 subgraph 在 container.subgraphs 里变成孤儿, 用户可手动清.
  replaceNodeID?: string
}

export function useRecording(opts: RecordOpts) {
  const { draft, activeGraph, syncFlowFromDraft, refreshSubgraphStore, saveDraft, toast } = opts
  const recordStore = useRecordingStore()
  const settingsStore = useSettingsStore()
  const { t } = useI18n()

  const countdownSec = ref(0)
  const filterMode = ref<'precise' | 'simple'>('precise')
  const replaceNodeID = ref<string | null>(null)

  // startRecording: 倒计时 → recordStore.start(mode, containerID). 倒计时中再点 = 取消.
  async function startRecording(mode: 'precise' | 'simple', extra?: StartRecordingOpts) {
    replaceNodeID.value = extra?.replaceNodeID ?? null
    if (!draft.value) return
    const containerID = draft.value.id
    if (!containerID) {
      toast.add({ title: t('recordComposable.no_container_id'), color: 'error' })
      return
    }
    if (recordStore.isRecording) return
    if (countdownSec.value > 0) {
      countdownSec.value = 0
      toast.add({ title: t('recordComposable.countdown_cancelled'), color: 'neutral' })
      return
    }

    filterMode.value = mode

    try {
      await backend.tools.openRecordingHUD()
    } catch (e) {
      console.warn('OpenRecordingHUD failed (countdown continues in editor)', e)
    }

    // 停录键标签传给 HUD 显示 (settings 配的, 默认 F12). HUD 是独立窗口拿不到本窗 store, 经事件带过去.
    const stopKey = settingsStore.data?.ui?.recordingStopHotkey || 'F12'
    countdownSec.value = 3
    for (let i = 3; i >= 1; i--) {
      countdownSec.value = i
      Events.Emit('recording:countdown', { sec: i, mode, stopKey })
      await new Promise((r) => setTimeout(r, 1000))
      if (countdownSec.value === 0) {
        try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
        Events.Emit('recording:countdown', { sec: 0, mode, stopKey })
        return
      }
    }
    countdownSec.value = 0

    try {
      await recordStore.start(mode, containerID)
      toast.add({
        title: t('recordComposable.recording_in_progress', { mode: mode === 'precise' ? t('recordComposable.mode_precise') : t('recordComposable.mode_simple') }),
        description: t('recordComposable.stop_methods'),
        color: 'success',
        duration: 5000,
      })
    } catch (e: any) {
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      toast.add({ title: t('recording.launch_failed'), description: String(e?.message ?? e), color: 'error' })
    }
  }

  // stopRecording: toolbar / 主窗口主动停 — 同步路径, recordStore.stop() 拿 payload.
  async function stopRecording() {
    if (!recordStore.isRecording) return
    try {
      const payload = await recordStore.stop()
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      if (!payload || !payload.subgraphID) {
        toast.add({ title: t('recording.no_steps'), color: 'warning' })
        return
      }
      await onSubgraphCreated(payload)
    } catch (e: any) {
      toast.add({ title: t('recording.stop_failed'), description: String(e?.message ?? e), color: 'error' })
    }
  }

  // onSubgraphCreated: 录完产物 = Subgraph (后端已落盘). 刷新 editorStore →
  // activeGraph 加 Subgraph 引用节点 → autoConnect Start → 自动保存.
  //
  // ⚠ 必须先 refreshSubgraphStore — activeGraph computed 从 editorStore 读子图列表,
  // 不刷新的话双击进 Subgraph 节点会拿不到, 画布空白 (实测踩过).
  async function onSubgraphCreated(payload: RecordingStopPayload) {
    if (!activeGraph.value || !draft.value) {
      toast.add({
        title: t('recording.completed_no_graph'),
        description: `subgraphID=${payload.subgraphID}`,
        color: 'primary',
      })
      // 仍刷一下 editorStore 让其他视图能看到
      try { await refreshSubgraphStore() } catch { /* ignore */ }
      return
    }

    // 绑定守卫: 子图由后端存进 payload.containerID (录制开始时锁定的容器). 若用户在录制期间
    // 切换/新建到别的容器, draft.value.id 已变 — 此时把 caller 节点加到当前 draft 会引用一个
    // 不在本容器的子图 → 永久 dangling (MISSING_SUBGRAPH + 输出 pin __missing__). 故拒绝加节点,
    // 提示用户切到真正存子图的容器. 子图本身已落盘, 不丢.
    if (payload.containerID && payload.containerID !== draft.value.id) {
      toast.add({
        title: t('recordComposable.container_mismatch', {
          target: payload.containerID,
          current: draft.value.id,
        }),
        color: 'warning',
        duration: 8000,
      })
      try { await refreshSubgraphStore() } catch { /* ignore */ }
      return
    }

    // 1) 刷新 editorStore 让 editorStore.subgraphsForCurrentContainer 拿到新 subgraph
    try {
      await refreshSubgraphStore()
    } catch (e: any) {
      toast.add({ title: t('recordComposable.refresh_subgraphs_failed'), description: String(e?.message ?? e), color: 'error' })
      return
    }

    // 2) 处理 replaceNodeID — 替换目标节点的 SubgraphID
    const replaceID = replaceNodeID.value
    replaceNodeID.value = null

    if (replaceID) {
      const target = (activeGraph.value.nodes as any[]).find((n) => n.id === replaceID)
      if (!target) {
        toast.add({ title: t('recordComposable.replace_node_missing'), color: 'warning' })
      } else if (target.kind !== 'Subgraph') {
        toast.add({
          title: t('recordComposable.replace_node_wrong_kind', { kind: target.kind }),
          color: 'warning',
        })
      } else {
        if (!target.config) target.config = {}
        target.config.SubgraphID = payload.subgraphID
        syncFlowFromDraft()
        await maybeSave()
        toast.add({ title: t('recording.rerecord_overwrite', { name: payload.label }), color: 'success' })
        return
      }
    }

    // 3) 新建 Subgraph 引用节点
    const nodeId = randID('n-sg')
    const newNode = {
      id: nodeId,
      kind: 'Subgraph',
      x: 300 + Math.random() * 100,
      y: 300 + Math.random() * 100,
      config: { SubgraphID: payload.subgraphID },
      createdAt: new Date().toISOString(),
    }
    ;(activeGraph.value.nodes as any[]).push(newNode)
    autoConnectToChainEnd(activeGraph.value, nodeId)
    syncFlowFromDraft()
    await maybeSave()
    toast.add({ title: t('recording.added_subgraph', { name: payload.label }), color: 'success' })
  }

  // autoConnectToChainEnd: BFS 从 Start 找链尾节点 (无出边、非 Stop、非新节点).
  //   - 恰好 1 个链尾 → 连 chainEnd.out → newNode.in
  //   - 0 个 (环) 或 >1 个 (Parallel 分支) → 不连, 留给用户手动
  //   - 无 Start (子图) → 跳过
  function autoConnectToChainEnd(g: Graph, newNodeId: string) {
    const start = (g.nodes as any[]).find((n) => n.kind === 'Start')
    if (!start) return

    // BFS: 收集从 Start 可达的所有节点 ID
    const reachable = new Set<string>([start.id])
    const queue: string[] = [start.id]
    while (queue.length > 0) {
      const cur = queue.shift()!
      for (const edge of g.edges as any[]) {
        if (typeof edge.from === 'string' && edge.from.startsWith(cur + '.')) {
          // edge.from = "<fromNode>.<pin>", extract node ID by stripping last ".xxx"
          const toNodeId = typeof edge.to === 'string' ? edge.to.replace(/\.[^.]+$/, '') : ''
          if (toNodeId && !reachable.has(toNodeId)) {
            reachable.add(toNodeId)
            queue.push(toNodeId)
          }
        }
      }
    }

    // 找链尾: 可达且无出边, 且不是 Stop、不是新节点本身
    const endpoints: string[] = []
    for (const nodeId of reachable) {
      if (nodeId === newNodeId) continue
      const node = (g.nodes as any[]).find((n) => n.id === nodeId)
      if (!node || node.kind === 'Stop') continue
      const hasOut = (g.edges as any[]).some(
        (e: any) => typeof e.from === 'string' && e.from.startsWith(nodeId + '.'),
      )
      if (!hasOut) endpoints.push(nodeId)
    }

    if (endpoints.length === 1) {
      ;(g.edges as any[]).push({ from: endpoints[0] + '.Done', to: newNodeId + '.In' })
    }
    // 0 (环) 或 >1 (Parallel 分支) → 不连
  }

  async function maybeSave() {
    try {
      await saveDraft()
    } catch (e) {
      console.warn('post-recording auto-save failed', e)
    }
  }

  // 窗口重新聚焦 → 跟后端对账录制状态. 这是 "切回 YHBox 自愈" 的核心: 录制中 F12/HUD 停了
  // 但 recording:state 事件没收到 (窗口在后台/race), 聚焦回来立即收敛, 不会卡 "录制中".
  function onWindowFocus() {
    void recordStore.reconcile()
  }

  // 订阅 'recording:completed' (F12 全局热键 / HUD 关 / StopAsync 触发).
  // payload: {subgraphID, containerID, label, filterMode} 或 {error}. (状态本身走 recording:state.)
  let unsubscribe: (() => void) | null = null
  onMounted(() => {
    void recordStore.reconcile() // 挂载即对账, 修正任何陈旧状态
    window.addEventListener('focus', onWindowFocus)
    unsubscribe = Events.On('recording:completed', async (ev: any) => {
      const raw = ev?.data ?? ev
      const firstArg = Array.isArray(raw) ? raw[0] : raw
      // 状态由后端 'recording:state' 广播 (已收敛到 idle); 这里对账一次兜底防丢事件.
      void recordStore.reconcile()
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      const errMsg = firstArg?.error
      if (errMsg) {
        toast.add({ title: t('recordComposable.recording_failed'), description: String(errMsg), color: 'error' })
        return
      }
      const sgID = firstArg?.subgraphID
      if (!sgID) {
        toast.add({ title: t('recordComposable.no_subgraph_id'), color: 'warning' })
        return
      }
      await onSubgraphCreated({
        subgraphID: firstArg.subgraphID,
        containerID: firstArg.containerID,
        label: firstArg.label ?? t('recordComposable.default_clip_name'),
        filterMode: firstArg.filterMode ?? '',
      })
    })
  })
  onUnmounted(() => {
    if (unsubscribe) unsubscribe()
    window.removeEventListener('focus', onWindowFocus)
  })

  return {
    countdownSec,
    filterMode,
    startRecording,
    stopRecording,
    isRecording: () => recordStore.isRecording,
  }
}
