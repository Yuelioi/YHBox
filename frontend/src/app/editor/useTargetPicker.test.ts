import { beforeEach, describe, expect, it, vi } from 'vitest'

const { openScreenPicker, awaitWailsEvent } = vi.hoisted(() => ({
  openScreenPicker: vi.fn(),
  awaitWailsEvent: vi.fn(),
}))

vi.mock('@/lib/backend', () => ({
  backend: { tools: { openScreenPicker, mousePos: vi.fn() } },
}))
vi.mock('@/composables/useWailsEvent', () => ({ awaitWailsEvent }))

import { pickTargetValue, type TargetPoint } from './useTargetPicker'

describe('target picker boundary', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns a successful result and forwards the exact target context', async () => {
    const payload: TargetPoint = {
      x: 10,
      y: 20,
      xRatio: 0.1,
      yRatio: 0.2,
      screenW: 100,
      screenH: 100,
    }
    awaitWailsEvent.mockResolvedValue({ id: 'ignored', payload })

    await expect(pickTargetValue<TargetPoint>('point', 'window-target')).resolves.toEqual(payload)
    expect(openScreenPicker).toHaveBeenCalledWith(
      'point',
      expect.stringMatching(/^authoring-point-/),
      'window-target',
      '',
    )
    expect(awaitWailsEvent.mock.calls[0]?.[2]).toBeInstanceOf(AbortSignal)
  })

  it('treats cancellation as no mutation result', async () => {
    awaitWailsEvent.mockResolvedValue({ payload: { cancelled: true } })

    await expect(pickTargetValue<TargetPoint>('point', 'window-target')).resolves.toBeNull()
  })

  it('propagates picker startup failures for local error feedback', async () => {
    let signal: AbortSignal | undefined
    awaitWailsEvent.mockImplementation(
      (_name: string, _match: (payload: unknown) => boolean, candidate: AbortSignal) => {
        signal = candidate
        return new Promise((_resolve, reject) => {
          candidate.addEventListener('abort', () => reject(candidate.reason), { once: true })
        })
      },
    )
    openScreenPicker.mockRejectedValue(new Error('target unavailable'))

    await expect(pickTargetValue<TargetPoint>('point', 'window-target')).rejects.toThrow(
      'target unavailable',
    )
    expect(signal?.aborted).toBe(true)
  })
})
