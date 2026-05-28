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

/** 从任意 thrown 值提取人类可读 message。Wails3 抛 {message,cause,kind} 对象。 */
export function errorMessage(e: unknown): string {
  if (e == null) return 'unknown error'
  if (e instanceof Error) return e.message
  if (typeof e === 'string') return e
  if (typeof e === 'object') {
    const obj = e as Record<string, unknown>
    if (typeof obj.message === 'string' && obj.message) return obj.message
    if (typeof obj.error === 'string' && obj.error) return obj.error
    try {
      return JSON.stringify(e)
    } catch {
      return String(e)
    }
  }
  return String(e)
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
