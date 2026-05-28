import { Events } from '@wailsio/runtime'
import { backend } from './backend'
import { useGameStore } from '@/stores/game'
import { useLogStore } from '@/stores/log'
import { useHotkeysStore } from '@/stores/hotkeys'

// wireEvents 把后端推送绑到对应 pinia store.
// v2: bot 概念已删, 只剩 log/game/hotkey 共享 event + container:* (Task 9 接入).
export function wireEvents() {
  // 日志: SYSTEM 路径 (zerolog → LogSink → log:lines)
  backend.events.onLogLines((d) => useLogStore().appendSystem(d.seq, d.lines))

  // 共享
  backend.events.onGameStatus((d) => useGameStore().setStatus(d))

  // hotkey:changed: HotkeyRegistry mutate 后广播 — 各窗口 reload 热键列表.
  backend.events.onHotkeyChanged(() => {
    void useHotkeysStore().reload()
  })

  // container 日志 — 后端 emit 走 string event name (container:log / container:node-enter-batch /
  // container:node-log), 不在 wails bindings 里. 用 raw Events.On.
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
  Events.On('container:node-log', (e: any) => {
    const payload = e?.data?.[0] ?? e?.data ?? e
    useLogStore().appendNodeLog({
      nodeId: String(payload?.nodeId ?? '?'),
      nodeKind: String(payload?.nodeKind ?? '?'),
      message: String(payload?.message ?? ''),
    })
  })
}
