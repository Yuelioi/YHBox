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
import { backend, type BlobRef } from '@/lib/backend'
import { i18n } from '@/i18n'

export type RecordingMode = 'simple' | 'precise'

export interface RecordingState {
  revision: number
  phase: 'idle' | 'recording' | 'paused' | 'finalizing' | 'pending'
  mode: RecordingMode | ''
  targetSlot: string
  tempID: string
  startedAtMs: number
  pausedMs: number // 累计已暂停毫秒; HUD 算录制时长 = now-startedAt-pausedMs
  pausedAtMs: number // 本次暂停起点 (>0 即暂停态, HUD 冻结计时); recording 态为 0
  pending: RecordingStopPayload | null
}

export interface RecordingStopPayload {
  pendingID: string
  targetSlot: string
  mode: RecordingMode
  durationUs: number
  eventCount: number
  preview: RecordingPreview
}

export interface RecordingPreview {
  mode: RecordingMode
  durationUs: number
  eventCount: number
  keyActions: number
  clickActions: number
  pointerMoves: number
  rawDeltas: number
  scrollActions: number
  steps: Array<{
    kind: 'keys' | 'click'
    atUs: number
    durationUs: number
    keys?: string[]
    button?: string
    point?: { x: number; y: number; unit: 'ratio' }
  }>
}

export interface RecordingWorkflowDraftNode {
  nodeTypeID: string
  config: Record<string, unknown>
  values: Record<string, unknown>
  blobs: Record<string, BlobRef>
  execInput: string
  execOutput: string
}

export interface RecordingWorkflowDraft {
  mode: RecordingMode
  nodes: RecordingWorkflowDraftNode[]
}

export interface RecordingFinalizePayload {
  clipID: string
  targetSlot: string
  label: string
  draft: RecordingWorkflowDraft
}

export function isRecordingStopPayload(value: unknown): value is RecordingStopPayload {
  if (!isRecord(value) || !isRecord(value.preview) || !Array.isArray(value.preview.steps))
    return false
  return (
    typeof value.pendingID === 'string' &&
    value.pendingID.length > 0 &&
    typeof value.targetSlot === 'string' &&
    (value.mode === 'simple' || value.mode === 'precise') &&
    nonnegativeNumber(value.durationUs) &&
    nonnegativeNumber(value.eventCount) &&
    value.preview.mode === value.mode &&
    nonnegativeNumber(value.preview.durationUs) &&
    nonnegativeNumber(value.preview.eventCount) &&
    nonnegativeNumber(value.preview.keyActions) &&
    nonnegativeNumber(value.preview.clickActions) &&
    nonnegativeNumber(value.preview.pointerMoves) &&
    nonnegativeNumber(value.preview.rawDeltas) &&
    nonnegativeNumber(value.preview.scrollActions)
  )
}

function isRecordingFinalizePayload(value: unknown): value is RecordingFinalizePayload {
  if (!isRecord(value) || !isRecord(value.draft)) return false
  return (
    typeof value.clipID === 'string' &&
    value.clipID.length > 0 &&
    typeof value.targetSlot === 'string' &&
    typeof value.label === 'string' &&
    (value.draft.mode === 'simple' || value.draft.mode === 'precise') &&
    Array.isArray(value.draft.nodes)
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function nonnegativeNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

const IDLE: RecordingState = {
  revision: 0,
  phase: 'idle',
  mode: '',
  targetSlot: '',
  tempID: '',
  startedAtMs: 0,
  pausedMs: 0,
  pausedAtMs: 0,
  pending: null,
}

function normalize(st: any): RecordingState {
  const p = st?.phase
  const pending = isRecordingStopPayload(st?.pending) ? st.pending : null
  return {
    revision: nonnegativeNumber(st?.revision) ? st.revision : 0,
    phase:
      p === 'recording' || p === 'paused' || p === 'finalizing' || p === 'pending' ? p : 'idle',
    mode: st?.mode === 'simple' || st?.mode === 'precise' ? st.mode : '',
    targetSlot: st?.targetSlot ?? '',
    tempID: st?.tempID ?? '',
    startedAtMs: st?.startedAtMs ?? 0,
    pausedMs: st?.pausedMs ?? 0,
    pausedAtMs: st?.pausedAtMs ?? 0,
    pending,
  }
}

export const useRecordingStore = defineStore('recording', () => {
  const state = ref<RecordingState>({ ...IDLE })
  const lastResult = ref<RecordingStopPayload | null>(null)

  // 派生值 — 无独立 flag, 不可能跟后端 desync.
  // isRecording 严格 = phase==='recording' (暂停时 false); 判"录制会话进行中"用 isRecording||isPaused.
  const isRecording = computed(() => state.value.phase === 'recording')
  const isPaused = computed(() => state.value.phase === 'paused')
  // Exact installed target slot remains present while paused/finalizing.
  const activeTargetSlot = computed(() =>
    state.value.phase === 'recording' ||
    state.value.phase === 'paused' ||
    state.value.phase === 'finalizing'
      ? state.value.targetSlot
      : '',
  )

  function applyState(st: any) {
    const next = normalize(st)
    if (next.revision < state.value.revision) return
    state.value = next
    if (next.pending) lastResult.value = next.pending
    else if (next.phase === 'idle') lastResult.value = null
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

  async function start(mode: RecordingMode, targetSlot: string): Promise<void> {
    if (state.value.phase !== 'idle') return
    if (!targetSlot)
      throw new Error('recording.start: targetSlot ' + i18n.global.t('common.required'))
    lastResult.value = null
    await backend.recording.start({ targetSlot, mode })
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
    const result = await backend.recording.stop()
    if (result != null && !isRecordingStopPayload(result))
      throw new Error('recording.stop: invalid result')
    const payload = result ?? null
    lastResult.value = payload
    await reconcile()
    return payload
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
    const result = await backend.recording.finalize(args)
    if (!isRecordingFinalizePayload(result)) throw new Error('recording.finalize: invalid result')
    await reconcile()
    return result
  }

  async function discard(pendingID: string): Promise<void> {
    await backend.recording.discard(pendingID)
    lastResult.value = null
    await reconcile()
  }

  return {
    state,
    lastResult,
    isRecording,
    isPaused,
    activeTargetSlot,
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
