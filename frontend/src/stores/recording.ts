// recording store — 后端录制状态机的【纯镜像】.
//
// 单一真相源是后端 recording.Service（含 armed/countdown/recording 等阶段）. 本 store 不自己存
// 可 desync 的 isRecording flag — state 只由两个来源更新:
//   ① 'recording:state' 事件 (后端每次转换广播全量 state)
//   ② reconcile() 主动调 GetState() 对账 (窗口聚焦 / 编辑器挂载时, 丢事件自愈)
//
// 命令 (start/stop) 幂等: 后端不在录时 stop 返 null 不报错; 本地不乐观置态, 一切以后端为准.
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Events } from '@wailsio/runtime'
import {
  backend,
  type BlobRef,
  type MacroAction,
  type MacroActionKind,
  type MacroDocument,
} from '@/lib/backend'
import { i18n } from '@/i18n'
import { toRPCError, type NormalizedError } from '@/lib/invoke'
import type { WorkflowResource } from '../../../contracts/workflow/current/workflow-source'

export type RecordingMode = 'simple' | 'precise'
export type RecordingInvocation = 'library' | 'editor'

export interface RecordingState {
  revision: number
  phase: 'idle' | 'armed' | 'countdown' | 'recording' | 'paused' | 'finalizing' | 'pending'
  mode: RecordingMode | ''
  targetSlot: string
  tempID: string
  startedAtMs: number
  pausedMs: number // 累计已暂停毫秒; HUD 算录制时长 = now-startedAt-pausedMs
  pausedAtMs: number // 本次暂停起点 (>0 即暂停态, HUD 冻结计时); recording 态为 0
  countdownEndsAtMs: number
  pending: RecordingStopPayload | null
}

export interface RecordingStopPayload {
  pendingID: string
  targetSlot: string
  mode: RecordingMode
  durationUs: number
  eventCount: number
  preview: RecordingPreview
  document?: MacroDocument
  environment: RecordingEnvironment
}

export interface RecordingEnvironment {
  baseResolution: [number, number]
  mouseMode: string
  mouseCounts360: number
}

export type { MacroAction, MacroActionKind }

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
    kind: MacroActionKind | 'move-path'
    atUs: number
    durationUs: number
    key?: string
    button?: string
    point?: { x: number; y: number; unit: 'ratio' }
    notches?: number
    samples?: number
  }>
  tracks: Array<{
    kind: 'keyboard' | 'mouse-buttons' | 'absolute-motion' | 'relative-motion' | 'scroll'
    count: number
    firstUs: number
    lastUs: number
  }>
}

export type RecordingFinalizePayload =
  | {
      destination: 'global-asset'
      targetSlot: string
      asset: {
        guid: string
        kind: 'macro' | 'clip'
        label: string
        blob: BlobRef
      }
    }
  | {
      destination: 'workflow-resource'
      targetSlot: string
      resource: WorkflowResource
    }

export function isRecordingStopPayload(value: unknown): value is RecordingStopPayload {
  if (!isRecord(value) || !isRecord(value.preview) || !Array.isArray(value.preview.steps))
    return false
  const document = value.document
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
    nonnegativeNumber(value.preview.scrollActions) &&
    Array.isArray(value.preview.tracks) &&
    isRecordingEnvironment(value.environment) &&
    (document === undefined || isMacroDocument(document))
  )
}

function normalizeRecordingStopPayload(value: unknown): RecordingStopPayload | null {
  if (!isRecord(value) || !isRecord(value.preview)) return null
  const preview = value.preview
  const normalized = {
    ...value,
    preview: {
      ...preview,
      // Older native snapshots and empty Go slices may arrive as null. Empty
      // means no rows, not an invalid recording envelope.
      steps: preview.steps == null ? [] : preview.steps,
      tracks: preview.tracks == null ? [] : preview.tracks,
    },
  }
  return isRecordingStopPayload(normalized) ? normalized : null
}

function isRecordingEnvironment(value: unknown): value is RecordingEnvironment {
  return (
    isRecord(value) &&
    Array.isArray(value.baseResolution) &&
    value.baseResolution.length === 2 &&
    value.baseResolution.every(positiveNumber) &&
    typeof value.mouseMode === 'string' &&
    nonnegativeNumber(value.mouseCounts360)
  )
}

function isMacroAction(value: unknown): value is MacroAction {
  if (!isRecord(value) || typeof value.id !== 'string' || !value.id) return false
  if (
    ![
      'key-down',
      'key-up',
      'mouse-down',
      'mouse-up',
      'click',
      'move',
      'drag',
      'scroll',
      'sleep',
    ].includes(String(value.kind))
  )
    return false
  if (value.kind === 'key-down' || value.kind === 'key-up')
    return typeof value.key === 'string' && value.key.length > 0
  if (value.kind === 'sleep') return positiveNumber(value.durationUs)
  if (!isRecord(value.point) || value.point.unit !== 'ratio') return false
  if (
    !nonnegativeNumber(value.point.x) ||
    !nonnegativeNumber(value.point.y) ||
    value.point.x > 1 ||
    value.point.y > 1
  )
    return false
  if (value.kind === 'move' || value.kind === 'drag') {
    if (!['instant', 'linear', 'bezier'].includes(String(value.motion))) return false
    if (
      value.motion === 'instant'
        ? value.durationUs !== undefined && value.durationUs !== 0
        : !positiveNumber(value.durationUs)
    )
      return false
    if (value.kind === 'move') return true
    if (!isRecord(value.from) || value.from.unit !== 'ratio') return false
    if (
      !nonnegativeNumber(value.from.x) ||
      !nonnegativeNumber(value.from.y) ||
      value.from.x > 1 ||
      value.from.y > 1
    )
      return false
  }
  if (value.kind === 'scroll')
    return (
      typeof value.notches === 'number' && Number.isInteger(value.notches) && value.notches !== 0
    )
  if (!['left', 'middle', 'right'].includes(String(value.button))) return false
  return value.kind !== 'click' || positiveNumber(value.durationUs)
}

function isMacroDocument(value: unknown): value is MacroDocument {
  if (
    !isRecord(value) ||
    value.schemaVersion !== 2 ||
    !Array.isArray(value.baseResolution) ||
    value.baseResolution.length !== 2 ||
    !value.baseResolution.every(positiveNumber) ||
    !isRecord(value.meta) ||
    !isRecord(value.meta.autoMove) ||
    !Array.isArray(value.actions) ||
    !value.actions.every(isMacroAction)
  )
    return false
  const autoMove = value.meta.autoMove
  if (
    typeof autoMove.enabled !== 'boolean' ||
    !['instant', 'linear', 'bezier'].includes(String(autoMove.mode)) ||
    !nonnegativeNumber(autoMove.durationMs) ||
    autoMove.durationMs > 60_000
  )
    return false
  return autoMove.mode === 'instant' ? autoMove.durationMs === 0 : autoMove.durationMs > 0
}

function isRecordingFinalizePayload(value: unknown): value is RecordingFinalizePayload {
  if (!isRecord(value) || typeof value.targetSlot !== 'string') return false
  if (value.destination === 'workflow-resource') return isWorkflowResource(value.resource)
  if (value.destination !== 'global-asset' || !isRecord(value.asset)) return false
  const asset = value.asset
  return (
    typeof asset.guid === 'string' &&
    asset.guid.length > 0 &&
    (asset.kind === 'macro' || asset.kind === 'clip') &&
    typeof asset.label === 'string' &&
    isBlobRef(asset.blob)
  )
}

function isBlobRef(value: unknown): value is BlobRef {
  return (
    isRecord(value) &&
    typeof value.mediaType === 'string' &&
    typeof value.digest === 'string' &&
    nonnegativeNumber(value.size)
  )
}

function isWorkflowResource(value: unknown): value is WorkflowResource {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.name !== 'string')
    return false
  if (value.kind === 'image')
    return (
      isRecord(value.image) &&
      Array.isArray(value.image.variants) &&
      value.image.variants.length > 0
    )
  if (value.kind === 'macro') return isRecord(value.macro) && isBlobRef(value.macro.blob)
  return value.kind === 'input-clip' && isRecord(value.inputClip) && isBlobRef(value.inputClip.blob)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function eventPayload(event: unknown): unknown {
  if (!isRecord(event)) return event
  const data = event.data
  return Array.isArray(data) && data.length === 1 ? data[0] : (data ?? event)
}

function nonnegativeNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function positiveNumber(value: unknown): value is number {
  return nonnegativeNumber(value) && value > 0
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
  countdownEndsAtMs: 0,
  pending: null,
}

function normalize(st: any): RecordingState {
  const p = st?.phase
  const pending = normalizeRecordingStopPayload(st?.pending)
  return {
    revision: nonnegativeNumber(st?.revision) ? st.revision : 0,
    phase:
      p === 'armed' ||
      p === 'countdown' ||
      p === 'recording' ||
      p === 'paused' ||
      p === 'finalizing' ||
      p === 'pending'
        ? p
        : 'idle',
    mode: st?.mode === 'simple' || st?.mode === 'precise' ? st.mode : '',
    targetSlot: st?.targetSlot ?? '',
    tempID: st?.tempID ?? '',
    startedAtMs: st?.startedAtMs ?? 0,
    pausedMs: st?.pausedMs ?? 0,
    pausedAtMs: st?.pausedAtMs ?? 0,
    countdownEndsAtMs: st?.countdownEndsAtMs ?? 0,
    pending,
  }
}

export const useRecordingStore = defineStore('recording', () => {
  const state = ref<RecordingState>({ ...IDLE })
  const lastResult = ref<RecordingStopPayload | null>(null)
  const invocation = ref<RecordingInvocation | null>(null)
  const completionFailure = ref<{ revision: number; problem: NormalizedError } | null>(null)

  // 派生值 — 无独立 flag, 不可能跟后端 desync.
  // isRecording 严格 = phase==='recording' (暂停时 false); 判"录制会话进行中"用 isRecording||isPaused.
  const isRecording = computed(() => state.value.phase === 'recording')
  const isPaused = computed(() => state.value.phase === 'paused')
  // Exact installed target slot remains present while paused/finalizing.
  const activeTargetSlot = computed(() =>
    state.value.phase === 'armed' ||
    state.value.phase === 'countdown' ||
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
    else if (next.phase === 'idle') {
      lastResult.value = null
      invocation.value = null
    }
  }

  // 后端权威状态广播 — 镜像入口. store 实例化时注册一次 (每窗口一份, 长生命周期不退订).
  Events.On('recording:state', (e: any) => {
    const st = e?.data?.[0] ?? e?.data ?? e
    if (st && typeof st === 'object') applyState(st)
  })

  Events.On('recording:completed', (event: unknown) => {
    const payload = eventPayload(event)
    if (!isRecord(payload)) return
    const projected = isRecord(payload.problem) ? (payload.problem as NormalizedError) : null
    if (!projected?.id) return
    completionFailure.value = {
      revision: (completionFailure.value?.revision ?? 0) + 1,
      problem: projected,
    }
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

  async function start(
    mode: RecordingMode,
    targetSlot: string,
    origin: RecordingInvocation,
  ): Promise<void> {
    if (state.value.phase !== 'idle') return
    if (!targetSlot)
      throw new Error('recording.start: targetSlot ' + i18n.global.t('common.required'))
    lastResult.value = null
    completionFailure.value = null
    invocation.value = origin
    try {
      await backend.recording.start({ targetSlot, mode })
    } catch (error) {
      invocation.value = null
      throw error
    }
    // 不乐观置态 — 后端 Start 成功即广播 recording:state(armed); 这里对账一次兜底.
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
    const payload = result == null ? null : normalizeRecordingStopPayload(result)
    if (result != null && !payload)
      throw toRPCError(
        { id: 'recording.stop.result_invalid', category: 'infrastructure' },
        'recording.stop',
      )
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
    destination: 'global-asset' | 'workflow-resource'
    label: string
    description: string
    category: string
    tags: string[]
    document?: MacroDocument
    trimStartUs?: number
    trimEndUs?: number
  }): Promise<RecordingFinalizePayload> {
    const result = await backend.recording.finalize(args)
    if (!isRecordingFinalizePayload(result))
      throw toRPCError(
        { id: 'recording.finalize.result_invalid', category: 'infrastructure' },
        'recording.finalize',
      )
    await reconcile()
    return result
  }

  async function discard(pendingID: string): Promise<void> {
    await backend.recording.discard(pendingID)
    lastResult.value = null
    await reconcile()
  }

  function claimInvocation(origin: RecordingInvocation): void {
    if (state.value.phase === 'pending' && invocation.value == null) invocation.value = origin
  }

  return {
    state,
    lastResult,
    invocation,
    completionFailure,
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
    claimInvocation,
  }
})
