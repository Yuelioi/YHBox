// Per-kind pin metadata。镜像 Go 端 internal/services/container/validate.go 的
// execInPins / execOutPins / dataOutPins / dataInPins。
//
// 用途：
//   1. Vue Flow 自定义节点根据 kind 渲染对应 handle
//   2. UI 上画出 exec edge vs data edge 不同样式
//   3. 帮助连线时校验 from/to pin 是否合法

export type PinDataType = 'point' | 'number' | 'any'

export interface PinSpec {
  execIn: string[] // 默认 ['in']
  execOut: string[] // 默认 ['out']
  dataIn: Record<string, PinDataType> // 默认空
  dataOut: Record<string, PinDataType> // 默认空
  /** Parallel/Race 分支数动态：branch0..branchN-1 由 config.n 决定 */
  dynamicBranches?: boolean
  /** InvokeAction params.* 动态：根据被调 Action.params 生成 */
  dynamicParams?: boolean
}

const DEFAULT_IN: string[] = ['in']
const DEFAULT_OUT: string[] = ['out']

export const PIN_SPECS: Record<string, PinSpec> = {
  Start: { execIn: [], execOut: ['out'], dataIn: {}, dataOut: {} },
  Sleep: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Loop: {
    execIn: ['in', 'loopback'],
    execOut: ['body', 'complete'],
    dataIn: {},
    dataOut: { iter: 'number' },
  },
  If: { execIn: DEFAULT_IN, execOut: ['then', 'else'], dataIn: {}, dataOut: {} },
  Parallel: {
    execIn: DEFAULT_IN,
    execOut: ['complete'],
    dataIn: {},
    dataOut: {},
    dynamicBranches: true,
  },
  Race: {
    execIn: DEFAULT_IN,
    execOut: ['complete'],
    dataIn: {},
    dataOut: { winnerIdx: 'number' },
    dynamicBranches: true,
  },
  Stop: { execIn: DEFAULT_IN, execOut: [], dataIn: {}, dataOut: {} },
  Break: { execIn: DEFAULT_IN, execOut: [], dataIn: {}, dataOut: {} },
  Continue: { execIn: DEFAULT_IN, execOut: [], dataIn: {}, dataOut: {} },
  SetVar: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  IncVar: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  WaitTemplate: {
    execIn: DEFAULT_IN,
    execOut: ['found', 'timeout'],
    dataIn: {},
    dataOut: { point: 'point' },
  },
  CheckTemplate: {
    execIn: DEFAULT_IN,
    execOut: ['yes', 'no'],
    dataIn: {},
    dataOut: { point: 'point' },
  },
  ClickTemplate: {
    execIn: DEFAULT_IN,
    execOut: ['done', 'timeout'],
    dataIn: {},
    dataOut: { point: 'point' },
  },
  DetectColor: {
    execIn: DEFAULT_IN,
    execOut: ['yes', 'no'],
    dataIn: {},
    dataOut: { point: 'point' },
  },
  InvokeAction: {
    execIn: DEFAULT_IN,
    execOut: DEFAULT_OUT,
    dataIn: {},
    dataOut: {},
    dynamicParams: true,
  },
  ClickAt: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: { pos: 'point' }, dataOut: {} },
  KeyPress: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  MouseMoveRel: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Scroll: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  OnEvent: { execIn: [], execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Log: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Toast: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
}

export function pinsFor(
  kind: string,
  configN?: number,
): { execIn: string[]; execOut: string[]; dataIn: string[]; dataOut: string[] } {
  const s = PIN_SPECS[kind] ?? { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} }
  let execOut = [...s.execOut]
  if (s.dynamicBranches) {
    const n = Math.max(2, Math.min(8, configN ?? 2))
    const branches: string[] = []
    for (let i = 0; i < n; i++) branches.push(`branch${i}`)
    execOut = [...branches, ...execOut]
  }
  return {
    execIn: s.execIn,
    execOut,
    dataIn: Object.keys(s.dataIn),
    dataOut: Object.keys(s.dataOut),
  }
}

/** 边类型（exec 还是 data），由 from-pin 决定。 */
export function edgeKind(fromKind: string, fromPin: string): 'exec' | 'data' {
  const s = PIN_SPECS[fromKind]
  if (!s) return 'exec'
  if (s.dataOut[fromPin]) return 'data'
  return 'exec'
}

// 节点中文显示名（面向用户的标签）。kind 字符串保持英文（schema 稳定）。
export const KIND_LABEL_ZH: Record<string, string> = {
  Start: '开始',
  Sleep: '休眠',
  Loop: '循环',
  If: '条件',
  Parallel: '并行',
  Race: '竞速',
  Stop: '结束',
  Break: '跳出循环',
  Continue: '跳到下轮',
  SetVar: '赋值变量',
  IncVar: '递增变量',
  WaitTemplate: '等待图像',
  CheckTemplate: '试探图像',
  ClickTemplate: '点击图像',
  DetectColor: '检测颜色',
  InvokeAction: '调用动作',
  ClickAt: '点击位置',
  KeyPress: '按键',
  MouseMoveRel: '相对移动',
  Scroll: '滚轮',
  OnEvent: '事件监听',
  Log: '日志',
  Toast: '弹窗提示',
}

// 节点用法说明，选中节点时显示在 Inspector 顶部
export const KIND_DESCRIPTION: Record<string, string> = {
  Start: '容器唯一入口。每个容器必须有且仅有一个 Start。',
  Sleep: '阻塞 durationMs 毫秒。可被强停打断。',
  Loop: '循环执行 body 子图。mode: count（固定次数）/ while（条件）/ forever（无限，需 Break 退出）',
  If: 'condition 为真走 then，否则走 else。',
  Parallel: 'spawn N 个分支并行跑，全部完成后走 complete。',
  Race: 'spawn N 个分支，第一个 reach 终点的胜出，其它被取消。$sys.winnerIdx 拿到胜出 index。',
  Stop: '立即结束当前容器（不影响外层 schedule 后续 target）。',
  Break: '跳出最近的 Loop（走该 Loop 的 complete pin）。必须在 Loop body 子图里。',
  Continue: '回到最近 Loop 头开始下一轮迭代。必须在 Loop body 子图里。',
  SetVar: '把 value expr 求值结果写到 varName。变量需先在容器属性面板声明。',
  IncVar: '把 varName 的值加 delta（默认 1）。仅 number 变量。',
  WaitTemplate:
    '周期截屏检测模板图像。命中 → found 出口（产 $sys.lastTemplate.point）；超时 → timeout 出口。',
  CheckTemplate: '单次检测，立即分支：命中走 yes，否则走 no。不阻塞等待。',
  ClickTemplate: '检测到模板图像后点击其中心。命中并点完走 done，超时走 timeout。',
  DetectColor:
    'ROI 内统计落在 HSV/RGB 范围内的像素。≥ minPixels 走 yes，否则 no。结果写 $sys.lastColor.{count, cx, cy}。',
  InvokeAction: '调用已录制的 Action（一段鼠键序列）。等待完成后走 out。',
  ClickAt: '点击客户区比例坐标 (xRatio, yRatio)。durationMs 是按下时长。',
  KeyPress: '按下并松开 vk（虚拟键码字符串如 "W"/"Space"），间隔 durationMs。',
  MouseMoveRel: '相对当前位置移动 (dx, dy) 像素，移动用时 durationMs。',
  Scroll: '在 (xRatio, yRatio) 滚动 delta 格。正数向上，负数向下。',
  OnEvent:
    '与 Start 同级的入口节点。容器启动时起 listener；事件命中 spawn 子图。v1 仅支持 template_appeared。',
  Log: '把 message expr 结果写到运行日志面板。level 控制颜色。',
  Toast: '弹一个 toast 通知。title / message 都是 expr，前端订阅显示。',
}

// 节点默认 config。新建节点时填进去，避免用户面对空表单不知所措。
// 表达式字段（durationMs 等）都用字符串（运行时 parse）。
export const KIND_DEFAULTS: Record<string, Record<string, any>> = {
  Sleep: { durationMs: '1000' },
  Loop: { mode: 'count', count: '10' },
  If: { condition: 'true' },
  Parallel: { n: '2' },
  Race: { n: '2' },
  Stop: {},
  Break: {},
  Continue: {},
  SetVar: { varName: '', value: '0' },
  IncVar: { varName: '', delta: '1' },
  WaitTemplate: { template: '', timeoutMs: '5000', threshold: '0.85' },
  CheckTemplate: { template: '', threshold: '0.85' },
  ClickTemplate: { template: '', timeoutMs: '5000', threshold: '0.85', button: 'left' },
  DetectColor: {
    region: '0.4,0.55,0.2,0.05',
    mode: 'hsv',
    range: '50,60,67,127,253,255',
    minPixels: '5',
  },
  InvokeAction: { actionId: '' },
  ClickAt: { xRatio: '0.5', yRatio: '0.5', durationMs: '50', button: 'left' },
  KeyPress: { vk: 'W', durationMs: '50' },
  MouseMoveRel: { dx: '0', dy: '0', durationMs: '200' },
  Scroll: { xRatio: '0.5', yRatio: '0.5', delta: '3' },
  OnEvent: {
    kind: 'template_appeared',
    template: '',
    pollIntervalMs: '100',
    maxConcurrent: '1',
    retriggerPolicy: 'drop',
    cooldownMs: '0',
  },
  Log: { level: 'info', message: '"hello"' },
  Toast: { title: '"提示"', message: '""', color: 'primary' },
}

// 节点 kind → 显示用图标 + 颜色组（统一调色板）
export const KIND_VISUAL: Record<string, { icon: string; bg: string; border: string }> = {
  Start: { icon: 'i-tabler-player-play', bg: 'bg-emerald-500/15', border: 'border-emerald-500/40' },
  Sleep: { icon: 'i-tabler-clock', bg: 'bg-zinc-500/15', border: 'border-zinc-500/40' },
  Loop: { icon: 'i-tabler-repeat', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  If: { icon: 'i-tabler-git-branch', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  Parallel: { icon: 'i-tabler-columns-3', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  Race: { icon: 'i-tabler-flag', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  Stop: { icon: 'i-tabler-square', bg: 'bg-rose-500/15', border: 'border-rose-500/40' },
  Break: {
    icon: 'i-tabler-player-skip-forward',
    bg: 'bg-rose-500/15',
    border: 'border-rose-500/40',
  },
  Continue: {
    icon: 'i-tabler-corner-down-left',
    bg: 'bg-rose-500/15',
    border: 'border-rose-500/40',
  },
  SetVar: { icon: 'i-tabler-equal', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  IncVar: { icon: 'i-tabler-circle-plus', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  WaitTemplate: { icon: 'i-tabler-eye', bg: 'bg-violet-500/15', border: 'border-violet-500/40' },
  CheckTemplate: {
    icon: 'i-tabler-search',
    bg: 'bg-violet-500/15',
    border: 'border-violet-500/40',
  },
  ClickTemplate: {
    icon: 'i-tabler-target',
    bg: 'bg-violet-500/15',
    border: 'border-violet-500/40',
  },
  DetectColor: {
    icon: 'i-tabler-color-picker',
    bg: 'bg-cyan-500/15',
    border: 'border-cyan-500/40',
  },
  InvokeAction: {
    icon: 'i-tabler-movie',
    bg: 'bg-fuchsia-500/15',
    border: 'border-fuchsia-500/40',
  },
  ClickAt: { icon: 'i-tabler-click', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  KeyPress: { icon: 'i-tabler-keyboard', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  MouseMoveRel: {
    icon: 'i-tabler-arrows-move',
    bg: 'bg-orange-500/15',
    border: 'border-orange-500/40',
  },
  Scroll: { icon: 'i-tabler-mouse', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  OnEvent: { icon: 'i-tabler-radio', bg: 'bg-pink-500/15', border: 'border-pink-500/40' },
  Log: { icon: 'i-tabler-file-text', bg: 'bg-slate-500/15', border: 'border-slate-500/40' },
  Toast: { icon: 'i-tabler-bell', bg: 'bg-slate-500/15', border: 'border-slate-500/40' },
}
