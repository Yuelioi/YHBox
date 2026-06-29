import { Events } from '@wailsio/runtime'
import { backend } from './backend'
import { useLogStore } from '@/stores/log'
import { useHotkeysStore } from '@/stores/hotkeys'

// wireEvents 把后端推送绑到对应 pinia store.
// v2: bot 概念已删, 只剩 log/hotkey 共享 event + container:* (Task 9 接入).
export function wireEvents() {
  // 日志: SYSTEM 路径 (zerolog → LogSink → log:lines)
  backend.events.onLogLines((d) => useLogStore().appendSystem(d.seq, d.lines))

  // 启动期初始 reload — 保证 keyFor() 在没开过「快捷键」页时也有真实绑定值 (否则一直回退).
  void useHotkeysStore().reload()

  // hotkey:changed: HotkeyRegistry mutate 后广播 — 各窗口 reload 热键列表.
  backend.events.onHotkeyChanged(() => {
    void useHotkeysStore().reload()
  })

  // container 日志 — 后端 emit 走 string event name (container:log / container:node-enter-batch /
  // container:node-dump-batch), 不在 wails bindings 里. 用 raw Events.On.
  Events.On('container:log', (e: any) => {
    const payload = e?.data?.[0] ?? e?.data ?? e
    useLogStore().appendContainerLog({
      level: String(payload?.level ?? 'info'),
      message: String(payload?.message ?? ''),
    })
  })
  Events.On('container:node-enter-batch', (e: any) => {
    const payload = e?.data?.[0] ?? e?.data ?? e
    const entries = payload?.entries
    if (Array.isArray(entries)) {
      useLogStore().appendNodeEnter(entries.map((x: any) => ({
        nodeId: String(x?.nodeId ?? '?'),
        nodeKind: String(x?.nodeKind ?? '?'),
        count: Number(x?.count ?? 1),
      })))
    }
  })
  Events.On('container:node-dump-batch', (e: any) => {
    const payload = e?.data?.[0] ?? e?.data ?? e
    const entries = payload?.entries
    if (Array.isArray(entries)) {
      useLogStore().appendNodeDump(entries.map((x: any) => ({
        nodeId: String(x?.nodeId ?? '?'),
        nodeKind: String(x?.nodeKind ?? '?'),
        lineKey: String(x?.lineKey ?? ''),
        line: String(x?.line ?? ''),
        count: Number(x?.count ?? 1),
        final: Boolean(x?.final),
      })))
    }
  })
  Events.On('container:action-trace', (e: any) => {
    const payload = e?.data?.[0] ?? e?.data ?? e
    if (payload && typeof payload === 'object') {
      useLogStore().appendActionTrace(payload)
    }
  })
}
