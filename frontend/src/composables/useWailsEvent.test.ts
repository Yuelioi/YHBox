import { beforeEach, describe, expect, it, vi } from 'vitest'

const { on } = vi.hoisted(() => ({ on: vi.fn() }))

vi.mock('@wailsio/runtime', () => ({ Events: { On: on } }))

import { awaitWailsEvent } from './useWailsEvent'

describe('awaitWailsEvent', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('unsubscribes after the first matching event', async () => {
    const off = vi.fn()
    let handler: ((event: unknown) => void) | undefined
    on.mockImplementation((_name: string, candidate: (event: unknown) => void) => {
      handler = candidate
      return off
    })

    const result = awaitWailsEvent<{ id: string }>('result', (payload) => payload.id === 'mine')
    handler?.({ data: { id: 'other' } })
    expect(off).not.toHaveBeenCalled()
    handler?.({ data: { id: 'mine' } })

    await expect(result).resolves.toEqual({ id: 'mine' })
    expect(off).toHaveBeenCalledOnce()
  })

  it('unsubscribes and rejects when the caller aborts', async () => {
    const off = vi.fn()
    on.mockReturnValue(off)
    const controller = new AbortController()
    const reason = new Error('picker failed to start')

    const result = awaitWailsEvent('result', () => true, controller.signal)
    controller.abort(reason)

    await expect(result).rejects.toBe(reason)
    expect(off).toHaveBeenCalledOnce()
  })
})
