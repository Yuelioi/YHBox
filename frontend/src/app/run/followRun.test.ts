import { describe, expect, it, vi } from 'vitest'
import { pollTerminalRunStatus } from './followRun'

describe('short Run reconciliation', () => {
  it('polls past a stale running snapshot until the terminal record is visible', async () => {
    const readStatus = vi
      .fn<() => Promise<string>>()
      .mockResolvedValueOnce('running')
      .mockResolvedValueOnce('running')
      .mockResolvedValueOnce('succeeded')
    const wait = vi.fn(async () => undefined)

    await expect(
      pollTerminalRunStatus(readStatus, () => false, { attempts: 4, wait }),
    ).resolves.toBe('succeeded')
    expect(readStatus).toHaveBeenCalledTimes(3)
    expect(wait).toHaveBeenCalledTimes(2)
  })

  it('stops polling when another launcher request replaces the observed Run', async () => {
    let stopped = false
    const readStatus = vi.fn(async () => {
      stopped = true
      return 'running'
    })

    await expect(
      pollTerminalRunStatus(readStatus, () => stopped, {
        attempts: 4,
        wait: async () => undefined,
      }),
    ).resolves.toBeUndefined()
    expect(readStatus).toHaveBeenCalledTimes(1)
  })
})
