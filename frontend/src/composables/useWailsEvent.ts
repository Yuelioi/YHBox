// useWailsEvent: wails3 Events.On 订阅的 Vue 自动清理包装。
//
// 之前多处复制：
//   const off = await Events.On('foo', cb) as unknown as () => void
//   onUnmounted(() => { if (off && typeof (off as any) === 'function') off() })
//
// 现在只写：
//   useWailsEvent('foo', (payload) => { ... })
//
// payload 是后端 emit 时传的 data（已展开），不是 WailsEvent 包装。
// 注意 wails3 Events.On 签名是 (name, callback) → 同步返 unsubscribe 函数。

import { onScopeDispose } from 'vue'
import { Events } from '@wailsio/runtime'

type AnyFn = (...args: any[]) => any

export function useWailsEvent<T = unknown>(name: string, handler: (payload: T) => void): void {
  // wails3 包装：callback 收 WailsEvent {name, data, sender}，业务关心 data
  const off = Events.On(name, (e: any) => {
    const payload: T = (e?.data ?? e) as T
    handler(payload)
  })
  onScopeDispose(() => {
    if (typeof (off as unknown as AnyFn) === 'function') {
      ;(off as unknown as AnyFn)()
    }
  })
}

// awaitWailsEvent 订阅一次，等到 match 函数返 true 的 payload 就 resolve 并自动 unsub。
// match 返 false 的 payload 继续等。默认永不超时；调用方可传 AbortSignal 取消等待。
//
// 典型用法（屏幕选择器）：
//   const r = await awaitWailsEvent<PickerResult>(
//     'tools:picker-result',
//     (p) => p.id === myID,
//   )
export function awaitWailsEvent<T = unknown>(
  name: string,
  match: (payload: T) => boolean,
  signal?: AbortSignal,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    let off: AnyFn | undefined
    let settled = false

    const cleanup = () => {
      off?.()
      signal?.removeEventListener('abort', handleAbort)
    }
    const handleAbort = () => {
      if (settled) return
      settled = true
      cleanup()
      reject(signal?.reason ?? new DOMException('The operation was aborted', 'AbortError'))
    }

    if (signal?.aborted) {
      handleAbort()
      return
    }

    const subscription = Events.On(name, (e: any) => {
      const payload: T = (e?.data ?? e) as T
      if (!match(payload)) return
      if (settled) return
      settled = true
      cleanup()
      resolve(payload)
    })
    if (typeof (subscription as unknown as AnyFn) === 'function') {
      off = subscription as unknown as AnyFn
    }
    signal?.addEventListener('abort', handleAbort, { once: true })
  })
}
