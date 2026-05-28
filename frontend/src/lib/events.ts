import { backend } from './backend'
import { useGameStore } from '@/stores/game'
import { useLogStore } from '@/stores/log'
import { useHotkeysStore } from '@/stores/hotkeys'

// wireEvents 把后端推送绑到对应 pinia store.
// v2: bot 概念已删, 只剩 log/game/hotkey 共享 event + container:* (Task 8 接入).
export function wireEvents() {
  // 日志: SYSTEM 路径 (zerolog → LogSink → log:lines)
  backend.events.onLogLines((d) => useLogStore().appendSystem(d.seq, d.lines))

  // 共享
  backend.events.onGameStatus((d) => useGameStore().setStatus(d))

  // hotkey:changed: HotkeyRegistry mutate 后广播 — 各窗口 reload 热键列表.
  backend.events.onHotkeyChanged(() => {
    void useHotkeysStore().reload()
  })
}
