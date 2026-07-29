import { describe, expect, it, vi } from 'vitest'
import { requestMainWindowClose } from './requestMainWindowClose'

describe('requestMainWindowClose', () => {
  it('keeps the window open when a dirty workflow aborts route leave', async () => {
    const push = vi.fn(async () => ({ type: 'aborted' }))
    const close = vi.fn()

    const closed = await requestMainWindowClose('workflow-edit', { push }, close)

    expect(push).toHaveBeenCalledWith({ name: 'workflows' })
    expect(close).not.toHaveBeenCalled()
    expect(closed).toBe(false)
  })

  it('closes after the workflow route accepts save or discard', async () => {
    const push = vi.fn(async () => undefined)
    const close = vi.fn()

    const closed = await requestMainWindowClose('workflow-edit', { push }, close)

    expect(close).toHaveBeenCalledOnce()
    expect(closed).toBe(true)
  })

  it('closes immediately outside the workflow editor', async () => {
    const push = vi.fn()
    const close = vi.fn()

    const closed = await requestMainWindowClose('settings', { push }, close)

    expect(push).not.toHaveBeenCalled()
    expect(close).toHaveBeenCalledOnce()
    expect(closed).toBe(true)
  })
})
