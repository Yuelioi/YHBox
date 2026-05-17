import { backend } from './backend'
import { BOTS } from './bot-registry'
import { useFishStore } from '@/stores/fish'
import { usePianoStore } from '@/stores/piano'
import { useBattleStore } from '@/stores/battle'
import { useGameStore } from '@/stores/game'
import { useLogStore } from '@/stores/log'
import { useHotkeysStore } from '@/stores/hotkeys'

// wireEvents 把所有后端推送绑到对应的 pinia store。
// 长跑型 bot 的 state 事件统一从 BOTS 数组派生；bot-specific 的 stats/progress
// 形态各异保留单独订阅。
// v2: actions 事件已废，录制走 container 编辑器直触发。
export function wireEvents() {
  // 日志
  backend.events.onLogLines((d) => useLogStore().append(d.seq, d.lines))

  // 4 个长跑型 bot 的 state 事件 → store.setState
  for (const bot of BOTS) {
    backend.events.onBotState(bot.kind, (d) => bot.useStore().setState(d.state))
  }

  // bot-specific 事件
  backend.events.onFishStats((d) => useFishStore().setStats(d))
  backend.events.onPianoProgress((d) => usePianoStore().setProgress(d))
  backend.events.onBattleState((d) => useBattleStore().setStatus(d))

  // 共享
  backend.events.onGameStatus((d) => useGameStore().setStatus(d))

  // hotkey:changed: HotkeyRegistry mutate 后广播 — 各窗口 reload 热键列表。
  backend.events.onHotkeyChanged(() => {
    void useHotkeysStore().reload()
  })
}
