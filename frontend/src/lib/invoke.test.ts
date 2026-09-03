import { describe, it, expect } from 'vitest'
import { normalizeError, errorMessage, invoke, RPCError } from './invoke'

describe('normalizeError', () => {
  it('通道A validation: cause.Errors 大写', () => {
    const e = {
      message: 'X map[]',
      cause: { Errors: [{ code: 'MISSING_ENTRY_GRAPH', params: {} }] },
      kind: 'RuntimeError',
    }
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'MISSING_ENTRY_GRAPH', params: {} }] })
  })
  it('通道A apperr: cause.id', () => {
    const e = { message: 'opaque', cause: { id: 'WAILS_NOT_READY', params: { x: 1 } } }
    expect(normalizeError(e)).toEqual({ id: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('native transport: MarshalError cause is a JSON string', () => {
    const e = {
      name: 'RuntimeError',
      message: 'method returned an error',
      cause: JSON.stringify({
        id: 'ai.authoring.provider_failed',
        category: 'adapter',
        params: { stage: 'transport', class: 'timeout' },
        operationId: 'backend-42',
        retryable: true,
      }),
    }
    expect(normalizeError(e)).toEqual({
      id: 'ai.authoring.provider_failed',
      category: 'adapter',
      params: { stage: 'transport', class: 'timeout' },
      operationId: 'backend-42',
      retryable: true,
    })
  })
  it('通道B worker 信封 errors 小写', () => {
    const e = { errors: [{ code: 'MISSING_ENTRY_GRAPH' }] }
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'MISSING_ENTRY_GRAPH' }] })
  })
  it('通道B worker 信封 id', () => {
    const e = { id: 'WAILS_NOT_READY', params: { x: 1 } }
    expect(normalizeError(e)).toEqual({ id: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('裸 message 不进入产品错误契约', () => {
    expect(normalizeError(new Error('boom'))).toEqual({})
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
  it('dev-fetch transport: 信封塞进 Error.message (apperr id)', () => {
    const e = new Error(
      JSON.stringify({
        message: 'opaque',
        cause: { id: 'WAILS_NOT_READY', params: { x: 1 } },
        kind: 'RuntimeError',
      }),
    )
    expect(normalizeError(e)).toEqual({ id: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('dev-fetch transport: e 本身是 JSON 字符串', () => {
    const e = JSON.stringify({ cause: { Errors: [{ code: 'MISSING_ENTRY_GRAPH' }] } })
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'MISSING_ENTRY_GRAPH' }] })
  })
  it('typed envelope: preserves id, params, correlation and retryability', () => {
    const e = {
      cause: {
        id: 'recording.finalize_failed',
        category: 'domain',
        params: { pendingId: 'p1' },
        operationId: 'backend-1',
        runId: 'run-1',
        retryable: true,
      },
    }
    expect(normalizeError(e)).toEqual({
      id: 'recording.finalize_failed',
      category: 'domain',
      params: { pendingId: 'p1' },
      operationId: 'backend-1',
      runId: 'run-1',
      retryable: true,
    })
  })
  it('preserves an already-normalized RPCError instead of decoding its opaque cause again', () => {
    const error = new RPCError(
      {
        id: 'ai.authoring.provider_failed',
        category: 'adapter',
        params: { stage: 'transport', class: 'unknown' },
        operationId: 'backend-op',
        retryable: false,
      },
      'SendWorkflowAIMessage',
      'frontend-op',
      new Error('opaque transport cause'),
    )
    expect(normalizeError(error)).toEqual({
      id: 'ai.authoring.provider_failed',
      category: 'adapter',
      params: { stage: 'transport', class: 'unknown' },
      operationId: 'backend-op',
      retryable: false,
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
    const message = errorMessage({ cause: { id: 'FOO_BAR' } })
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
  it('裸 message 使用安全 fallback', () => {
    expect(errorMessage(new Error('boom'))).toContain('未知错误')
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
  it('把后端 operation ID 展示给用户和开发者用于日志关联', () => {
    const message = errorMessage({
      cause: {
        id: 'recording.finalize.failed',
        params: { destination: 'workflow-resource' },
        operationId: 'backend-recording-42',
      },
    })
    expect(message).toContain('backend-recording-42')
  })
})

describe('invoke', () => {
  it('returns only on success and rethrows a typed RPCError without automatic toast', async () => {
    await expect(invoke(async () => undefined)).resolves.toBeUndefined()
    const rejected = invoke(async function SaveSettings() {
      throw { cause: { id: 'settings.save_failed', category: 'domain' } }
    })
    await expect(rejected).rejects.toMatchObject({
      name: 'RPCError',
      id: 'settings.save_failed',
      category: 'domain',
      operation: 'SaveSettings',
    })
  })

  it('preserves validation arrays after converting to RPCError', async () => {
    const rejected = invoke(async function ApplyPatch() {
      throw { cause: { errors: [{ code: 'INVALID_FIELD', params: { commandIndex: 0 } }] } }
    })
    await expect(rejected).rejects.toMatchObject({
      errors: [{ code: 'INVALID_FIELD', params: { commandIndex: 0 } }],
    })
  })

  it('classifies an unstructured transport rejection instead of calling it system unexpected', async () => {
    const rejected = invoke(async function ApplyPatch() {
      throw new Error('opaque transport rejection')
    })
    await expect(rejected).rejects.toMatchObject({
      id: 'transport.unstructured_failure',
      params: { operation: 'ApplyPatch' },
    })
  })

  it('value RPCs do not turn failure into undefined success', async () => {
    const rejected = invoke(async function FinalizeRecording() {
      throw new Error('encode failed')
    })
    await expect(rejected).rejects.toBeInstanceOf(RPCError)
  })
})
