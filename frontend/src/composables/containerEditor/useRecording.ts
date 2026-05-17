// useRecording 录制流程 (v2 Subgraph-only):
//   1. 倒计时 → recordStore.start(filterMode, containerID)  (hook 已挂, F12 / HUD / toolbar 都能停)
//   2. 停录路径:
//       - toolbar 停: stopRecording() → recordStore.stop() 同步拿 payload
//       - F12 / HUD 停: 后端 emit 'recording:completed' {subgraphID, containerID, label, filterMode} | {error}
//   3. 拿到 subgraphID → refreshSubgraphStore() 让 editorStore.subgraphsForCurrentContainer
//      拿到新 subgraph (activeGraph computed 从这里读, 不读 draft.subgraphs!) →
//      在 activeGraph 加 Subgraph 引用节点 (config.subgraphId = subgraphID) →
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
import type { Ref, ComputedRef } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Container, type Graph } from '@/lib/backend'
import { useRecordingStore, type RecordingStopPayload } from '@/stores/recording'

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
  // 而是把该节点 (应为 Subgraph kind) config.subgraphId 改成新 subgraphID.
  // 旧 subgraph 在 container.subgraphs 里变成孤儿, 用户可手动清.
  replaceNodeID?: string
}

export function useRecording(opts: RecordOpts) {
  const { draft, activeGraph, syncFlowFromDraft, refreshSubgraphStore, saveDraft, toast } = opts
  const recordStore = useRecordingStore()

  const countdownSec = ref(0)
  const filterMode = ref<'precise' | 'simple'>('precise')
  const replaceNodeID = ref<string | null>(null)

  // startRecording: 倒计时 → recordStore.start(mode, containerID). 倒计时中再点 = 取消.
  async function startRecording(mode: 'precise' | 'simple', extra?: StartRecordingOpts) {
    replaceNodeID.value = extra?.replaceNodeID ?? null
    if (!draft.value) return
    const containerID = draft.value.id
    if (!containerID) {
      toast.add({ title: '当前 container 没 ID, 无法录制', color: 'error' })
      return
    }
    if (recordStore.isRecording) return
    if (countdownSec.value > 0) {
      countdownSec.value = 0
      toast.add({ title: '已取消录制倒计时', color: 'neutral' })
      return
    }

    filterMode.value = mode

    try {
      await backend.tools.openRecordingHUD()
    } catch (e) {
      console.warn('OpenRecordingHUD 失败 (倒计时继续走编辑器内)', e)
    }

    countdownSec.value = 3
    for (let i = 3; i >= 1; i--) {
      countdownSec.value = i
      Events.Emit('recording:countdown', { sec: i, mode })
      await new Promise((r) => setTimeout(r, 1000))
      if (countdownSec.value === 0) {
        try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
        Events.Emit('recording:countdown', { sec: 0, mode })
        return
      }
    }
    countdownSec.value = 0

    try {
      await recordStore.start(mode, containerID)
      Events.Emit('recording:started', { mode })
      toast.add({
        title: '正在录制 (' + (mode === 'precise' ? '精准' : '简易') + ')',
        description: '游戏前台按 F12 / 点悬浮窗 / 切回 YHBox 都能停止',
        color: 'success',
        duration: 5000,
      })
    } catch (e: any) {
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      toast.add({ title: '启动录制失败', description: String(e?.message ?? e), color: 'error' })
    }
  }

  // stopRecording: toolbar / 主窗口主动停 — 同步路径, recordStore.stop() 拿 payload.
  async function stopRecording() {
    if (!recordStore.isRecording) return
    try {
      const payload = await recordStore.stop()
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      if (!payload || !payload.subgraphID) {
        toast.add({ title: '没录到任何步骤', color: 'warning' })
        return
      }
      await onSubgraphCreated(payload)
    } catch (e: any) {
      toast.add({ title: '停止录制失败', description: String(e?.message ?? e), color: 'error' })
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
        title: '录制完成, 但当前没活跃图; subgraph 已落盘到容器',
        description: `subgraphID=${payload.subgraphID}`,
        color: 'primary',
      })
      // 仍刷一下 editorStore 让其他视图能看到
      try { await refreshSubgraphStore() } catch { /* ignore */ }
      return
    }

    // 1) 刷新 editorStore 让 editorStore.subgraphsForCurrentContainer 拿到新 subgraph
    try {
      await refreshSubgraphStore()
    } catch (e: any) {
      toast.add({ title: '刷新子图列表失败', description: String(e?.message ?? e), color: 'error' })
      return
    }

    // 2) 处理 replaceNodeID — 替换目标节点的 subgraphId
    const replaceID = replaceNodeID.value
    replaceNodeID.value = null

    if (replaceID) {
      const target = (activeGraph.value.nodes as any[]).find((n) => n.id === replaceID)
      if (!target) {
        toast.add({ title: '替换目标节点已不存在, 改为新建', color: 'warning' })
      } else if (target.kind !== 'Subgraph') {
        toast.add({
          title: `目标节点 kind=${target.kind}, 非 Subgraph, 改为新建`,
          color: 'warning',
        })
      } else {
        if (!target.config) target.config = {}
        target.config.subgraphId = payload.subgraphID
        syncFlowFromDraft()
        await maybeSave()
        toast.add({ title: `已重新录制覆盖: ${payload.label}`, color: 'success' })
        return
      }
    }

    // 3) 新建 Subgraph 引用节点
    const nodeId = 'n-sg-' + Math.random().toString(36).slice(2, 8)
    const newNode = {
      id: nodeId,
      kind: 'Subgraph',
      x: 300 + Math.random() * 100,
      y: 300 + Math.random() * 100,
      config: { subgraphId: payload.subgraphID },
      createdAt: new Date().toISOString(),
    }
    ;(activeGraph.value.nodes as any[]).push(newNode)
    autoConnectFromStart(activeGraph.value, nodeId)
    syncFlowFromDraft()
    await maybeSave()
    toast.add({ title: `已添加 Subgraph 节点: ${payload.label}`, color: 'success' })
  }

  // autoConnectFromStart: 找 Start 节点, 若它没下游就连一条 Start.out → newNode.in.
  // 主图通常只一个 Start; 子图无 Start 跳过. 已有下游的 Start 不动 (用户已自行组流程).
  function autoConnectFromStart(g: Graph, newNodeId: string) {
    const start = (g.nodes as any[]).find((n) => n.kind === 'Start')
    if (!start) return
    const hasOut = (g.edges as any[]).some((e: any) => typeof e.from === 'string' && e.from === start.id + '.out')
    if (hasOut) return
    ;(g.edges as any[]).push({ from: start.id + '.out', to: newNodeId + '.in' })
  }

  async function maybeSave() {
    try {
      await saveDraft()
    } catch (e) {
      console.warn('录制完成自动保存失败', e)
    }
  }

  // 订阅 'recording:completed' (F12 全局热键 / HUD 关 / StopAsync 触发).
  // payload: {subgraphID, containerID, label, filterMode} 或 {error}.
  let unsubscribe: (() => void) | null = null
  onMounted(() => {
    unsubscribe = Events.On('recording:completed', async (ev: any) => {
      const raw = ev?.data ?? ev
      const firstArg = Array.isArray(raw) ? raw[0] : raw
      ;(recordStore as any).isRecording = false
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      const errMsg = firstArg?.error
      if (errMsg) {
        toast.add({ title: '录制失败', description: String(errMsg), color: 'error' })
        return
      }
      const sgID = firstArg?.subgraphID
      if (!sgID) {
        toast.add({ title: '录制结束但未拿到 subgraphID', color: 'warning' })
        return
      }
      await onSubgraphCreated({
        subgraphID: firstArg.subgraphID,
        containerID: firstArg.containerID,
        label: firstArg.label ?? '录制片段',
        filterMode: firstArg.filterMode ?? '',
      })
    })
  })
  onUnmounted(() => {
    if (unsubscribe) unsubscribe()
  })

  return {
    countdownSec,
    filterMode,
    startRecording,
    stopRecording,
    isRecording: () => recordStore.isRecording,
  }
}
