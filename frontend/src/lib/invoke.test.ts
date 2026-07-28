import { describe, it, expect } from 'vitest'
import { normalizeError, errorMessage, friendlyRawErrorMessage, invoke, RPCError } from './invoke'

describe('normalizeError', () => {
  it('通道A validation: cause.Errors 大写', () => {
    const e = {
      message: 'X map[]',
      cause: { Errors: [{ code: 'MISSING_ENTRY_GRAPH', params: {} }] },
      kind: 'RuntimeError',
    }
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'MISSING_ENTRY_GRAPH', params: {} }] })
  })
  it('通道A apperr: cause.code 小写', () => {
    const e = { message: 'WAILS_NOT_READY', cause: { code: 'WAILS_NOT_READY', params: { x: 1 } } }
    expect(normalizeError(e)).toEqual({ code: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('通道B worker 信封 errors 小写', () => {
    const e = { errors: [{ code: 'MISSING_ENTRY_GRAPH' }] }
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'MISSING_ENTRY_GRAPH' }] })
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
        message: 'UNKNOWN_NODE_TYPE map[]',
        cause: {
          Errors: [{ severity: 'error', code: 'UNKNOWN_NODE_TYPE', graphPath: ['main'] }],
        },
        kind: 'RuntimeError',
      }),
    )
    expect(normalizeError(e)).toEqual({
      errors: [{ severity: 'error', code: 'UNKNOWN_NODE_TYPE', graphPath: ['main'] }],
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
    const e = JSON.stringify({ cause: { Errors: [{ code: 'MISSING_ENTRY_GRAPH' }] } })
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'MISSING_ENTRY_GRAPH' }] })
  })
  it('typed envelope: preserves category, details, correlation and retryability', () => {
    const e = {
      cause: {
        code: 'recording.finalize_failed',
        category: 'domain',
        message: 'finalize failed',
        details: { pendingId: 'p1' },
        operationId: 'backend-1',
        runId: 'run-1',
        retryable: true,
      },
    }
    expect(normalizeError(e)).toEqual({
      code: 'recording.finalize_failed',
      category: 'domain',
      params: { pendingId: 'p1' },
      message: 'finalize failed',
      details: { pendingId: 'p1' },
      operationId: 'backend-1',
      runId: 'run-1',
      retryable: true,
    })
  })
})

describe('errorMessage', () => {
  it('validation 首条本地化 + 还有 N 个', () => {
    const e = {
      cause: { Errors: [{ code: 'MISSING_ENTRY_GRAPH' }, { code: 'UNKNOWN_NODE_TYPE' }] },
    }
    const msg = errorMessage(e)
    expect(msg).toContain('入口图')
    expect(msg).toMatch(/1/)
  })
  it('缺 i18n key → 给出处理建议而不是裸错误码', () => {
    const message = errorMessage({ cause: { code: 'FOO_BAR' } })
    expect(message).toContain('重试')
    expect(message).toContain('FOO_BAR')
    expect(message).not.toBe('FOO_BAR')
  })
  it('未知 validation code 也不会直接展示', () => {
    const message = errorMessage({ cause: { Errors: [{ code: 'UNKNOWN_VALIDATION_CODE' }] } })
    expect(message).toContain('重试')
    expect(message).toContain('UNKNOWN_VALIDATION_CODE')
    expect(message).not.toBe('UNKNOWN_VALIDATION_CODE')
  })
  it('裸 message 直接显示', () => {
    expect(errorMessage(new Error('boom'))).toBe('boom')
  })
  it('Wails 只返回错误码 message 时仍本地化', () => {
    const message = errorMessage(new Error('RECORDING_CALIBRATION_REQUIRED'))
    expect(message).toContain('精准相对录制')
    expect(message).not.toContain('RECORDING_CALIBRATION_REQUIRED')
  })
  it('未知裸错误码也不会直接展示', () => {
    const message = errorMessage(new Error('UNMAPPED_FAILURE'))
    expect(message).toContain('重试')
    expect(message).toContain('UNMAPPED_FAILURE')
    expect(message).not.toBe('UNMAPPED_FAILURE')
  })
  it('{} → UNKNOWN_ERROR 文案', () => {
    const msg = errorMessage({})
    expect(msg).not.toBe('[object Object]')
    expect(msg.length).toBeGreaterThan(0)
  })
  it('dev-fetch transport 信封 → 本地化, 不糊 JSON', () => {
    const e = new Error(
      JSON.stringify({
        message: 'UNKNOWN_NODE_TYPE map[]',
        cause: { Errors: [{ code: 'UNKNOWN_NODE_TYPE' }] },
        kind: 'RuntimeError',
      }),
    )
    const msg = errorMessage(e)
    expect(msg).not.toContain('{') // 不再糊裸 JSON
    expect(msg).not.toContain('map[]')
    expect(msg).toContain('节点类型') // zh user-facing diagnostic, not raw code
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
})

describe('invoke', () => {
  it('returns only on success and rethrows a typed RPCError without automatic toast', async () => {
    await expect(invoke(async () => undefined)).resolves.toBeUndefined()
    const rejected = invoke(async function SaveSettings() {
      throw { cause: { code: 'settings.save_failed', category: 'domain', message: 'save failed' } }
    })
    await expect(rejected).rejects.toMatchObject({
      name: 'RPCError',
      code: 'settings.save_failed',
      category: 'domain',
      operation: 'SaveSettings',
    })
  })

  it('value RPCs do not turn failure into undefined success', async () => {
    const rejected = invoke(async function FinalizeRecording() {
      throw new Error('encode failed')
    })
    await expect(rejected).rejects.toBeInstanceOf(RPCError)
  })
})
