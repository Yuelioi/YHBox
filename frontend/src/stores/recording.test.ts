import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// recordStore 依赖 backend RPC + i18n — 都 mock 掉, 只验 activeTargetContainerID 生命周期 (A1).
vi.mock('@/lib/backend', () => ({
  backend: {
    recording: {
      start: vi.fn(async () => 'temp-123'),
      stop: vi.fn(async () => ({
        subgraphID: 'sg-x',
        containerID: 'cA',
        label: 'rec',
        filterMode: 'precise',
      })),
    },
  },
}))
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k } } }))

import { useRecordingStore } from './recording'

describe('recordStore activeTargetContainerID 生命周期 (A1 单一来源)', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('start 锁定 target; stop 清空', async () => {
    const s = useRecordingStore()
    expect(s.activeTargetContainerID).toBe('')

    await s.start('precise', 'cA')
    expect(s.isRecording).toBe(true)
    expect(s.activeTargetContainerID).toBe('cA')

    await s.stop()
    expect(s.isRecording).toBe(false)
    expect(s.activeTargetContainerID).toBe('') // stop() finally 清
  })

  it('markStopped 清 isRecording + target (F12/HUD 异步停录路径, 不经 stop RPC)', async () => {
    const s = useRecordingStore()
    await s.start('precise', 'cB')
    expect(s.activeTargetContainerID).toBe('cB')

    s.markStopped()
    expect(s.isRecording).toBe(false)
    expect(s.activeTargetContainerID).toBe('')
  })
})
