// invoke wrapper：统一处理 RPC 调用的 toast / error normalize。
// useToast() 在异步 await 后 Vue injection context 会丢，所以在模块顶层装配 toast 函数。
//
// ToastOpts 只列我们用得到的字段——参数最终传给 NuxtUI useToast().add，那边接受任何
// 兼容的对象。用宽松对象类型避免跟 @nuxt/ui 内部类型耦合，免去 main.ts 那个 as any。
import { i18n } from '@/i18n'

type ToastOpts = Record<string, unknown> & { title: string }
type ToastAdd = (opts: ToastOpts) => void

let _toastAdd: ToastAdd | null = null

export function setupInvoker(toastAdd: ToastAdd) {
  _toastAdd = toastAdd
}

export type NormalizedError = {
  errors?: Array<{ code: string; params?: Record<string, unknown> }>
  code?: string
  params?: Record<string, unknown>
  message?: string
}

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
  // validation 数组: 通道A 是大写 Errors (无 json tag), 通道B 是小写 errors。
  const errs = (src.Errors ?? src.errors) as unknown
  if (Array.isArray(errs) && errs.length > 0) {
    return { errors: errs as NormalizedError['errors'] }
  }
  if (typeof src.code === 'string' && src.code) {
    return { code: src.code, params: src.params as Record<string, unknown> | undefined }
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
    return te(key) ? t(key, (n.params ?? {}) as Record<string, unknown>) : n.code
  }
  if (n.message) return n.message
  return t('error.UNKNOWN_ERROR')
}

function copyText(s: string) {
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(s).catch(() => fallbackCopy(s))
  } else {
    fallbackCopy(s)
  }
}

function fallbackCopy(s: string) {
  const ta = document.createElement('textarea')
  ta.value = s
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  try {
    document.execCommand('copy')
  } catch {}
  document.body.removeChild(ta)
}

export function toastError(msg: string, title?: string) {
  const t = i18n.global.t
  _toastAdd?.({
    title: title ?? t('toast.operation_failed'),
    description: msg,
    color: 'error',
    duration: 6000,
    actions: [
      { label: t('common.copy'), icon: 'i-tabler-copy', color: 'neutral', onClick: () => copyText(msg) },
    ],
  })
}

/** 信息/成功类 toast (走模块级 toast, 可在 store / 异步后安全调用). */
export function toastInfo(title: string, description?: string) {
  _toastAdd?.({ title, description, color: 'success', icon: 'i-tabler-check', duration: 4000 })
}

export async function invoke<R, A extends any[]>(
  fn: (...args: A) => Promise<R>,
  ...args: A
): Promise<R | undefined> {
  try {
    return await fn(...args)
  } catch (e) {
    toastError(errorMessage(e))
    return undefined
  }
}
