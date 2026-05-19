// frontend/src/components/containers/nodeRegistry/specs/control.ts
// Control-flow node kinds (10): Start, Stop, Sleep, Loop, If, Switch, Parallel, Race, Break, Continue.
// Mirrors backend internal/services/container/nodekind/specs/control.go.
import { register } from '../registry'

// Parallel/Race dynamic branch count — clamped [2,8]. v4 spec §6.2: reads cfg.literal.n
// (NOT cfg.n — that was the v3 lookup, fixed in backend per "no compat" rule).
function parallelBranchPins(cfg: Record<string, unknown> | null | undefined): string[] {
  const lit = (cfg?.literal as Record<string, unknown> | undefined) ?? {}
  const raw = Number(lit.n)
  const n = Math.max(2, Math.min(8, Number.isFinite(raw) && raw > 0 ? Math.floor(raw) : 2))
  const out: string[] = []
  for (let i = 0; i < n; i++) out.push(`branch${i}`)
  out.push('complete')
  return out
}

register({
  kind: 'Start',
  group: 'control',
  labelZh: '开始',
  description: '容器唯一入口。每个容器必须有且仅有一个 Start。',
  visual: { icon: 'i-tabler-player-play', bg: 'bg-emerald-500/15', border: 'border-emerald-500/40' },
  execIn: [],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: {},
})

register({
  kind: 'Stop',
  group: 'control',
  labelZh: '结束',
  description: '立即结束当前容器（不影响外层 schedule 后续 target）。',
  visual: { icon: 'i-tabler-square', bg: 'bg-rose-500/15', border: 'border-rose-500/40' },
  execIn: ['in'],
  execOut: [],
  dataIn: {},
  dataOut: {},
  fields: [{ key: 'error', label: '错误信息(可选)', type: 'text' }],
  defaults: {},
})

register({
  kind: 'Sleep',
  group: 'control',
  labelZh: '休眠',
  description: '阻塞 durationMs 毫秒。可被强停打断。',
  visual: { icon: 'i-tabler-clock', bg: 'bg-zinc-500/15', border: 'border-zinc-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { durationMs: 'number' },
  dataOut: {},
  fields: [],
  defaults: { literal: { durationMs: 1000 } },
})

register({
  kind: 'Loop',
  group: 'control',
  labelZh: '循环',
  description: '循环执行 body 子图。mode: count（固定次数）/ while（条件）/ forever（无限，需 Break 退出）',
  visual: { icon: 'i-tabler-repeat', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  execIn: ['in', 'loopback'],
  execOut: ['body', 'complete'],
  dataIn: { count: 'number', condition: 'bool' },
  dataOut: { iter: 'number' },
  fields: [
    {
      key: 'mode',
      label: '模式',
      type: 'select',
      options: [
        { value: 'count', label: '固定次数 (count)' },
        { value: 'while', label: '条件循环 (while)' },
        { value: 'forever', label: '无限循环 (forever)' },
      ],
      hint: 'count → 读 data-in pin "count"; while → 读 data-in pin "condition"; forever → 都不读',
    },
  ],
  defaults: { mode: 'count', literal: { count: 10, condition: true } },
})

register({
  kind: 'If',
  group: 'control',
  labelZh: '条件',
  description: 'condition 为真走 then，否则走 else。',
  visual: { icon: 'i-tabler-git-branch', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  execIn: ['in'],
  execOut: ['then', 'else'],
  dataIn: { condition: 'bool' },
  dataOut: {},
  fields: [],
  defaults: { literal: { condition: true } },
})

register({
  kind: 'Switch',
  group: 'control',
  labelZh: '多路分支',
  description:
    '按 cases 数组中的字符串派生多路 exec-out。value 从 data-in pin 取 (连数据边或在"数据输入"段填字面量), 匹配某 case 走对应出口, 无匹配走 default。',
  visual: { icon: 'i-tabler-switch-3', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  execIn: ['in'],
  execOut: ['default'],
  execOutFn: (cfg) => {
    const cases = Array.isArray(cfg?.cases) ? (cfg!.cases as unknown[]) : []
    const out: string[] = []
    const seen = new Set<string>()
    for (const c of cases) {
      if (typeof c !== 'string' || c === '' || seen.has(c)) continue
      seen.add(c)
      out.push(c)
    }
    out.push('default')
    return out
  },
  dataIn: { value: 'string' },
  dataOut: {},
  fields: [],
  defaults: { cases: ['a', 'b'], literal: { value: '' } },
})

register({
  kind: 'Parallel',
  group: 'control',
  labelZh: '并行',
  description: 'spawn N 个分支并行跑，全部完成后走 complete。',
  visual: { icon: 'i-tabler-columns-3', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  execIn: ['in'],
  execOut: ['complete'],
  execOutFn: parallelBranchPins,
  dataIn: { n: 'number' },
  dataOut: {},
  fields: [],
  defaults: { literal: { n: 2 } },
})

register({
  kind: 'Race',
  group: 'control',
  labelZh: '竞速',
  description: 'spawn N 个分支，第一个 reach 终点的胜出，其它被取消。$sys.winnerIdx 拿到胜出 index。',
  visual: { icon: 'i-tabler-flag', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  execIn: ['in'],
  execOut: ['complete'],
  execOutFn: parallelBranchPins,
  dataIn: { n: 'number' },
  dataOut: { winnerIdx: 'number' },
  fields: [],
  defaults: { literal: { n: 2 } },
})

register({
  kind: 'Break',
  group: 'control',
  labelZh: '跳出循环',
  description: '跳出最近的 Loop（走该 Loop 的 complete pin）。必须在 Loop body 子图里。',
  visual: { icon: 'i-tabler-player-skip-forward', bg: 'bg-rose-500/15', border: 'border-rose-500/40' },
  execIn: ['in'],
  execOut: [],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: {},
})

register({
  kind: 'Continue',
  group: 'control',
  labelZh: '跳到下轮',
  description: '回到最近 Loop 头开始下一轮迭代。必须在 Loop body 子图里。',
  visual: { icon: 'i-tabler-corner-down-left', bg: 'bg-rose-500/15', border: 'border-rose-500/40' },
  execIn: ['in'],
  execOut: [],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: {},
})
