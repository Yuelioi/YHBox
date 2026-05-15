// bot-registry.ts —— 前端 bot 单点注册。新增/移除长跑型 bot 只改这一个数组。
//
// 跟后端 cmd/yhbox/services/registry.go 的 KindFish/KindCook/... 对齐。
// battle 是 hotkey-driven 形态不同（state 不是 idle/running/paused 而是 enabled bool），
// 不进 BOTS，由 sidebar/events.ts 单独处理。

import { useFishStore } from '@/stores/fish'
import { useCookStore } from '@/stores/cook'
import { usePianoStore } from '@/stores/piano'
import { useRhythmStore } from '@/stores/rhythm'

export type BotKind = 'fish' | 'cook' | 'piano' | 'rhythm'

export type BotState = 'idle' | 'running' | 'paused'

// BotStateStore 是 BOTS 数组里所有 store 必须实现的最小接口
// （store 上还有别的字段：fish.stats / piano.progress 等，但不在 registry 通用契约里）。
export interface BotStateStore {
  state: BotState
  setState(s: BotState): void
}

export interface BotEntry {
  kind: BotKind
  label: string // sidebar 显示文本（i18n 时改 t(label) 即可）
  icon: string // lucide 图标 name
  route: string // vue-router path
  useStore: () => BotStateStore // pinia store composable
}

// BOTS 是 4 个长跑型 bot 的元数据数组。sidebar 顺序由此决定。
export const BOTS: BotEntry[] = [
  {
    kind: 'fish',
    label: '钓鱼',
    icon: 'i-tabler-fish',
    route: '/',
    useStore: useFishStore,
  },
  {
    kind: 'cook',
    label: '店长',
    icon: 'i-tabler-tools-kitchen-2',
    route: '/cook',
    useStore: useCookStore,
  },
  {
    kind: 'piano',
    label: '弹琴',
    icon: 'i-tabler-music',
    route: '/piano',
    useStore: usePianoStore,
  },
  {
    kind: 'rhythm',
    label: '音游',
    icon: 'i-tabler-music',
    route: '/rhythm',
    useStore: useRhythmStore,
  },
]

// botStateEventName 从 kind 派生 "fish:state" / "rhythm:state"。
// 跟后端 services.EventStateName 完全一致。
export function botStateEventName(kind: BotKind): string {
  return `${kind}:state`
}
