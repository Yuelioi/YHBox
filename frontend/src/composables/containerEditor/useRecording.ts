// useRecording 录制流程 (Phase 4 — InputClip 重写):
//   1. 倒计时 → recordStore.start(filterMode)  (hook 已挂, F12 / HUD / toolbar 都能停)
//   2. 停录路径:
//       - toolbar 停: stopRecording() → recordStore.stop() 拿 clip 同步
//       - F12 / HUD 停: 后端 emit 'recording:completed' {clipID} | {error}
//   3. 拿到 clipID → 父图 activeGraph 加 PlayClip 节点 (config.clipID).
//
// 旧 compactRawEvents + rawStepsToSubgraph 折叠路径已删除 — PlayClip 直接消费 clip.
import { onMounted, onUnmounted, ref } from 'vue'
import type { Ref, ComputedRef } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Container, type Graph } from '@/lib/backend'
import { useRecordingStore } from '@/stores/recording'
import { useClipsStore } from '@/stores/clips'

export interface RecordOpts {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  syncFlowFromDraft: () => void
  saveDraft: () => Promise<unknown> | unknown
  toast: { add: (o: Record<string, unknown>) => unknown }
}

export interface StartRecordingOpts {
  // replaceNodeID: NodeInspector 点 "重新录制覆盖" 时传入. 录完不创建新节点, 而是替换该节点 clipID.
  replaceNodeID?: string
}

export function useRecording(opts: RecordOpts) {
  const { draft, activeGraph, syncFlowFromDraft, saveDraft, toast } = opts
  const recordStore = useRecordingStore()
  const clipsStore = useClipsStore()

  const countdownSec = ref(0)
  const filterMode = ref<'precise' | 'simple'>('precise')
  // 替换目标: 设了就 onClipCreated 时改该节点 config.clipID 而不是建新节点.
  const replaceNodeID = ref<string | null>(null)

  // startRecording: 倒计时 → recordStore.start(mode). 倒计时中再点 = 取消.
  async function startRecording(mode: 'precise' | 'simple', extra?: StartRecordingOpts) {
    replaceNodeID.value = extra?.replaceNodeID ?? null
    if (!draft.value) return
    if (recordStore.isRecording) return
    if (countdownSec.value > 0) {
      // 二次点 = 取消倒计时
      countdownSec.value = 0
      toast.add({ title: '已取消录制倒计时', color: 'neutral' })
      return
    }

    filterMode.value = mode

    // 倒计时一开始就开 HUD 浮窗 (AlwaysOnTop), 用户切游戏窗口也能看到 3-2-1.
    try {
      await backend.tools.openRecordingHUD()
    } catch (e) {
      console.warn('OpenRecordingHUD 失败 (倒计时继续走编辑器内)', e)
    }

    countdownSec.value = 3
    for (let i = 3; i >= 1; i--) {
      countdownSec.value = i
      // 广播给 HUD 窗口同步显示倒计时数字
      Events.Emit('recording:countdown', { sec: i, mode })
      await new Promise((r) => setTimeout(r, 1000))
      if (countdownSec.value === 0) {
        // 取消 → 关 HUD
        try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
        Events.Emit('recording:countdown', { sec: 0, mode })
        return
      }
    }
    countdownSec.value = 0

    try {
      await recordStore.start(mode)
      // 广播 started — HUD 切到 REC 状态.
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

  // stopRecording: toolbar / 主窗口主动停 — 同步路径, recordStore.stop() 拿 clip.
  async function stopRecording() {
    if (!recordStore.isRecording) return
    try {
      const clip = await recordStore.stop()
      try {
        await backend.tools.closeRecordingHUD()
      } catch {
        /* ignore */
      }
      if (!clip || !clip.id) {
        toast.add({ title: '没录到任何步骤', color: 'warning' })
        return
      }
      await onClipCreated(clip.id, clip.label || '录制片段')
    } catch (e: any) {
      toast.add({ title: '停止录制失败', description: String(e?.message ?? e), color: 'error' })
    }
  }

  // onClipCreated: 拿到 clipID → activeGraph 加 PlayClip 节点 + 自动连边 + 自动保存.
  //
  // 精准 vs 简易的区别**只在 recorder 层** (filterMode='simple' 不录 RawDelta + MouseMove),
  // 回放路径同一条 — clip 进 PlayClip 节点直接 backend.Send. 简易 clip 不含相机转向, 不需要
  // MouseCalibration, 也不需要前端再编译成子图节点链.
  async function onClipCreated(clipID: string, label: string) {
    if (!activeGraph.value) {
      toast.add({ title: '录制完成, 但当前没活跃图; 已落盘到 clips', color: 'primary' })
      await clipsStore.refresh()
      return
    }

    const replaceID = replaceNodeID.value
    replaceNodeID.value = null

    if (replaceID) {
      const target = activeGraph.value.nodes.find((n: any) => n.id === replaceID)
      if (!target) {
        toast.add({ title: '替换目标节点已不存在, 改为新建', color: 'warning' })
      } else {
        if (!target.config) target.config = {}
        target.config.clipID = clipID
        syncFlowFromDraft()
        await clipsStore.refresh()
        await maybeSave()
        toast.add({ title: `已重新录制覆盖: ${label}`, color: 'success' })
        return
      }
    }

    const nodeId = 'n-clip-' + Math.random().toString(36).slice(2, 8)
    const newNode = {
      id: nodeId,
      kind: 'PlayClip',
      x: 300 + Math.random() * 100,
      y: 300 + Math.random() * 100,
      config: { clipID },
      createdAt: new Date().toISOString(),
    }
    activeGraph.value.nodes.push(newNode as any)
    autoConnectFromStart(activeGraph.value, nodeId)
    syncFlowFromDraft()
    await clipsStore.refresh()
    await maybeSave()
    toast.add({ title: `已添加 PlayClip 节点: ${label}`, color: 'success' })
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

  // maybeSave: 录制完成后自动落盘 draft, 让 ▶ 试运行直接生效.
  // saveDraft 由 ContainerEditorView 提供 (调 backend.containers.update + setOpen 等流程).
  async function maybeSave() {
    try {
      await saveDraft()
    } catch (e) {
      console.warn('录制完成自动保存失败', e)
    }
  }

  // 订阅 'recording:completed' (F12 全局热键 / HUD 关 / StopAsync 触发).
  // payload: {clipID: string} 或 {error: string}.
  let unsubscribe: (() => void) | null = null
  onMounted(() => {
    unsubscribe = Events.On('recording:completed', async (ev: any) => {
      // wails3 event payload shapes: ev.data 可能是单 object 或 [object] 数组.
      const raw = ev?.data ?? ev
      const firstArg = Array.isArray(raw) ? raw[0] : raw
      ;(recordStore as any).isRecording = false
      try {
        await backend.tools.closeRecordingHUD()
      } catch {
        /* ignore */
      }
      const errMsg = firstArg?.error
      if (errMsg) {
        toast.add({ title: '录制失败', description: String(errMsg), color: 'error' })
        return
      }
      const clipID = firstArg?.clipID
      if (!clipID) {
        toast.add({ title: '录制结束但未拿到 clipID', color: 'warning' })
        return
      }
      const clip = await clipsStore.get(clipID)
      await onClipCreated(clipID, clip?.label || '录制片段')
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
