import { effectScope } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => {
  const order: string[] = []
  return {
    order,
    emit: vi.fn(),
    validateTarget: vi.fn(async () => {
      order.push('validate')
    }),
    openHUD: vi.fn(async () => {
      order.push('open-hud')
    }),
    closeHUD: vi.fn(async () => {
      order.push('close-hud')
    }),
    reloadHotkeys: vi.fn(async () => {
      order.push('reload-hotkeys')
    }),
    startRecording: vi.fn(async () => {
      order.push('start-recording')
    }),
  }
})

vi.mock('@wailsio/runtime', () => ({ Events: { Emit: mocks.emit } }))
vi.mock('@/lib/backend', () => ({
  backend: {
    recording: { validateTarget: mocks.validateTarget },
    tools: { openRecordingHUD: mocks.openHUD, closeRecordingHUD: mocks.closeHUD },
  },
}))
vi.mock('@/stores/hotkeys', () => ({
  useHotkeysStore: () => ({
    list: [
      { key: 'recording.stop', hotkeyStr: 'F12' },
      { key: 'recording.pause', hotkeyStr: 'F11' },
    ],
    reload: mocks.reloadHotkeys,
  }),
}))
vi.mock('@/stores/recording', () => ({
  useRecordingStore: () => ({
    state: { phase: 'idle' },
    start: mocks.startRecording,
  }),
}))

import { useRecordingStart } from './useRecordingStart'

describe('useRecordingStart', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mocks.order.length = 0
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('validates, opens the HUD, counts down, then starts the selected explicit mode', async () => {
    const scope = effectScope()
    const controller = scope.run(() => useRecordingStart())!
    const result = controller.start('precise', 'game')
    await vi.runAllTimersAsync()

    await expect(result).resolves.toBe(true)
    expect(mocks.order).toEqual(['validate', 'open-hud', 'reload-hotkeys', 'start-recording'])
    expect(mocks.startRecording).toHaveBeenCalledWith('precise', 'game')
    expect(mocks.emit.mock.calls).toEqual([
      ['recording:countdown', { sec: 3, mode: 'precise', stopKey: 'F12', pauseKey: 'F11' }],
      ['recording:countdown', { sec: 2, mode: 'precise', stopKey: 'F12', pauseKey: 'F11' }],
      ['recording:countdown', { sec: 1, mode: 'precise', stopKey: 'F12', pauseKey: 'F11' }],
    ])
    scope.stop()
  })

  it('does not open the HUD when target validation fails', async () => {
    mocks.validateTarget.mockRejectedValueOnce(new Error('unavailable'))
    const scope = effectScope()
    const controller = scope.run(() => useRecordingStart())!

    await expect(controller.start('simple', 'game')).rejects.toThrow('unavailable')
    expect(mocks.openHUD).not.toHaveBeenCalled()
    expect(mocks.startRecording).not.toHaveBeenCalled()
    expect(mocks.closeHUD).toHaveBeenCalledTimes(1)
    scope.stop()
  })
})
