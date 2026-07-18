// Wails transport seam：统一解码并抛出 typed RPCError。
// 这里不决定 UI 反馈；调用方按 domain action 选择 inline、modal 或 failure toast。
import { i18n } from '@/i18n'

export type NormalizedError = {
  errors?: Array<{ code: string; params?: Record<string, unknown> }>
  code?: string
  category?: string
  params?: Record<string, unknown>
  message?: string
  details?: unknown
  operationId?: string
  runId?: string
  retryable?: boolean
}

export class RPCError extends Error {
  readonly code: string
  readonly category: string
  readonly details?: unknown
  readonly operation: string
  readonly operationId: string
  readonly runId?: string
  readonly retryable: boolean
  readonly source: unknown

  constructor(error: NormalizedError, operation: string, operationId: string, source: unknown) {
    super(error.message || error.code || 'RPC call failed', { cause: source })
    this.name = 'RPCError'
    this.code = error.code || 'rpc.unclassified'
    this.category = error.category || 'infrastructure'
    this.details = error.details ?? error.params ?? error.errors
    this.operation = operation
    this.operationId = error.operationId || operationId
    this.runId = error.runId
    this.retryable = error.retryable === true
    this.source = source
  }
}

let operationSequence = 0

/** 把两条投递通道(wails RPC .cause / worker 事件信封)规整成统一形态。 */
export function normalizeError(e: unknown): NormalizedError {
  if (e == null) return {}
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
  const src = (obj.cause ?? obj) as Record<string, unknown>
  const details = src.details
  const detailObject =
    details && typeof details === 'object' ? (details as Record<string, unknown>) : {}
  // validation 数组: 历史通道是大写 Errors/小写 errors；typed envelope 放 details。
  const errs = (src.Errors ?? src.errors ?? detailObject.Errors ?? detailObject.errors) as unknown
  if (Array.isArray(errs) && errs.length > 0) {
    const result: NormalizedError = { errors: errs as NormalizedError['errors'] }
    if (typeof src.code === 'string') result.code = src.code
    if (typeof src.category === 'string') result.category = src.category
    if (typeof src.message === 'string') result.message = src.message
    if (details !== undefined) result.details = details
    if (typeof src.operationId === 'string') result.operationId = src.operationId
    if (typeof src.runId === 'string') result.runId = src.runId
    if (typeof src.retryable === 'boolean') result.retryable = src.retryable
    return result
  }
  if (typeof src.code === 'string' && src.code) {
    const params =
      src.params && typeof src.params === 'object'
        ? (src.params as Record<string, unknown>)
        : details && typeof details === 'object' && !Array.isArray(details)
          ? (details as Record<string, unknown>)
          : undefined
    const result: NormalizedError = { code: src.code }
    if (typeof src.category === 'string') result.category = src.category
    if (params !== undefined) result.params = params
    if (typeof src.message === 'string') result.message = src.message
    if (details !== undefined) result.details = details
    if (typeof src.operationId === 'string') result.operationId = src.operationId
    if (typeof src.runId === 'string') result.runId = src.runId
    if (typeof src.retryable === 'boolean') result.retryable = src.retryable
    return result
  }
  // obj.message 优先: 信封已解包时这是人类可读的那行, 不是整坨 JSON。
  if (typeof obj.message === 'string' && obj.message) return { message: obj.message }
  if (e instanceof Error && e.message) return { message: e.message }
  if (typeof e === 'string' && e) return { message: e }
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
    const head = te(key) ? t(key, (first.params ?? {}) as Record<string, unknown>) : first.code
    const rest = n.errors.length - 1
    return rest > 0 ? `${head}${t('toast.and_n_more', { n: rest })}` : head
  }
  if (n.code) {
    const key = `error.${n.code}`
    if (te(key)) return t(key, (n.params ?? {}) as Record<string, unknown>)
    if (n.code === 'rpc.unclassified' && n.message) return friendlyRawErrorMessage(n.message)
    return n.code
  }
  if (n.message) return friendlyRawErrorMessage(n.message)
  return t('error.UNKNOWN_ERROR')
}

/** 把底层 transport 文案收敛为用户知道下一步该做什么的提示。 */
export function friendlyRawErrorMessage(message: string): string {
  const normalized = message.trim().toLowerCase()
  if (
    normalized.includes('context deadline exceeded') ||
    normalized.includes('deadline exceeded') ||
    normalized.includes('request canceled while waiting')
  ) {
    return i18n.global.t('error.TRANSPORT_TIMEOUT')
  }
  if (
    normalized.includes('connection refused') ||
    normalized.includes('no connection could be made')
  ) {
    return i18n.global.t('error.TRANSPORT_UNAVAILABLE')
  }
  return message
}

export function toRPCError(error: unknown, operation = 'rpc.call'): RPCError {
  if (error instanceof RPCError) return error
  const normalized = normalizeError(error)
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
