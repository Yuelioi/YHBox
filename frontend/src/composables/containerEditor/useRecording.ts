// useRecording 录制流程:
//   1. 倒计时 → recordStore.start(filterMode, containerID)  (hook 已挂, F12 / HUD / toolbar 都能停)
//   2. 停录路径:
//       - toolbar 停: stopRecording() → recordStore.stop() 同步拿 payload
//       - F12 / HUD 停: 后端 emit 'recording:completed' {subgraphID|clipID, containerID, label, filterMode} | {error}
//   3. onRecordingProduct 按 filterMode 分叉落产物节点:
//       - simple: refreshSubgraphStore() 让 editorStore.subgraphsForCurrentContainer 拿到新 subgraph
//         (activeGraph computed 从这里读, 不读 draft.subgraphs!) → 加 Subgraph 调用节点 (config.SubgraphID)。
//       - precise: 直接加一个裸 PlayClip 节点 (config.ClipID), 无新子图, 不必刷 store。
//      两路都落在当前视口中心、不自动连线、落下即选中 → 自动 saveDraft (用户自己接线)。
//
// 关键陷阱 (simple): container.Subgraphs 在 Go 端是 json:"-", backend.containers.get(id) 拿不到 subgraphs.
// 子图列表走单独 RPC → 灌进 editorStore. 录完不 refresh store 的话, 用户点新 Subgraph 节点进入会拿到 null → 画布空白.
//
// 精准 vs 简易差别:
//   - 录制层: simple 模式 drainLoop 丢 RawDelta/MouseMove (省内存)
//   - 产物: simple → 多个 KeyPress/ClickAt/Sleep 节点打包成子图; precise → 一个 PlayClip 节点 (回放原始录像)
//   - 画布: simple 是 Subgraph 调用节点 (可进入), precise 是裸 PlayClip 节点 (所见即所得)。
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Ref, ComputedRef } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Container, type Graph } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useRecordingStore, type RecordingStopPayload } from '@/stores/recording'
import { useHotkeysStore } from '@/stores/hotkeys'
import { randID } from './ids'

export interface RecordOpts {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  syncFlowFromDraft: () => void
  // refreshSubgraphStore: 必传 — 录完后调一次, 否则 editorStore 没新 subgraph, 双击进入显示空白.
  refreshSubgraphStore: () => Promise<void> | void
  saveDraft: () => Promise<unknown> | unknown
  // dropPoint: 录制产物节点的落点 (flow 坐标). 取当前视口中心 — 录完节点出现在用户正看的地方.
  dropPoint: () => { x: number; y: number }
  // selectNode: 落下后选中该节点 (用户接着自己接线).
  selectNode: (id: string) => void
  toast: { add: (o: Record<string, unknown>) => unknown }
}

export interface StartRecordingOpts {
  // replaceNodeID: NodeInspector 点 "重新录制覆盖" 时传入. 录完不创建新节点, 而是改写目标节点的引用
  // (simple 产物 → 目标应为 Subgraph 节点, 改 config.SubgraphID; precise 产物 → 目标应为 PlayClip,
  // 改 config.ClipID). kind 不匹配则退化成新建. 旧子图/clip 变孤儿, 用户可手动清.
  replaceNodeID?: string
}

export function useRecording(opts: RecordOpts) {
  const { draft, activeGraph, syncFlowFromDraft, refreshSubgraphStore, saveDraft, dropPoint, selectNode, toast } = opts
  const recordStore = useRecordingStore()
  const hotkeysStore = useHotkeysStore()
  const { t } = useI18n()

  const countdownSec = ref(0)
  const filterMode = ref<'precise' | 'simple'>('precise')
  const replaceNodeID = ref<string | null>(null)
  // 归属化: 仅本窗发起的录制才处理 recording:completed. recording:completed 是全局广播,
  // 多个容器编辑器窗口都订阅 — 非发起方窗口若也处理会误报 container_mismatch (子图存对了, 别的窗口瞎报警).
  const ownsRecording = ref(false)

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

    // 倒计时前预检窗口 (① 没设/找不到窗口立刻报错, 不用等录完; ④ 顺带把游戏拉到前台).
    // 失败 → 不开 HUD 不进倒计时. Start 内仍有同样校验作 race 兜底.
    try {
      await backend.recording.validateTarget(containerID)
    } catch (e: any) {
      toast.add({ title: t('recording.launch_failed'), description: errorMessage(e), color: 'error' })
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
      Events.Emit('recording:countdown', { sec: i, mode, stopKey, pauseKey })
      await new Promise((r) => setTimeout(r, 1000))
      if (countdownSec.value === 0) {
        try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
        Events.Emit('recording:countdown', { sec: 0, mode, stopKey, pauseKey })
        return
      }
    }
    countdownSec.value = 0

    try {
      await recordStore.start(mode, containerID)
      ownsRecording.value = true // 本窗发起 — 后续 recording:completed 由本窗处理
      toast.add({
        title: t('recordComposable.recording_in_progress', { mode: mode === 'precise' ? t('recordComposable.mode_precise') : t('recordComposable.mode_simple') }),
        description: t('recordComposable.stop_methods', { hk: hotkeysStore.keyFor('recording.stop', 'F12') }),
        color: 'success',
        duration: 5000,
      })
    } catch (e: any) {
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      toast.add({ title: t('recording.launch_failed'), description: errorMessage(e), color: 'error' })
    }
  }

  // stopRecording: toolbar / 主窗口主动停 — 同步路径, recordStore.stop() 拿 payload.
  async function stopRecording() {
    // recording 或 paused 都可停 (paused 直接停, 后端守卫已放开).
    if (!recordStore.isRecording && !recordStore.isPaused) return
    try {
      const payload = await recordStore.stop()
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      if (!payload || (!payload.subgraphID && !payload.clipID)) {
        toast.add({ title: t('recording.no_steps'), color: 'warning' })
        return
      }
      await onRecordingProduct(payload)
    } catch (e: any) {
      toast.add({ title: t('recording.stop_failed'), description: errorMessage(e), color: 'error' })
    }
  }

  // onRecordingProduct: 录完产物落地. 按 filterMode 分叉 —
  //   simple  → 产物是子图 (后端已落盘), 加一个 Subgraph 调用节点;
  //   precise → 产物是 clip (后端已落盘), 加一个裸 PlayClip 节点 (config.ClipID)。
  //
  // ⚠ simple 必须先 refreshSubgraphStore — activeGraph computed 从 editorStore 读子图列表,
  // 不刷新的话双击进 Subgraph 节点会拿不到, 画布空白 (实测踩过). precise 无新子图, 跳过.
  async function onRecordingProduct(payload: RecordingStopPayload) {
    // 会话结束 (无论哪条停录路径都汇流到这) → 清归属标记.
    ownsRecording.value = false
    const isPrecise = payload.filterMode === 'precise'

    if (!activeGraph.value || !draft.value) {
      toast.add({
        title: t('recording.completed_no_graph'),
        description: isPrecise ? `clipID=${payload.clipID}` : `subgraphID=${payload.subgraphID}`,
        color: 'primary',
      })
      // simple 仍刷一下 editorStore 让其他视图能看到新子图.
      if (!isPrecise) { try { await refreshSubgraphStore() } catch { /* ignore */ } }
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
      } catch { /* fallback 裸 ID */ }
      toast.add({
        title: t('recordComposable.container_mismatch', {
          target: targetName,
          current: draft.value.name || draft.value.id,
        }),
        color: 'warning',
        duration: 8000,
      })
      if (!isPrecise) { try { await refreshSubgraphStore() } catch { /* ignore */ } }
      return
    }

    // 1) simple 刷新 editorStore 让 subgraphsForCurrentContainer 拿到新 subgraph (双击进入用).
    //    precise (裸 PlayClip) 无新子图, 跳过.
    if (!isPrecise) {
      try {
        await refreshSubgraphStore()
      } catch (e: any) {
        toast.add({ title: t('recordComposable.refresh_subgraphs_failed'), description: errorMessage(e), color: 'error' })
        return
      }
    }

    // 2) 处理 replaceNodeID — 重新录制覆盖目标节点的引用 (simple→SubgraphID, precise→ClipID).
    const replaceID = replaceNodeID.value
    replaceNodeID.value = null

    if (replaceID) {
      const target = (activeGraph.value.nodes as any[]).find((n) => n.id === replaceID)
      const wantKind = isPrecise ? 'PlayClip' : 'Subgraph'
      if (!target) {
        toast.add({ title: t('recordComposable.replace_node_missing'), color: 'warning' })
      } else if (target.kind !== wantKind) {
        toast.add({
          title: t('recordComposable.replace_node_wrong_kind', { kind: target.kind }),
          color: 'warning',
        })
      } else {
        if (!target.config) target.config = {}
        if (isPrecise) target.config.ClipID = payload.clipID
        else target.config.SubgraphID = payload.subgraphID
        syncFlowFromDraft()
        await maybeSave()
        toast.add({ title: t('recording.rerecord_overwrite', { name: payload.label }), color: 'success' })
        return
      }
    }

    // 3) 新建产物节点 — 落在当前视口中心, 不自动连线, 落下即选中 (用户自己接线).
    const nodeId = randID(isPrecise ? 'n-clip' : 'n-sg')
    const pt = dropPoint()
    const newNode = {
      id: nodeId,
      kind: isPrecise ? 'PlayClip' : 'Subgraph',
      // 节点宽 ~240 / 高 ~90, 落点减半个身位让节点视觉居中, 而非左上角压在中心.
      x: pt.x - 120,
      y: pt.y - 45,
      config: isPrecise ? { ClipID: payload.clipID } : { SubgraphID: payload.subgraphID },
      createdAt: new Date().toISOString(),
    }
    ;(activeGraph.value.nodes as any[]).push(newNode)
    syncFlowFromDraft()
    await maybeSave()
    selectNode(nodeId)
    toast.add({
      title: t(isPrecise ? 'recording.added_clip' : 'recording.added_subgraph', { name: payload.label }),
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

  // 订阅 'recording:completed' (F12 全局热键 / HUD 关 / StopAsync 触发).
  // payload: {subgraphID, containerID, label, filterMode} 或 {error}. (状态本身走 recording:state.)
  let unsubscribe: (() => void) | null = null
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
      try { await backend.tools.closeRecordingHUD() } catch { /* ignore */ }
      const errMsg = firstArg?.error
      if (errMsg) {
        toast.add({ title: t('recordComposable.recording_failed'), description: String(errMsg), color: 'error' })
        return
      }
      const hasProduct = firstArg?.subgraphID || firstArg?.clipID
      if (!hasProduct) {
        toast.add({ title: t('recordComposable.no_product'), color: 'warning' })
        return
      }
      await onRecordingProduct({
        subgraphID: firstArg.subgraphID ?? '',
        clipID: firstArg.clipID ?? '',
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
