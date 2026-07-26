import { effectScope } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => {
  const order: string[] = []
  return {
    order,
    validateTarget: vi.fn(async () => {
      order.push('validate')
    }),
    openHUD: vi.fn(async () => {
      order.push('open-hud')
    }),
    closeHUD: vi.fn(async () => {
      order.push('close-hud')
    }),
    startRecording: vi.fn(async () => {
      order.push('start-recording')
    }),
  }
})

vi.mock('@/lib/backend', () => ({
  backend: {
    recording: { validateTarget: mocks.validateTarget },
    tools: { openRecordingHUD: mocks.openHUD, closeRecordingHUD: mocks.closeHUD },
  },
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
    mocks.order.length = 0
    vi.clearAllMocks()
  })

  it('validates, opens the HUD, then arms the selected mode without a frontend countdown', async () => {
    const scope = effectScope()
    const controller = scope.run(() => useRecordingStart())!
    const result = controller.start('precise', 'game', 'editor')

    await expect(result).resolves.toBe(true)
    expect(mocks.order).toEqual(['validate', 'open-hud', 'start-recording'])
    expect(mocks.startRecording).toHaveBeenCalledWith('precise', 'game', 'editor')
    scope.stop()
  })

  it('does not open the HUD when target validation fails', async () => {
    mocks.validateTarget.mockRejectedValueOnce(new Error('unavailable'))
    const scope = effectScope()
    const controller = scope.run(() => useRecordingStart())!

    await expect(controller.start('simple', 'game', 'library')).rejects.toThrow('unavailable')
    expect(mocks.openHUD).not.toHaveBeenCalled()
    expect(mocks.startRecording).not.toHaveBeenCalled()
    expect(mocks.closeHUD).toHaveBeenCalledTimes(1)
    scope.stop()
  })
})
