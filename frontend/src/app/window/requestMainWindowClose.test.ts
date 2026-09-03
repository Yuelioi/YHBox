import { afterEach, describe, expect, it, vi } from 'vitest'
import { registerMainWindowCloseGuard } from './mainWindowCloseGuard'
import { requestMainWindowClose } from './requestMainWindowClose'

let cleanup: (() => void) | undefined

afterEach(() => cleanup?.())

describe('requestMainWindowClose', () => {
  it('keeps the window open when the active editor guard refuses', async () => {
    cleanup = registerMainWindowCloseGuard(async () => false)
    const close = vi.fn()

    expect(await requestMainWindowClose(close)).toBe(false)
    expect(close).not.toHaveBeenCalled()
  })

  it('closes directly after the active editor guard accepts', async () => {
    const guard = vi.fn(async () => true)
    cleanup = registerMainWindowCloseGuard(guard)
    const close = vi.fn()

    expect(await requestMainWindowClose(close)).toBe(true)
    expect(guard).toHaveBeenCalledOnce()
    expect(close).toHaveBeenCalledOnce()
  })

  it('reports checking and closing stages around a slow close', async () => {
    const stages: string[] = []
    let releaseClose!: () => void
    const close = vi.fn(() => new Promise<void>((resolve) => (releaseClose = resolve)))

    const pending = requestMainWindowClose(close, (stage) => stages.push(stage))
    await vi.waitFor(() => expect(stages).toEqual(['checking', 'closing']))
    expect(close).toHaveBeenCalledOnce()
    releaseClose()

    await expect(pending).resolves.toBe(true)
  })

  it('lets the active guard report restoration before it owns closing', async () => {
    const stages: string[] = []
    cleanup = registerMainWindowCloseGuard(async ({ close, setStage }) => {
      setStage('restoring')
      await close()
      return 'handled' as const
    })

    await expect(requestMainWindowClose(vi.fn(), (stage) => stages.push(stage))).resolves.toBe(true)
    expect(stages).toEqual(['checking', 'restoring', 'closing'])
  })

  it('closes immediately when no guarded surface is active', async () => {
    const close = vi.fn()

    expect(await requestMainWindowClose(close)).toBe(true)
    expect(close).toHaveBeenCalledOnce()
  })

  it('does not close twice when the active guard owns the pending close flow', async () => {
    const close = vi.fn(async () => undefined)
    cleanup = registerMainWindowCloseGuard(async ({ close: guardedClose }) => {
      await guardedClose()
      return 'handled' as const
    })

    expect(await requestMainWindowClose(close)).toBe(true)
    expect(close).toHaveBeenCalledOnce()
  })
})
