// LogLine 解析 + tag/level Tailwind class 映射。
// 全部按 dark-mode 终端配色（黑底亮字），与 walk 时代的 light-mode 色板完全不同。
// parseLine 必须容错（zerolog 输出通常 well-formed，但绕过 logger 写到 stderr 的
// 非 JSON 行——panic 栈、第三方库直写——会 sneak 进同一 sink。一条坏行不能炸 LogPanel。

export interface LogLine {
  time: string
  level: string
  bot?: string
  tag?: string
  message: string
  source: 'SYS' | 'CTR'
}

export function parseLine(raw: string): LogLine {
  try {
    const o = JSON.parse(raw)
    return {
      time: o.time ?? '',
      level: o.level ?? 'info',
      bot: o.bot,
      tag: o.tag,
      message: o.message ?? raw,
      source: 'SYS',
    }
  } catch {
    return { time: '', level: 'error', message: raw, source: 'SYS' }
  }
}

// TAG 颜色 — dark-mode 暗底专用，使用 -300 / -400 系列保证对比度。
// 同时保留 -500/15 背景让 tag 像 chip 一样跳出来。
const TAG_COLORS: Record<string, string> = {
  SYSTEM: 'text-rose-300    bg-rose-500/10',
  READY: 'text-toned       bg-elevated',
  IDLE: 'text-toned       bg-elevated',
  SETUP: 'text-teal-300    bg-teal-500/10',
  WAITING: 'text-toned       bg-elevated',
  CAST: 'text-purple-300  bg-purple-500/10',
  FIGHT: 'text-sky-300     bg-sky-500/10',
  FISHING: 'text-sky-300     bg-sky-500/10',
  // RESULT 是钓鱼成功的结算画面，要最显眼；用 amber 跟其它 emerald 区分
  RESULT: 'text-amber-200   bg-amber-500/20 font-bold',
  RECOVERING: 'text-amber-300   bg-amber-500/10',
  RECOV: 'text-amber-300   bg-amber-500/10',
  BUY: 'text-emerald-300 bg-emerald-500/10',
  BUYBAIT: 'text-emerald-300 bg-emerald-500/10',
  SHOPSELL: 'text-emerald-300 bg-emerald-500/10',
  SELL: 'text-emerald-300 bg-emerald-500/10',
  EQUIP: 'text-purple-300  bg-purple-500/10',
  CHANGEBAIT: 'text-purple-300  bg-purple-500/10',
  MISS: 'text-rose-300    bg-rose-500/10',
  STOP: 'text-rose-300    bg-rose-500/10',
  PAUSE: 'text-amber-300   bg-amber-500/10',
  CLICK: 'text-violet-300  bg-violet-500/10',
}
const TAG_FALLBACK = 'text-indigo-300 bg-indigo-500/10'

export function tagClass(tag: string): string {
  return TAG_COLORS[tag?.toUpperCase()] ?? TAG_FALLBACK
}

// LEVEL 颜色 — 黑底正文。info 用 highlighted（明显），不再依赖 dark: 切换。
const LEVEL_COLORS: Record<string, string> = {
  trace: 'text-dimmed',
  debug: 'text-muted',
  info: 'text-highlighted',
  warn: 'text-amber-300',
  error: 'text-rose-300',
  fatal: 'text-rose-200 font-semibold',
}

export function levelClass(level: string): string {
  return LEVEL_COLORS[level?.toLowerCase()] ?? LEVEL_COLORS.info
}
