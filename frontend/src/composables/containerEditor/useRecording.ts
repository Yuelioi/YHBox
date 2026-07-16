// useRecording 录制流程:
//   1. 倒计时 → recordStore.start(containerID)（hook 已挂，F12 / HUD / toolbar 都能停）
//   2. 停录路径:
//       - toolbar 停: stopRecording() → recordStore.stop() 同步拿 payload
//       - F12 / HUD 停: 后端 emit 'recording:completed' {pendingID, containerID, durationUs, eventCount} | {error}
//   3. Stop 只拿 pending token; 用户填写名称/分类/标签后 Finalize 才创建资产.
//   4. 生成一个不可变 InputClip，并在当前视口中心添加 PlayClip 节点。
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Ref, ComputedRef } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Container, type Graph } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import {
  useRecordingStore,
  type RecordingFinalizePayload,
  type RecordingStopPayload,
} from '@/stores/recording'
import { useHotkeysStore } from '@/stores/hotkeys'
import { randID } from './ids'

export interface RecordOpts {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  syncFlowFromDraft: () => void
  saveDraft: () => Promise<unknown> | unknown
  // dropPoint: 录制产物节点的落点 = 节点左上角 (flow 坐标), 已含居中补偿 → 节点视觉落在
  // 当前视口正中 (录完出现在用户正看的地方). 实参是 useInsertPoint().viewportCenterForNode.
  dropPoint: () => { x: number; y: number }
  // selectNode: 落下后选中该节点 (用户接着自己接线).
  selectNode: (id: string) => void
  toast: { add: (o: Record<string, unknown>) => unknown }
}

export interface StartRecordingOpts {
  // replaceNodeID: NodeInspector 点 "重新录制覆盖" 时传入. 录完不创建新节点, 而是改写目标节点的引用
  // 目标必须为 PlayClip，完成后改写 config.ClipID；kind 不匹配则创建新节点。
  replaceNodeID?: string
}

export function useRecording(opts: RecordOpts) {
  const { draft, activeGraph, syncFlowFromDraft, saveDraft, dropPoint, selectNode, toast } = opts
  const recordStore = useRecordingStore()
  const hotkeysStore = useHotkeysStore()
  const { t } = useI18n()

  const countdownSec = ref(0)
  const replaceNodeID = ref<string | null>(null)
  const pendingRecording = ref<RecordingStopPayload | null>(null)
  const pendingBusy = ref(false)
  const pendingReplaceMode = computed(() => !!replaceNodeID.value)
  // 归属化: 仅本窗发起的录制才处理 recording:completed. recording:completed 是全局广播,
  // 多个容器编辑器窗口都订阅 — 非发起方窗口若也处理会误报 container_mismatch (子图存对了, 别的窗口瞎报警).
  const ownsRecording = ref(false)

  // startRecording: 倒计时 → recordStore.start(containerID). 倒计时中再点 = 取消.
  async function startRecording(extra?: StartRecordingOpts) {
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

    // 倒计时前预检窗口 (① 没设/找不到窗口立刻报错, 不用等录完; ④ 顺带把游戏拉到前台).
    // 失败 → 不开 HUD 不进倒计时. Start 内仍有同样校验作 race 兜底.
    try {
      await backend.recording.validateTarget(containerID)
    } catch (e: any) {
      toast.add({
        title: t('recording.launch_failed'),
        description: errorMessage(e),
        color: 'error',
      })
      return
    }

    try {
      await backend.tools.openRecordingHUD()
    } catch (e) {
      console.warn('OpenRecordingHUD failed (countdown continues in editor)', e)
    }

    // 停录/暂停键标签传给 HUD 显示. registry 是权威 (用户在「快捷键」页 rebind 即时生效),
    // 先 reload 拿最新值. HUD 是独立窗口拿不到本窗 store, 经事件带过去.
    await hotkeysStore.reload()
    const stopKey = hotkeysStore.list.find((e) => e.key === 'recording.stop')?.hotkeyStr || 'F12'
    const pauseKey = hotkeysStore.list.find((e) => e.key === 'recording.pause')?.hotkeyStr || 'F11'
    countdownSec.value = 3
    for (let i = 3; i >= 1; i--) {
      countdownSec.value = i
      Events.Emit('recording:countdown', { sec: i, stopKey, pauseKey })
      await new Promise((r) => setTimeout(r, 1000))
      if (countdownSec.value === 0) {
        try {
          await backend.tools.closeRecordingHUD()
        } catch {
          /* ignore */
        }
        Events.Emit('recording:countdown', { sec: 0, stopKey, pauseKey })
        return
      }
    }
    countdownSec.value = 0

    try {
      await recordStore.start(containerID)
      ownsRecording.value = true // 本窗发起 — 后续 recording:completed 由本窗处理
      toast.add({
        title: t('recordComposable.recording_in_progress'),
        description: t('recordComposable.stop_methods', {
          hk: hotkeysStore.keyFor('recording.stop', 'F12'),
        }),
        color: 'success',
        duration: 5000,
      })
    } catch (e: any) {
      try {
        await backend.tools.closeRecordingHUD()
      } catch {
        /* ignore */
      }
      toast.add({
        title: t('recording.launch_failed'),
        description: errorMessage(e),
        color: 'error',
      })
    }
  }

  // stopRecording: toolbar / 主窗口主动停 — 同步路径, recordStore.stop() 拿 payload.
  async function stopRecording() {
    // recording 或 paused 都可停 (paused 直接停, 后端守卫已放开).
    if (!recordStore.isRecording && !recordStore.isPaused) return
    try {
      const payload = await recordStore.stop()
      try {
        await backend.tools.closeRecordingHUD()
      } catch {
        /* ignore */
      }
      if (!payload?.pendingID) {
        toast.add({ title: t('recording.no_steps'), color: 'warning' })
        return
      }
      presentPending(payload)
    } catch (e: any) {
      toast.add({ title: t('recording.stop_failed'), description: errorMessage(e), color: 'error' })
    }
  }

  function presentPending(payload: RecordingStopPayload) {
    ownsRecording.value = false
    pendingRecording.value = payload
  }

  async function finalizePending(metadata: {
    label: string
    description: string
    category: string
    tags: string[]
  }) {
    const pending = pendingRecording.value
    if (!pending || pendingBusy.value) return
    pendingBusy.value = true
    try {
      const product = await recordStore.finalize({ pendingID: pending.pendingID, ...metadata })
      pendingRecording.value = null
      await attachFinalizedProduct(product)
    } catch (e: any) {
      toast.add({
        title: t('recordingSave.save_failed'),
        description: errorMessage(e),
        color: 'error',
      })
    } finally {
      pendingBusy.value = false
    }
  }

  async function discardPending() {
    const pending = pendingRecording.value
    if (!pending || pendingBusy.value) return
    pendingBusy.value = true
    try {
      await recordStore.discard(pending.pendingID)
      pendingRecording.value = null
      replaceNodeID.value = null
    } catch (e: any) {
      toast.add({
        title: t('recordingSave.discard_failed'),
        description: errorMessage(e),
        color: 'error',
      })
    } finally {
      pendingBusy.value = false
    }
  }

  async function attachFinalizedProduct(payload: RecordingFinalizePayload) {
    if (!activeGraph.value || !draft.value) {
      toast.add({
        title: t('recording.completed_no_graph'),
        description: `clipID=${payload.clipID}`,
        color: 'primary',
      })
      return
    }

    // 绑定守卫: 产物由后端存进 payload.containerID (录制开始时锁定的容器). 若用户在录制期间
    // 切换/新建到别的容器, draft.value.id 已变 — 此时把节点加到当前 draft 会引用一个不属当前上下文的
    // 产物. 故拒绝加节点, 提示用户切回. 产物本身已落盘 (子图/clip 都是全局资产), 不丢.
    if (payload.containerID && payload.containerID !== draft.value.id) {
      // 显示用容器名而非 UUID. target 名查容器表 (取不到 fallback 短 ID).
      let targetName = payload.containerID
      try {
        const c = await backend.containers.get(payload.containerID)
        if (c?.name) targetName = c.name
      } catch {
        /* fallback 裸 ID */
      }
      toast.add({
        title: t('recordComposable.container_mismatch', {
          target: targetName,
          current: draft.value.name || draft.value.id,
        }),
        color: 'warning',
        duration: 8000,
      })
      return
    }

    // 处理 replaceNodeID — 重新录制覆盖 PlayClip 的引用。
    const replaceID = replaceNodeID.value
    replaceNodeID.value = null

    if (replaceID) {
      const target = (activeGraph.value.nodes as any[]).find((n) => n.id === replaceID)
      const wantKind = 'PlayClip'
      if (!target) {
        toast.add({ title: t('recordComposable.replace_node_missing'), color: 'warning' })
      } else if (target.kind !== wantKind) {
        toast.add({
          title: t('recordComposable.replace_node_wrong_kind', { kind: target.kind }),
          color: 'warning',
        })
      } else {
        if (!target.config) target.config = {}
        target.config.ClipID = payload.clipID
        syncFlowFromDraft()
        await maybeSave()
        toast.add({
          title: t('recording.rerecord_overwrite', { name: payload.label }),
          color: 'success',
        })
        return
      }
    }

    // 新建 PlayClip 节点 — 落在当前视口中心，不自动连线，落下即选中。
    const nodeId = randID('n-clip')
    const pt = dropPoint()
    const newNode = {
      id: nodeId,
      kind: 'PlayClip',
      x: pt.x,
      y: pt.y,
      config: { ClipID: payload.clipID },
      createdAt: new Date().toISOString(),
    }
    ;(activeGraph.value.nodes as any[]).push(newNode)
    syncFlowFromDraft()
    await maybeSave()
    selectNode(nodeId)
    toast.add({
      title: t('recording.added_clip', {
        name: payload.label,
      }),
      color: 'success',
    })
  }

  async function maybeSave() {
    try {
      await saveDraft()
    } catch (e) {
      console.warn('post-recording auto-save failed', e)
    }
  }

  // 窗口重新聚焦 → 跟后端对账录制状态. 这是 "切回 Yotta 自愈" 的核心: 录制中 F12/HUD 停了
  // 但 recording:state 事件没收到 (窗口在后台/race), 聚焦回来立即收敛, 不会卡 "录制中".
  function onWindowFocus() {
    void recordStore.reconcile()
  }

  let unsubscribe: (() => void) | null = null
  let unsubscribeCancelled: (() => void) | null = null
  onMounted(() => {
    void recordStore.reconcile() // 挂载即对账, 修正任何陈旧状态
    window.addEventListener('focus', onWindowFocus)
    unsubscribe = Events.On('recording:completed', async (ev: any) => {
      // 归属守卫: recording:completed 是全局广播, 非本窗发起的录制 → 静默 return.
      // 不弹 toast / 不加节点 / 不关 HUD (那是发起方窗口的事). 状态镜像走 recording:state, 无需本窗插手.
      if (!ownsRecording.value) return
      // 本窗会话到此结束 — 立即清归属, 覆盖下面 error / 无 subgraph 等所有提前 return 分支,
      // 否则残留 true 会让本窗误处理下一次别窗的录制完成事件 (正是要修的多窗口误报根因).
      ownsRecording.value = false
      const raw = ev?.data ?? ev
      const firstArg = Array.isArray(raw) ? raw[0] : raw
      // 状态由后端 'recording:state' 广播 (已收敛到 idle); 这里对账一次兜底防丢事件.
      void recordStore.reconcile()
      try {
        await backend.tools.closeRecordingHUD()
      } catch {
        /* ignore */
      }
      const errMsg = firstArg?.error
      if (errMsg) {
        toast.add({
          title: t('recordComposable.recording_failed'),
          description: String(errMsg),
          color: 'error',
        })
        return
      }
      if (!firstArg?.pendingID) {
        toast.add({ title: t('recordComposable.no_product'), color: 'warning' })
        return
      }
      presentPending({
        pendingID: firstArg.pendingID,
        containerID: firstArg.containerID,
        durationUs: Number(firstArg.durationUs ?? 0),
        eventCount: Number(firstArg.eventCount ?? 0),
      })
    })
    unsubscribeCancelled = Events.On('recording:cancelled', () => {
      ownsRecording.value = false
      replaceNodeID.value = null
      void recordStore.reconcile()
    })
  })
  onUnmounted(() => {
    if (unsubscribe) unsubscribe()
    if (unsubscribeCancelled) unsubscribeCancelled()
    window.removeEventListener('focus', onWindowFocus)
  })

  return {
    countdownSec,
    startRecording,
    stopRecording,
    pendingRecording,
    pendingBusy,
    pendingReplaceMode,
    finalizePending,
    discardPending,
    isRecording: () => recordStore.isRecording,
  }
}
