// 视觉注册中心 — 颜色 + 图标的单一真源.
//
// 颜色: 一组 canonical Tailwind tone, 既给 group 节点配色 (tailwind class), 也给用户可选色板 (hex swatch).
//   ⚠ Tailwind class 必须字面量 (JIT 扫源码才会打进产物), 不可 `bg-${tone}-500` 拼接.
//   hex 用于动态内联上色 (swatch / CommentBox 卡片) —— class 拼不出来的场合.
// 图标: 一组可选 tabler 图标 (CommentBox / 以后 per-node 自定义).
//
// 取代散落的: adapter VISUAL_BY_GROUP / useNodeGroupColor GROUP_TONE(已与前者漂移)/
//            SaveSnippetDrawer colorPalette / commentBoxColors. 全部从这里取, 消重 + 修漂移.

export interface PaletteEntry {
  key: string
  labelKey: string // i18n: palette.<key>
  bg: string // 字面 tailwind, e.g. 'bg-blue-500/15'
  border: string // 'border-blue-500/40'
  text: string // 'text-blue-400'
  hex: string // tailwind-500 hex, 给 swatch / 内联上色
}

// 字面量写全 (不可拼接 — Tailwind JIT).
export const PALETTE: Record<string, PaletteEntry> = {
  red: { key: 'red', labelKey: 'palette.red', bg: 'bg-red-500/15', border: 'border-red-500/40', text: 'text-red-400', hex: '#ef4444' },
  orange: { key: 'orange', labelKey: 'palette.orange', bg: 'bg-orange-500/15', border: 'border-orange-500/40', text: 'text-orange-400', hex: '#f97316' },
  amber: { key: 'amber', labelKey: 'palette.amber', bg: 'bg-amber-500/15', border: 'border-amber-500/40', text: 'text-amber-400', hex: '#f59e0b' },
  yellow: { key: 'yellow', labelKey: 'palette.yellow', bg: 'bg-yellow-500/15', border: 'border-yellow-500/40', text: 'text-yellow-400', hex: '#eab308' },
  lime: { key: 'lime', labelKey: 'palette.lime', bg: 'bg-lime-500/15', border: 'border-lime-500/40', text: 'text-lime-400', hex: '#84cc16' },
  green: { key: 'green', labelKey: 'palette.green', bg: 'bg-green-500/15', border: 'border-green-500/40', text: 'text-green-400', hex: '#22c55e' },
  emerald: { key: 'emerald', labelKey: 'palette.emerald', bg: 'bg-emerald-500/15', border: 'border-emerald-500/40', text: 'text-emerald-400', hex: '#10b981' },
  teal: { key: 'teal', labelKey: 'palette.teal', bg: 'bg-teal-500/15', border: 'border-teal-500/40', text: 'text-teal-400', hex: '#14b8a6' },
  cyan: { key: 'cyan', labelKey: 'palette.cyan', bg: 'bg-cyan-500/15', border: 'border-cyan-500/40', text: 'text-cyan-400', hex: '#06b6d4' },
  sky: { key: 'sky', labelKey: 'palette.sky', bg: 'bg-sky-500/15', border: 'border-sky-500/40', text: 'text-sky-400', hex: '#0ea5e9' },
  blue: { key: 'blue', labelKey: 'palette.blue', bg: 'bg-blue-500/15', border: 'border-blue-500/40', text: 'text-blue-400', hex: '#3b82f6' },
  indigo: { key: 'indigo', labelKey: 'palette.indigo', bg: 'bg-indigo-500/15', border: 'border-indigo-500/40', text: 'text-indigo-400', hex: '#6366f1' },
  violet: { key: 'violet', labelKey: 'palette.violet', bg: 'bg-violet-500/15', border: 'border-violet-500/40', text: 'text-violet-400', hex: '#8b5cf6' },
  purple: { key: 'purple', labelKey: 'palette.purple', bg: 'bg-purple-500/15', border: 'border-purple-500/40', text: 'text-purple-400', hex: '#a855f7' },
  pink: { key: 'pink', labelKey: 'palette.pink', bg: 'bg-pink-500/15', border: 'border-pink-500/40', text: 'text-pink-400', hex: '#ec4899' },
  rose: { key: 'rose', labelKey: 'palette.rose', bg: 'bg-rose-500/15', border: 'border-rose-500/40', text: 'text-rose-400', hex: '#f43f5e' },
  zinc: { key: 'zinc', labelKey: 'palette.zinc', bg: 'bg-zinc-500/15', border: 'border-zinc-500/40', text: 'text-zinc-400', hex: '#71717a' },
}

export const PALETTE_KEYS = Object.keys(PALETTE)
export const DEFAULT_PALETTE_KEY = 'zinc'

export function resolvePalette(key: string | undefined | null): PaletteEntry {
  return PALETTE[key ?? ''] ?? PALETTE[DEFAULT_PALETTE_KEY]
}

/** hex → palette key (供 snippet 等存 hex 的旧消费方反查选中态). 找不到返 undefined. */
export function paletteKeyByHex(hex: string | undefined | null): string | undefined {
  if (!hex) return undefined
  return PALETTE_KEYS.find((k) => PALETTE[k].hex.toLowerCase() === hex.toLowerCase())
}

// group → 配色 key + 图标 (取代 VISUAL_BY_GROUP; 沿用迁移前的真实色, 节点配色目测不变).
export const GROUP_VISUAL: Record<string, { color: string; icon: string }> = {
  control: { color: 'blue', icon: 'i-tabler-arrow-bounce' },
  variables: { color: 'cyan', icon: 'i-tabler-variable' },
  purefunc: { color: 'lime', icon: 'i-tabler-math-function' },
  detect: { color: 'violet', icon: 'i-tabler-eye' },
  input: { color: 'orange', icon: 'i-tabler-device-gamepad' },
  system: { color: 'zinc', icon: 'i-tabler-settings' },
  io: { color: 'sky', icon: 'i-tabler-message-circle' },
  stopwatch: { color: 'amber', icon: 'i-tabler-clock' },
  mock: { color: 'pink', icon: 'i-tabler-test-pipe' },
  test: { color: 'pink', icon: 'i-tabler-flask' },
  event: { color: 'rose', icon: 'i-tabler-player-play' },
  random: { color: 'teal', icon: 'i-tabler-dice' },
  list: { color: 'indigo', icon: 'i-tabler-list' },
}

export function groupVisual(group: string | undefined): { color: string; icon: string } {
  return GROUP_VISUAL[group ?? ''] ?? GROUP_VISUAL.system
}

// 可选图标集 (CommentBox / 以后 per-node 自定义). 存的是完整 tabler 名 (直接喂 UIcon :name).
export interface IconEntry {
  icon: string
}
export const ICONS: IconEntry[] = [
  { icon: 'i-tabler-note' },
  { icon: 'i-tabler-info-circle' },
  { icon: 'i-tabler-alert-triangle' },
  { icon: 'i-tabler-bulb' },
  { icon: 'i-tabler-star' },
  { icon: 'i-tabler-flag' },
  { icon: 'i-tabler-pin' },
  { icon: 'i-tabler-bookmark' },
  { icon: 'i-tabler-help-circle' },
  { icon: 'i-tabler-circle-check' },
  { icon: 'i-tabler-circle-x' },
  { icon: 'i-tabler-list-check' },
  { icon: 'i-tabler-book' },
  { icon: 'i-tabler-bolt' },
  { icon: 'i-tabler-settings' },
  { icon: 'i-tabler-target' },
]
export const DEFAULT_ICON = 'i-tabler-note'
