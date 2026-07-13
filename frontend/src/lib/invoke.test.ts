import { describe, it, expect, vi } from 'vitest'
import {
  normalizeError,
  errorMessage,
  friendlyRawErrorMessage,
  setupInvoker,
  toastError,
} from './invoke'

describe('normalizeError', () => {
  it('通道A validation: cause.Errors 大写', () => {
    const e = {
      message: 'X map[]',
      cause: { Errors: [{ code: 'NO_START', params: {} }] },
      kind: 'RuntimeError',
    }
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'NO_START', params: {} }] })
  })
  it('通道A apperr: cause.code 小写', () => {
    const e = { message: 'WAILS_NOT_READY', cause: { code: 'WAILS_NOT_READY', params: { x: 1 } } }
    expect(normalizeError(e)).toEqual({ code: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('通道B worker 信封 errors 小写', () => {
    const e = { errors: [{ code: 'NO_START' }] }
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'NO_START' }] })
  })
  it('通道B worker 信封 code', () => {
    const e = { code: 'WAILS_NOT_READY', params: { x: 1 } }
    expect(normalizeError(e)).toEqual({ code: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('裸 message 回落', () => {
    expect(normalizeError(new Error('boom'))).toEqual({ message: 'boom' })
  })
  it('空/未知 → {}', () => {
    expect(normalizeError({})).toEqual({})
  })
  // wails dev-mode fetch transport (runtime.js:103) 抛 `new Error(responseText)`:
  // 整个 {message,cause,kind} 信封被塞进 Error.message 字符串, 没拆成 .cause 属性。
  it('dev-fetch transport: 信封塞进 Error.message (validation)', () => {
    const e = new Error(
      JSON.stringify({
        message: 'MISSING_WIN32_WINDOW_TARGET map[]',
        cause: {
          Errors: [{ severity: 'error', code: 'MISSING_WIN32_WINDOW_TARGET', graphPath: ['main'] }],
        },
        kind: 'RuntimeError',
      }),
    )
    expect(normalizeError(e)).toEqual({
      errors: [{ severity: 'error', code: 'MISSING_WIN32_WINDOW_TARGET', graphPath: ['main'] }],
    })
  })
  it('dev-fetch transport: 信封塞进 Error.message (apperr code)', () => {
    const e = new Error(
      JSON.stringify({
        message: 'WAILS_NOT_READY',
        cause: { code: 'WAILS_NOT_READY', params: { x: 1 } },
        kind: 'RuntimeError',
      }),
    )
    expect(normalizeError(e)).toEqual({ code: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('dev-fetch transport: e 本身是 JSON 字符串', () => {
    const e = JSON.stringify({ cause: { Errors: [{ code: 'NO_START' }] } })
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'NO_START' }] })
  })
})

describe('errorMessage', () => {
  it('validation 首条本地化 + 还有 N 个', () => {
    const e = { cause: { Errors: [{ code: 'NO_START' }, { code: 'MISSING_WIN32_WINDOW_TARGET' }] } }
    const msg = errorMessage(e)
    expect(msg).toContain('Start')
    expect(msg).toMatch(/1/)
  })
  it('缺 i18n key → 回落 code 字面', () => {
    expect(errorMessage({ cause: { code: 'FOO_BAR' } })).toBe('FOO_BAR')
  })
  it('裸 message 直接显示', () => {
    expect(errorMessage(new Error('boom'))).toBe('boom')
  })
  it('{} → UNKNOWN_ERROR 文案', () => {
    const msg = errorMessage({})
    expect(msg).not.toBe('[object Object]')
    expect(msg.length).toBeGreaterThan(0)
  })
  it('dev-fetch transport 信封 → 本地化, 不糊 JSON', () => {
    const e = new Error(
      JSON.stringify({
        message: 'MISSING_WIN32_WINDOW_TARGET map[]',
        cause: { Errors: [{ code: 'MISSING_WIN32_WINDOW_TARGET' }] },
        kind: 'RuntimeError',
      }),
    )
    const msg = errorMessage(e)
    expect(msg).not.toContain('{') // 不再糊裸 JSON
    expect(msg).not.toContain('map[]')
    expect(msg).toContain('Windows 窗口目标') // zh user-facing label, not raw kind
  })
})

describe('friendlyRawErrorMessage', () => {
  it('turns a transport timeout into an actionable message', () => {
    const msg = friendlyRawErrorMessage('context deadline exceeded')
    expect(msg).toContain('超时')
    expect(msg).toContain('重试')
    expect(msg).not.toContain('deadline')
  })

  it('keeps unknown technical messages intact for diagnosis', () => {
    expect(friendlyRawErrorMessage('boom')).toBe('boom')
  })

  it('offers retry while preserving the technical detail for copy', () => {
    const retry = vi.fn()
    let toast: Record<string, unknown> | undefined
    setupInvoker((options) => {
      toast = options
    })

    toastError(new Error('context deadline exceeded'), '加载设备失败', retry)

    const actions = toast?.actions as Array<{ onClick: () => void }> | undefined
    expect(String(toast?.description)).toContain('超时')
    expect(actions).toHaveLength(2)
    actions?.[0]?.onClick()
    expect(retry).toHaveBeenCalledOnce()
  })
})
