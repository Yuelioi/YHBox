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
