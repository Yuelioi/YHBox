// recording store — 后端录制状态机的【纯镜像】.
//
// 单一真相源是后端 recording.Service (Phase: idle|recording|finalizing). 本 store 不自己存
// 可 desync 的 isRecording flag — state 只由两个来源更新:
//   ① 'recording:state' 事件 (后端每次转换广播全量 state)
//   ② reconcile() 主动调 GetState() 对账 (窗口聚焦 / 编辑器挂载时, 丢事件自愈)
//
// 命令 (start/stop) 幂等: 后端不在录时 stop 返 null 不报错; 本地不乐观置态, 一切以后端为准.
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { i18n } from '@/i18n'

export interface RecordingState {
  phase: 'idle' | 'recording' | 'paused' | 'finalizing'
  containerID: string
  filterMode: string
  tempID: string
  startedAtMs: number
  pausedMs: number // 累计已暂停毫秒; HUD 算录制时长 = now-startedAt-pausedMs
  pausedAtMs: number // 本次暂停起点 (>0 即暂停态, HUD 冻结计时); recording 态为 0
}

export interface RecordingStopPayload {
  pendingID: string
  containerID: string
  filterMode: string
  durationUs: number
  eventCount: number
}

export interface RecordingFinalizePayload {
  subgraphID: string
  clipID: string
  containerID: string
  label: string
  filterMode: string
}

const IDLE: RecordingState = {
  phase: 'idle',
  containerID: '',
  filterMode: '',
  tempID: '',
  startedAtMs: 0,
  pausedMs: 0,
  pausedAtMs: 0,
}

function normalize(st: any): RecordingState {
  const p = st?.phase
  return {
    phase: p === 'recording' || p === 'paused' || p === 'finalizing' ? p : 'idle',
    containerID: st?.containerID ?? '',
    filterMode: st?.filterMode ?? '',
    tempID: st?.tempID ?? '',
    startedAtMs: st?.startedAtMs ?? 0,
    pausedMs: st?.pausedMs ?? 0,
    pausedAtMs: st?.pausedAtMs ?? 0,
  }
}

export const useRecordingStore = defineStore('recording', () => {
  const state = ref<RecordingState>({ ...IDLE })
  const lastResult = ref<RecordingStopPayload | null>(null)

  // 派生值 — 无独立 flag, 不可能跟后端 desync.
  // isRecording 严格 = phase==='recording' (暂停时 false); 判"录制会话进行中"用 isRecording||isPaused.
  const isRecording = computed(() => state.value.phase === 'recording')
  const isPaused = computed(() => state.value.phase === 'paused')
  // 录制目标容器 (A2 删除守卫 / A3 离开确认 / A4 指示器 读它). idle 时为空 (paused 仍有目标).
  const activeTargetContainerID = computed(() =>
    state.value.phase === 'idle' ? '' : state.value.containerID,
  )

  function applyState(st: any) {
    state.value = normalize(st)
  }

  // 后端权威状态广播 — 镜像入口. store 实例化时注册一次 (每窗口一份, 长生命周期不退订).
  Events.On('recording:state', (e: any) => {
    const st = e?.data?.[0] ?? e?.data ?? e
    if (st && typeof st === 'object') applyState(st)
  })

  // reconcile 主动跟后端对账. 窗口聚焦 / 编辑器挂载时调 — 任何 recording:state 事件丢失,
  // 下次对账自动收敛, 卡死状态自愈.
  async function reconcile() {
    try {
      const st = await backend.recording.getState()
      if (st) applyState(st)
    } catch (e) {
      console.warn('recording reconcile failed', e)
    }
  }

  async function start(filterMode: 'precise' | 'simple', containerID: string): Promise<void> {
    if (isRecording.value) return
    if (!containerID)
      throw new Error('recording.start: containerID ' + i18n.global.t('common.required'))
    lastResult.value = null
    await backend.recording.start({ filterMode, containerID })
    // 不乐观置态 — 后端 Start 成功即广播 recording:state(recording); 这里对账一次兜底.
    await reconcile()
  }

  // pause/resume: 后端幂等 (非对应 phase no-op). 不乐观置态 — 后端广播 recording:state 收敛.
  async function pause(): Promise<void> {
    if (!isRecording.value) return
    await backend.recording.pause()
    await reconcile()
  }
  async function resume(): Promise<void> {
    if (!isPaused.value) return
    await backend.recording.resume()
    await reconcile()
  }

  async function stop(): Promise<RecordingStopPayload | null> {
    // 幂等: 后端不在录 → 返 null 不抛错. 拿到产物 (或 null) 后对账收敛状态.
    const payload = (await backend.recording.stop()) as RecordingStopPayload | null | undefined
    lastResult.value = payload ?? null
    await reconcile()
    return payload ?? null
  }

  async function cancel(): Promise<void> {
    await backend.recording.cancel()
    lastResult.value = null
    await reconcile()
  }

  async function finalize(args: {
    pendingID: string
    label: string
    description: string
    category: string
    tags: string[]
  }): Promise<RecordingFinalizePayload> {
    const payload = (await backend.recording.finalize(args)) as
      | RecordingFinalizePayload
      | null
      | undefined
    if (!payload) throw new Error('recording.finalize: empty result')
    return payload
  }

  async function discard(pendingID: string): Promise<void> {
    await backend.recording.discard(pendingID)
    lastResult.value = null
  }

  return {
    state,
    lastResult,
    isRecording,
    isPaused,
    activeTargetContainerID,
    applyState,
    reconcile,
    start,
    pause,
    resume,
    stop,
    cancel,
    finalize,
    discard,
  }
})
