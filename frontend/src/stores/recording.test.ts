import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// recordStore 是后端状态机的纯镜像. 验: applyState 镜像 / isRecording 派生 /
// activeTargetContainerID 派生 / reconcile 走 getState 对账.
// vi.hoisted: vi.mock 工厂被提升到文件顶, 引用的 mock 必须也提升, 否则 ReferenceError.
const { getStateMock } = vi.hoisted(() => ({
  getStateMock: vi.fn(async () => ({
    phase: 'recording',
    containerID: 'cReconciled',
    tempID: 't1',
    startedAtMs: 123,
  })),
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
    },
  },
}))
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k } } }))
// store 体内 Events.On('recording:state') 订阅后端广播 — 测试里 stub 掉, 只验镜像逻辑.
vi.mock('@wailsio/runtime', () => ({ Events: { On: vi.fn(() => () => {}) } }))

import { useRecordingStore } from './recording'

describe('recordStore — 后端状态机镜像', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('applyState(recording) → isRecording 派生 true, activeTargetContainerID 派生容器', () => {
    const s = useRecordingStore()
    expect(s.isRecording).toBe(false) // 初始 idle
    expect(s.activeTargetContainerID).toBe('')

    s.applyState({
      phase: 'recording',
      containerID: 'cA',
      tempID: 'x',
      startedAtMs: 1,
    })
    expect(s.isRecording).toBe(true)
    expect(s.activeTargetContainerID).toBe('cA')
  })

  it('applyState(idle) → isRecording false, target 清空 (镜像收敛)', () => {
    const s = useRecordingStore()
    s.applyState({ phase: 'recording', containerID: 'cA' })
    s.applyState({ phase: 'idle' })
    expect(s.isRecording).toBe(false)
    expect(s.activeTargetContainerID).toBe('')
  })

  it('finalizing 阶段不算 recording, 但 target 仍可见 (收尾期)', () => {
    const s = useRecordingStore()
    s.applyState({ phase: 'finalizing', containerID: 'cA' })
    expect(s.isRecording).toBe(false)
    expect(s.activeTargetContainerID).toBe('cA')
  })

  it('applyState(paused) → isPaused 派生 true, isRecording false, target 仍可见', () => {
    const s = useRecordingStore()
    s.applyState({
      phase: 'paused',
      containerID: 'cA',
      startedAtMs: 1,
      pausedMs: 500,
      pausedAtMs: 2000,
    })
    expect(s.isPaused).toBe(true)
    expect(s.isRecording).toBe(false) // 暂停时严格 false (会话进行中判 isRecording||isPaused)
    expect(s.activeTargetContainerID).toBe('cA')
    expect(s.state.pausedMs).toBe(500)
    expect(s.state.pausedAtMs).toBe(2000)
  })

  it('pause() 仅 recording 态调后端 RPC; resume() 仅 paused 态调', async () => {
    const s = useRecordingStore()
    // recording → pause 调 RPC; resume 此时 no-op (非 paused).
    s.applyState({ phase: 'recording', containerID: 'cA' })
    await s.resume()
    expect(resumeMock).not.toHaveBeenCalled()
    await s.pause()
    expect(pauseMock).toHaveBeenCalledTimes(1)
    // paused → resume 调 RPC; pause no-op.
    s.applyState({ phase: 'paused', containerID: 'cA' })
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
    expect(s.activeTargetContainerID).toBe('cReconciled')
  })
})
