// Wails transport seam：统一解码并抛出 typed RPCError。
// 这里不决定 UI 反馈；调用方按 domain action 选择 inline、modal 或 failure toast。
import { i18n } from '@/i18n'

export type NormalizedError = {
  errors?: Array<{ code: string; params?: Record<string, unknown> }>
  id?: string
  category?: string
  params?: Record<string, unknown>
  operationId?: string
  runId?: string
  retryable?: boolean
}

export class RPCError extends Error {
  readonly errors?: Array<{ code: string; params?: Record<string, unknown> }>
  readonly id: string
  readonly category: string
  readonly params?: Record<string, unknown>
  readonly operation: string
  readonly operationId: string
  readonly runId?: string
  readonly retryable: boolean
  readonly source: unknown

  constructor(error: NormalizedError, operation: string, operationId: string, source: unknown) {
    super(error.id || 'system.unexpected', { cause: source })
    this.name = 'RPCError'
    this.errors = error.errors
    this.id = error.id || 'system.unexpected'
    this.category = error.category || 'infrastructure'
    this.params = error.params
    this.operation = operation
    this.operationId = error.operationId || operationId
    this.runId = error.runId
    this.retryable = error.retryable === true
    this.source = source
  }
}

let operationSequence = 0

export type TransportContractViolation = { operation: string; shape: string }
let reportTransportContractViolation = (violation: TransportContractViolation) => {
  console.error('RPC transport contract violation', violation)
}

export function setTransportContractViolationReporter(
  reporter: (violation: TransportContractViolation) => void,
): () => void {
  const previous = reportTransportContractViolation
  reportTransportContractViolation = reporter
  return () => {
    reportTransportContractViolation = previous
  }
}

function transportShape(value: unknown): string {
  if (value === null) return 'null'
  if (value === undefined) return 'undefined'
  if (value instanceof Error) return 'error'
  if (Array.isArray(value)) return 'array'
  if (typeof value === 'object') {
    const keys = Object.keys(value as Record<string, unknown>)
      .sort()
      .slice(0, 8)
    return keys.length ? `object:${keys.join(',')}` : 'object:empty'
  }
  return typeof value
}

/** 把两条投递通道(wails RPC .cause / worker 事件信封)规整成统一形态。 */
export function normalizeError(e: unknown): NormalizedError {
  if (e == null) return {}
  if (e instanceof RPCError) {
    const normalized: NormalizedError = {
      id: e.id,
      category: e.category,
      operationId: e.operationId,
      retryable: e.retryable,
    }
    if (e.errors !== undefined) normalized.errors = e.errors
    if (e.params !== undefined) normalized.params = e.params
    if (e.runId !== undefined) normalized.runId = e.runId
    return normalized
  }
  let obj = typeof e === 'object' ? (e as Record<string, unknown>) : {}
  // wails dev-mode fetch transport (@wailsio/runtime runtime.js:103) 抛 `new Error(responseText)` ——
  // 整个 {message,cause,kind} 信封被塞进字符串 (e 本身是 string, 或 Error.message), 没拆成 .cause 属性。
  // 先把信封 JSON 解出来当错误对象, 才能拿到 cause。原生 transport 直接给对象时这步是 no-op。
  const envStr = typeof e === 'string' ? e : typeof obj.message === 'string' ? obj.message : ''
  if (envStr.startsWith('{')) {
    try {
      const env = JSON.parse(envStr)
      if (env && typeof env === 'object') obj = env as Record<string, unknown>
    } catch {
      /* 不是 JSON 信封, 保持原 obj */
    }
  }
  // 通道A: wails 包了一层 .cause; 通道B: e 本身就是 RunError 对象。两者都从 src 取。
  let source = obj.cause ?? obj
  if (typeof source === 'string' && source.trimStart().startsWith('{')) {
    try {
      source = JSON.parse(source)
    } catch {
      /* malformed native cause remains unstructured */
    }
  }
  const src = source && typeof source === 'object' ? (source as Record<string, unknown>) : {}
  const params =
    src.params && typeof src.params === 'object'
      ? (src.params as Record<string, unknown>)
      : undefined
  // validation 数组: 历史通道是大写 Errors/小写 errors；typed envelope 放 details。
  const errs = (src.Errors ?? src.errors) as unknown
  if (Array.isArray(errs) && errs.length > 0) {
    const result: NormalizedError = { errors: errs as NormalizedError['errors'] }
    if (typeof src.id === 'string') result.id = src.id
    if (typeof src.category === 'string') result.category = src.category
    if (params !== undefined) result.params = params
    if (typeof src.operationId === 'string') result.operationId = src.operationId
    if (typeof src.runId === 'string') result.runId = src.runId
    if (typeof src.retryable === 'boolean') result.retryable = src.retryable
    return result
  }
  if (typeof src.id === 'string' && src.id) {
    const result: NormalizedError = { id: src.id }
    if (typeof src.category === 'string') result.category = src.category
    if (params !== undefined) result.params = params
    if (typeof src.operationId === 'string') result.operationId = src.operationId
    if (typeof src.runId === 'string') result.runId = src.runId
    if (typeof src.retryable === 'boolean') result.retryable = src.retryable
    return result
  }
  // Unknown transport values never become product text. They are retained only
  // as RPCError.source for developer diagnostics.
  return {}
}

/** 从任意 thrown 值/事件错误信封提取**已本地化**的人类可读 message。 */
export function errorMessage(e: unknown): string {
  const t = i18n.global.t
  const te = i18n.global.te
  const n = normalizeError(e)
  if (n.errors && n.errors.length > 0) {
    const first = n.errors[0]
    const key = `error.${first.code}`
    const head = te(key)
      ? t(key, (first.params ?? {}) as Record<string, unknown>)
      : t('error.UNEXPECTED_CODE', { code: first.code })
    const rest = n.errors.length - 1
    return correlatedErrorMessage(
      rest > 0 ? `${head}${t('toast.and_n_more', { n: rest })}` : head,
      n,
    )
  }
  if (n.id) {
    const key = `error.${n.id}`
    const message = te(key)
      ? t(key, (n.params ?? {}) as Record<string, unknown>)
      : t('error.UNEXPECTED_CODE', { code: n.id })
    return correlatedErrorMessage(message, n)
  }
  return t('error.UNKNOWN_ERROR')
}

function correlatedErrorMessage(message: string, error: NormalizedError): string {
  if (!error.operationId) return message
  return `${message}\n${i18n.global.t('error.OPERATION_ID', { id: error.operationId })}`
}

export function toRPCError(error: unknown, operation = 'rpc.call'): RPCError {
  if (error instanceof RPCError) return error
  const normalized = normalizeError(error)
  if (!normalized.id && !normalized.errors?.length) {
    reportTransportContractViolation({ operation, shape: transportShape(error) })
    normalized.id = 'transport.unstructured_failure'
    normalized.category = 'infrastructure'
    normalized.params = { operation }
  }
  const operationId = `${operation}:${++operationSequence}`
  return new RPCError(normalized, operation, operationId, error)
}

export async function callRPC<R>(operation: string, call: () => Promise<R>): Promise<R> {
  try {
    return await call()
  } catch (error) {
    throw toRPCError(error, operation)
  }
}

export async function invoke<R, A extends any[]>(
  fn: (...args: A) => Promise<R>,
  ...args: A
): Promise<R> {
  return callRPC(fn.name || 'rpc.call', () => fn(...args))
}
