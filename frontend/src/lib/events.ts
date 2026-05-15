import { backend } from './backend'
import { BOTS } from './bot-registry'
import { router } from '@/router'
import { useFishStore } from '@/stores/fish'
import { usePianoStore } from '@/stores/piano'
import { useBattleStore } from '@/stores/battle'
import { useGameStore } from '@/stores/game'
import { useLogStore } from '@/stores/log'
import { useActionsStore } from '@/stores/actions'
import { useHotkeysStore } from '@/stores/hotkeys'

// wireEvents 把所有后端推送绑到对应的 pinia store。
// 长跑型 bot 的 state 事件统一从 BOTS 数组派生；bot-specific 的 stats/progress
// 形态各异保留单独订阅。
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

  // actions 事件：state/recorder-state/recorder-event/recorder-toggle/changed 全部转发到 actions store
  backend.events.onActionState((d) => useActionsStore().onStateEvent(d))
  backend.events.onRecorderState((d) => useActionsStore().onRecorderEvent(d))
  backend.events.onRecorderEvent(() => useActionsStore().incRecorderEventCount())
  backend.events.onRecorderToggle(() => {
    // 热键触发时如果不在 /tasks 下，先切过去再 toggle —— 让用户看到倒计时 dialog。
    // 之前判 name==='actions' 是死代码（没有名为 'actions' 的路由），导致每次按 Ctrl+Shift+R
    // 都重 push 一次 /actions → redirect /tasks，把用户从其他 tab 拽走。
    const cur = router.currentRoute.value.path
    if (!cur.startsWith('/tasks')) {
      void router.push('/tasks')
    }
    void useActionsStore().toggleRecording()
  })
  // action:changed: 任何 mutation（create/update/delete/setHotkey/录制保存）后后端广播，
  // 让所有窗口（主窗口 + 独立编辑器窗口）reload list 保持同步。
  backend.events.onActionChanged(() => {
    void useActionsStore().reload()
  })
  // hotkey:changed: HotkeyRegistry mutate 后广播 — 各窗口 reload 热键列表。
  backend.events.onHotkeyChanged(() => {
    void useHotkeysStore().reload()
  })
}
