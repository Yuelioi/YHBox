import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// recordStore 是后端状态机的纯镜像. 验: applyState 镜像 / isRecording 派生 /
// activeTargetSlot 派生 / reconcile 走 getState 对账.
// vi.hoisted: vi.mock 工厂被提升到文件顶, 引用的 mock 必须也提升, 否则 ReferenceError.
const { getStateMock, finalizeMock } = vi.hoisted(() => ({
  getStateMock: vi.fn(async () => ({
    phase: 'recording',
    targetSlot: 'editor',
    tempID: 't1',
    startedAtMs: 123,
  })),
  finalizeMock: vi.fn(),
}))
const { pauseMock, resumeMock } = vi.hoisted(() => ({
  pauseMock: vi.fn(async () => {}),
  resumeMock: vi.fn(async () => {}),
}))
vi.mock('@/lib/backend', () => ({
  backend: {
    recording: {
      getState: getStateMock,
      start: vi.fn(),
      stop: vi.fn(),
      pause: pauseMock,
      resume: resumeMock,
      finalize: finalizeMock,
    },
  },
}))
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k } } }))
// store 体内 Events.On('recording:state') 订阅后端广播 — 测试里 stub 掉, 只验镜像逻辑.
vi.mock('@wailsio/runtime', () => ({ Events: { On: vi.fn(() => () => {}) } }))

import { isRecordingStopPayload, useRecordingStore } from './recording'

describe('recordStore — 后端状态机镜像', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('accepts the destination-tagged Workflow Resource finalize result', async () => {
    finalizeMock.mockResolvedValueOnce({
      destination: 'workflow-resource',
      targetSlot: 'editor',
      resource: {
        id: 'macro-local',
        kind: 'macro',
        name: 'Local macro',
        macro: {
          blob: {
            mediaType: 'application/vnd.yotta.macro+json',
            digest: `sha256:${'a'.repeat(64)}`,
            size: 42,
          },
          baseResolution: [1280, 720],
          actionCount: 1,
          durationUs: 25_000,
        },
      },
    })
    const store = useRecordingStore()
    const result = await store.finalize({
      pendingID: 'pending-local',
      destination: 'workflow-resource',
      label: 'Local macro',
      description: '',
      category: '',
      tags: [],
    })
    expect(result.destination).toBe('workflow-resource')
    expect(finalizeMock).toHaveBeenCalledWith(
      expect.objectContaining({ destination: 'workflow-resource' }),
    )
  })

  it('applyState(recording) mirrors the exact installed target slot', () => {
    const s = useRecordingStore()
    expect(s.isRecording).toBe(false) // 初始 idle
    expect(s.activeTargetSlot).toBe('')

    s.applyState({
      phase: 'recording',
      targetSlot: 'editor',
      tempID: 'x',
      startedAtMs: 1,
    })
    expect(s.isRecording).toBe(true)
    expect(s.activeTargetSlot).toBe('editor')
  })

  it('applyState(idle) → isRecording false, target 清空 (镜像收敛)', () => {
    const s = useRecordingStore()
    s.applyState({ phase: 'recording', targetSlot: 'editor' })
    s.applyState({ phase: 'idle' })
    expect(s.isRecording).toBe(false)
    expect(s.activeTargetSlot).toBe('')
  })

  it('armed/countdown keep the target visible without claiming capture has started', () => {
    const s = useRecordingStore()
    s.applyState({ phase: 'armed', targetSlot: 'editor' })
    expect(s.isRecording).toBe(false)
    expect(s.activeTargetSlot).toBe('editor')

    s.applyState({ phase: 'countdown', targetSlot: 'editor', countdownEndsAtMs: 4_000 })
    expect(s.isRecording).toBe(false)
    expect(s.activeTargetSlot).toBe('editor')
    expect(s.state.countdownEndsAtMs).toBe(4_000)
  })

  it('finalizing 阶段不算 recording, 但 target 仍可见 (收尾期)', () => {
    const s = useRecordingStore()
    s.applyState({ phase: 'finalizing', targetSlot: 'editor' })
    expect(s.isRecording).toBe(false)
    expect(s.activeTargetSlot).toBe('editor')
  })

  it('applyState(paused) → isPaused 派生 true, isRecording false, target 仍可见', () => {
    const s = useRecordingStore()
    s.applyState({
      phase: 'paused',
      targetSlot: 'editor',
      startedAtMs: 1,
      pausedMs: 500,
      pausedAtMs: 2000,
    })
    expect(s.isPaused).toBe(true)
    expect(s.isRecording).toBe(false) // 暂停时严格 false (会话进行中判 isRecording||isPaused)
    expect(s.activeTargetSlot).toBe('editor')
    expect(s.state.pausedMs).toBe(500)
    expect(s.state.pausedAtMs).toBe(2000)
  })

  it('pause() 仅 recording 态调后端 RPC; resume() 仅 paused 态调', async () => {
    const s = useRecordingStore()
    // recording → pause 调 RPC; resume 此时 no-op (非 paused).
    s.applyState({ phase: 'recording', targetSlot: 'editor' })
    await s.resume()
    expect(resumeMock).not.toHaveBeenCalled()
    await s.pause()
    expect(pauseMock).toHaveBeenCalledTimes(1)
    // paused → resume 调 RPC; pause no-op.
    s.applyState({ phase: 'paused', targetSlot: 'editor' })
    await s.pause()
    expect(pauseMock).toHaveBeenCalledTimes(1) // 没再调
    await s.resume()
    expect(resumeMock).toHaveBeenCalledTimes(1)
  })

  it('reconcile() 调 getState 对账 → 镜像后端权威状态 (丢事件自愈)', async () => {
    const s = useRecordingStore()
    expect(s.isRecording).toBe(false)
    await s.reconcile()
    expect(getStateMock).toHaveBeenCalled()
    expect(s.isRecording).toBe(true)
    expect(s.activeTargetSlot).toBe('editor')
  })

  it('accepts only completed events that include the recording preview contract', () => {
    expect(
      isRecordingStopPayload({
        pendingID: 'pending-session',
        targetSlot: 'editor',
        mode: 'simple',
        durationUs: 25_000,
        eventCount: 2,
        preview: {
          mode: 'simple',
          durationUs: 25_000,
          eventCount: 2,
          keyActions: 1,
          clickActions: 0,
          pointerMoves: 0,
          rawDeltas: 0,
          scrollActions: 0,
          steps: [],
          tracks: [],
        },
        environment: {
          baseResolution: [1280, 720],
          mouseMode: 'absolute',
          mouseCounts360: 0,
        },
      }),
    ).toBe(true)
    expect(
      isRecordingStopPayload({
        pendingID: 'pending-session',
        targetSlot: 'editor',
        durationUs: 25_000,
        eventCount: 2,
      }),
    ).toBe(false)
  })

  it('ignores stale snapshots and restores an authoritative pending result', () => {
    const s = useRecordingStore()
    const pending = {
      pendingID: 'pending-session',
      targetSlot: 'editor',
      mode: 'precise' as const,
      durationUs: 25_000,
      eventCount: 2,
      preview: {
        mode: 'precise' as const,
        durationUs: 25_000,
        eventCount: 2,
        keyActions: 0,
        clickActions: 0,
        pointerMoves: 1,
        rawDeltas: 0,
        scrollActions: 0,
        steps: [],
        tracks: [{ kind: 'absolute-motion' as const, count: 1, firstUs: 0, lastUs: 25_000 }],
      },
      environment: {
        baseResolution: [1280, 720] as [number, number],
        mouseMode: 'absolute',
        mouseCounts360: 0,
      },
    }
    s.applyState({ revision: 7, phase: 'pending', mode: 'precise', pending })
    s.applyState({ revision: 6, phase: 'idle' })
    expect(s.state.phase).toBe('pending')
    expect(s.lastResult?.pendingID).toBe('pending-session')
  })

  it('normalizes nullable preview collections from a native pending snapshot', () => {
    const s = useRecordingStore()
    s.applyState({
      revision: 8,
      phase: 'pending',
      mode: 'simple',
      pending: {
        pendingID: 'pending-native-session',
        targetSlot: 'window-target',
        mode: 'simple',
        durationUs: 390_000,
        eventCount: 6,
        preview: {
          mode: 'simple',
          durationUs: 390_000,
          eventCount: 6,
          keyActions: 6,
          clickActions: 0,
          pointerMoves: 0,
          rawDeltas: 0,
          scrollActions: 0,
          steps: [],
          tracks: null,
        },
        actions: [{ id: 'action-1', kind: 'key-down', key: 'W' }],
        environment: {
          baseResolution: [1280, 720],
          mouseMode: 'relative',
          mouseCounts360: 0,
        },
      },
    })

    expect(s.state.phase).toBe('pending')
    expect(s.lastResult?.pendingID).toBe('pending-native-session')
    expect(s.lastResult?.preview.tracks).toEqual([])
  })
})
