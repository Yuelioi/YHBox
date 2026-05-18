// Per-kind pin metadata。镜像 Go 端 internal/services/container/validate.go 的
// execInPins / execOutPins / dataOutPins / dataInPins。
//
// 用途：
//   1. Vue Flow 自定义节点根据 kind 渲染对应 handle
//   2. UI 上画出 exec edge vs data edge 不同样式
//   3. 帮助连线时校验 from/to pin 是否合法

// v4: 5-type data pin system (matches backend internal/services/container/runtime/pin_types.go)
export type PinDataType = 'number' | 'bool' | 'string' | 'point' | 'any'
/** v4 alias used by GetVar/SetVar/Expr/pure-func node specs */
export type PinType = PinDataType

/** v4 color map for typed data pins (spec §2.2). vue-flow Handle uses this for background. */
export const TYPE_COLOR: Record<PinType, string> = {
  number: '#60a5fa', // blue
  bool: '#f87171', // red
  string: '#a78bfa', // purple
  point: '#34d399', // green
  any: '#9ca3af', // gray
}

/** v4 data pin descriptor returned by dataInFn / dataOutFn (preserves type info for color + validator). */
export interface DataPinSpec {
  name: string
  type: PinType
}

/**
 * v4 pin type compatibility (spec §2.3) — mirrors backend runtime.PinTypeCompat.
 * @returns allow=can connect, warn=allowed but coerced (UI gives hint)
 */
export function pinTypeCompat(from: PinType, to: PinType): { allow: boolean; warn: boolean } {
  if (from === to || from === 'any' || to === 'any') return { allow: true, warn: false }
  if (from === 'number' && (to === 'bool' || to === 'string')) return { allow: true, warn: true }
  if (from === 'bool' && (to === 'number' || to === 'string')) return { allow: true, warn: true }
  return { allow: false, warn: false }
}

export interface PinSpec {
  execIn: string[] // 默认 ['in']
  execOut: string[] // 静态 fallback if execOutFn 缺
  dataIn: Record<string, PinDataType> // 默认空
  dataOut: Record<string, PinDataType> // 默认空
  /** 按节点 config 动态派生 exec out pins. 优先级高于 execOut. */
  execOutFn?: (config: Record<string, unknown> | null | undefined) => string[]
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
  Switch: {
    execIn: DEFAULT_IN,
    execOut: ['default'], // fallback if execOutFn 未运行
    dataIn: {},
    dataOut: {},
    execOutFn: (config) => {
      const cases = Array.isArray(config?.cases) ? config.cases : []
      const out: string[] = []
      const seen = new Set<string>()
      for (const c of cases) {
        if (typeof c !== 'string' || c === '') continue
        if (seen.has(c)) continue
        seen.add(c)
        out.push(c)
      }
      out.push('default')
      return out
    },
  },
  Parallel: {
    execIn: DEFAULT_IN,
    execOut: ['complete'],
    dataIn: {},
    dataOut: {},
    execOutFn: (config) => {
      const raw = config?.n
      const parsed = Number(raw)
      const n = Math.max(2, Math.min(8, Number.isNaN(parsed) || parsed <= 0 ? 2 : parsed))
      const out: string[] = []
      for (let i = 0; i < n; i++) out.push(`branch${i}`)
      out.push('complete')
      return out
    },
  },
  Race: {
    execIn: DEFAULT_IN,
    execOut: ['complete'],
    dataIn: {},
    dataOut: { winnerIdx: 'number' },
    execOutFn: (config) => {
      const raw = config?.n
      const parsed = Number(raw)
      const n = Math.max(2, Math.min(8, Number.isNaN(parsed) || parsed <= 0 ? 2 : parsed))
      const out: string[] = []
      for (let i = 0; i < n; i++) out.push(`branch${i}`)
      out.push('complete')
      return out
    },
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
  ClickAt: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: { pos: 'point' }, dataOut: {} },
  KeyPress: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  MouseMoveRel: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Scroll: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  OnEvent: { execIn: [], execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Log: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Toast: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Subgraph: {
    execIn: DEFAULT_IN,
    execOut: ['__placeholder__'],
    dataIn: {},
    dataOut: {},
  },
  SubgraphInput: {
    execIn: [],
    execOut: ['out'],
    dataIn: {},
    dataOut: {},
  },
  SubgraphOutput: {
    execIn: DEFAULT_IN,
    execOut: [],
    dataIn: {},
    dataOut: {},
  },
  BringGameForeground: {
    execIn: DEFAULT_IN,
    execOut: DEFAULT_OUT,
    dataIn: {},
    dataOut: {},
  },
  MouseCalibration: {
    execIn: [],
    execOut: [],
    dataIn: {},
    dataOut: {},
  },
  WindowTarget: {
    execIn: [],
    execOut: [],
    dataIn: {},
    dataOut: {},
  },
  PlayClip: {
    execIn: ['in'],
    execOut: ['out'],
    dataIn: {},
    dataOut: {},
  },
  // v3 Phase C
  DetectColorHSV: {
    execIn: DEFAULT_IN,
    execOut: ['yes', 'no', 'timeout'],
    dataIn: {},
    dataOut: { pixelCount: 'number', pixelRatio: 'number' },
  },
  ROIColorScan: {
    execIn: DEFAULT_IN,
    execOut: ['found', 'notFound', 'timeout'],
    dataIn: {},
    dataOut: { clusters: 'any', clusterCount: 'number' },
  },
  Screenshot: {
    execIn: DEFAULT_IN,
    execOut: ['done'],
    dataIn: {},
    dataOut: { path: 'string' },
  },
  KeyHoldStart: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  KeyHoldStop: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  MouseHoldStart: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  MouseHoldStop: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  Try: {
    execIn: DEFAULT_IN,
    execOut: ['done', 'timeout', 'error'],
    dataIn: {},
    dataOut: { errorMsg: 'string' },
  },
  Throw: { execIn: DEFAULT_IN, execOut: [], dataIn: {}, dataOut: {} },
  StopwatchStart: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  StopwatchStop: { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} },
  StopwatchRead: {
    execIn: DEFAULT_IN,
    execOut: DEFAULT_OUT,
    dataIn: {},
    dataOut: { elapsedMs: 'number' },
  },
}

export function pinsFor(
  kind: string,
  config?: Record<string, unknown> | null,
): { execIn: string[]; execOut: string[]; dataIn: string[]; dataOut: string[] } {
  const s = PIN_SPECS[kind] ?? { execIn: DEFAULT_IN, execOut: DEFAULT_OUT, dataIn: {}, dataOut: {} }
  const execOut = execOutPinsFor(kind, config)
  // Subgraph 调用节点：exec-out 由被调子图的 OutputPins 动态派生
  // 这里没法直接拿到 subgraph 数据；真实派生交给 ContainerFlowNode.vue 拿到
  // nodes/edges 上下文时执行（Subgraph 保留 __placeholder__ 占位）。
  return {
    execIn: s.execIn,
    execOut,
    dataIn: Object.keys(s.dataIn),
    dataOut: Object.keys(s.dataOut),
  }
}

/**
 * 计算节点的 exec out pin 集合.
 * 优先用 spec.execOutFn (动态), fallback spec.execOut (静态), 最后 ['out'].
 *
 * 替代旧的 spec.dynamicBranches 分支判断 — Switch / Parallel / Race 全走 execOutFn.
 */
export function execOutPinsFor(kind: string, config: Record<string, unknown> | null | undefined): string[] {
  const spec = PIN_SPECS[kind]
  if (!spec) return DEFAULT_OUT
  if (spec.execOutFn) return spec.execOutFn(config)
  return spec.execOut.length > 0 ? spec.execOut : DEFAULT_OUT
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
  Switch: '多路分支',
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
  ClickAt: '点击位置',
  KeyPress: '按键',
  MouseMoveRel: '相对移动',
  Scroll: '滚轮',
  OnEvent: '事件监听',
  Log: '日志',
  Toast: '弹窗提示',
  Subgraph: '子图',
  SubgraphInput: '子图入口',
  SubgraphOutput: '子图出口',
  BringGameForeground: '游戏置前',
  MouseCalibration: '鼠标校准',
  WindowTarget: '目标窗口',
  PlayClip: '播放录制',
  // v3 Phase C
  DetectColorHSV: 'HSV 颜色检测',
  ROIColorScan: 'ROI 颜色扫描',
  Screenshot: '截图存盘',
  KeyHoldStart: '按下键 (异步)',
  KeyHoldStop: '松开键',
  MouseHoldStart: '按下鼠标 (异步)',
  MouseHoldStop: '松开鼠标',
  Try: '错误捕获子图',
  Throw: '抛出错误',
  StopwatchStart: '计时开始',
  StopwatchStop: '计时停止',
  StopwatchRead: '计时读取',
}

// 节点用法说明，选中节点时显示在 Inspector 顶部
export const KIND_DESCRIPTION: Record<string, string> = {
  Start: '容器唯一入口。每个容器必须有且仅有一个 Start。',
  Sleep: '阻塞 durationMs 毫秒。可被强停打断。',
  Loop: '循环执行 body 子图。mode: count（固定次数）/ while（条件）/ forever（无限，需 Break 退出）',
  If: 'condition 为真走 then，否则走 else。',
  Switch: '按 cases 数组中的字符串派生多路 exec-out。value expr 匹配某 case 走对应出口，无匹配走 default。',
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
  ClickAt: '点击客户区比例坐标 (xRatio, yRatio)。durationMs 是按下时长。',
  KeyPress: '按下并松开 vk（虚拟键码字符串如 "W"/"Space"），间隔 durationMs。',
  MouseMoveRel: '相对当前位置移动 (dx, dy) 像素，移动用时 durationMs。',
  Scroll: '在 (xRatio, yRatio) 滚动 delta 格。正数向上，负数向下。',
  OnEvent:
    '与 Start 同级的入口节点。容器启动时起 listener；事件命中 spawn 子图。v1 仅支持 template_appeared。',
  Log: '把 message expr 结果写到运行日志面板。level 控制颜色。',
  Toast: '弹一个 toast 通知。title / message 都是 expr，前端订阅显示。',
  Subgraph: '调用一个子图（容器内已存在的 Subgraph）。双击该节点进入子图编辑。',
  SubgraphInput: '子图唯一入口节点。每个子图自动有一个，不可删除。',
  SubgraphOutput:
    '子图命名出口。declID 引用所属子图 OutputPins 中的 Decl ID；父图的 Subgraph 调用节点会以此派生 exec-out pin。',
  BringGameForeground:
    '把当前检测到的游戏窗口置于前台。会自动重试 3 次（每次间隔 50ms），覆盖全屏独占 / 反作弊场景的偶发失败。',
  MouseCalibration:
    '声明本容器主图依赖鼠标校准值。config.counts360 持有本机转 360° 累积 HID counts。运行时 MouseMoveRel 根据该值缩放。仅允许放在主图。',
  WindowTarget:
    '声明 container 操作的目标游戏窗口 + input/capture backend. 必须在主图, 1 个.',
  PlayClip:
    '播放一段录制的输入事件流 (键鼠 events). config.clipID 指定 clip; config.keepRanges 可选裁剪 [{fromUs, toUs}, ...] 让你跳过录制中的停顿段.',
  // v3 Phase C
  DetectColorHSV:
    '按 HSV 范围在 ROI 中扫描像素, 命中比例 ≥ minPixelRatio 走 yes, 否则下次 poll; 超时走 timeout. 输出 pixelCount / pixelRatio.',
  ROIColorScan:
    'ROI 内沿 scanAxis 聚簇扫描色块, 输出 clusters 数组 (含 startPx/endPx/centerPx/pxCount). 用于溜鱼方向判断等场景.',
  Screenshot:
    '抓全帧或 ROI 帧, 按 pathTemplate 存 PNG (限定 bin/data/screenshots/ 下). 占位符: {ts}/{containerId}/{nodeId}/{date}.',
  KeyHoldStart: '按下虚拟键并立即返回 (异步长按). 配对 KeyHoldStop; 容器终止时 ReleaseAll 兜底.',
  KeyHoldStop: '松开虚拟键. 跟 KeyHoldStart 配对.',
  MouseHoldStart: '在指定客户区坐标 (x, y) 按下鼠标键并立即返回. 配对 MouseHoldStop.',
  MouseHoldStop: '松开鼠标键 (Backend stateful, 不需位置). 跟 MouseHoldStart 配对.',
  Try:
    '跑 subgraphId 子图, 三路出口: 正常完成走 done, 超时走 timeout, 子图内节点 Throw 或 Backend 异常走 error (errorMsg 数据口拿消息).',
  Throw: '立刻触发最近 Try 的 error 出口, 携带 message (模板字符串, 支持 {var:foo} 占位符).',
  StopwatchStart: '按 key 启动/重置计时器. key 命名空间独立于变量表, 防与 SetVar 重名.',
  StopwatchStop: '按 key 停止计时器 (key 不存在静默 warn, 不报错).',
  StopwatchRead: '按 key 读取经过毫秒数到 elapsedMs 数据口. running → now-start; stopped → stop-start.',
}

// 节点默认 config。新建节点时填进去，避免用户面对空表单不知所措。
// 表达式字段（durationMs 等）都用字符串（运行时 parse）。
export const KIND_DEFAULTS: Record<string, Record<string, any>> = {
  Sleep: { durationMs: '1000' },
  Loop: { mode: 'count', count: '10' },
  If: { condition: 'true' },
  Switch: { value: '$vars.state', cases: ['a', 'b'] },
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
  Subgraph: { subgraphId: '' },
  SubgraphInput: {},
  SubgraphOutput: { declID: '' },
  BringGameForeground: {},
  MouseCalibration: { counts360: 0 },
  WindowTarget: {
    match: { title: '', class: '', processName: '', titleMatch: 'exact' },
    runtime: { inputBackend: 'postmessage', captureBackend: 'auto' },
  },
  PlayClip: { clipID: '', keepRanges: [] },
  // v3 Phase C
  DetectColorHSV: {
    roi: { x: 0, y: 0, w: 100, h: 100 },
    hsv: { hMin: 0, hMax: 180, sMin: 0, sMax: 255, vMin: 0, vMax: 255 },
    minPixelRatio: 0.05,
    pollIntervalMs: 100,
    timeoutMs: 5000,
  },
  ROIColorScan: {
    roi: { x: 0, y: 0, w: 100, h: 100 },
    hsv: { hMin: 0, hMax: 180, sMin: 0, sMax: 255, vMin: 0, vMax: 255 },
    scanAxis: 'x',
    minClusterPx: 2,
    maxClusterPx: 0,
    minClusterCount: 1,
    pollIntervalMs: 100,
    timeoutMs: 5000,
  },
  Screenshot: { pathTemplate: 'screenshots/{ts}.png' },
  KeyHoldStart: { vk: 'A' },
  KeyHoldStop: { vk: 'A' },
  MouseHoldStart: { button: 'left', xRatio: '0.5', yRatio: '0.5' },
  MouseHoldStop: { button: 'left' },
  Try: { subgraphId: '', timeoutMs: 0 },
  Throw: { message: '' },
  StopwatchStart: { key: 'default' },
  StopwatchStop: { key: 'default' },
  StopwatchRead: { key: 'default' },
}

// 节点 kind → 显示用图标 + 颜色组（统一调色板）
// 每个 kind 用一种 tailwind accent (emerald/blue/rose/amber/violet/...) 作类别标识 —— 这是
// 跟 NuxtUI semantic token (default/elevated/...) 平行的"节点类别"色系, 不当主题色用.
// Sleep 用 zinc 表达"等待/中性"语义, ContainerEditorView.vue MiniMap 的 border→hex 映射会读它,
// 不要改成 semantic.
export const KIND_VISUAL: Record<string, { icon: string; bg: string; border: string }> = {
  Start: { icon: 'i-tabler-player-play', bg: 'bg-emerald-500/15', border: 'border-emerald-500/40' },
  Sleep: { icon: 'i-tabler-clock', bg: 'bg-zinc-500/15', border: 'border-zinc-500/40' },
  Loop: { icon: 'i-tabler-repeat', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  If: { icon: 'i-tabler-git-branch', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  Switch: { icon: 'i-tabler-switch-3', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
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
  Subgraph: { icon: 'i-tabler-package', bg: 'bg-fuchsia-500/15', border: 'border-fuchsia-500/40' },
  SubgraphInput: {
    icon: 'i-tabler-arrow-bar-to-right',
    bg: 'bg-emerald-500/15',
    border: 'border-emerald-500/40',
  },
  SubgraphOutput: {
    icon: 'i-tabler-arrow-bar-to-left',
    bg: 'bg-rose-500/15',
    border: 'border-rose-500/40',
  },
  BringGameForeground: {
    icon: 'i-tabler-app-window',
    bg: 'bg-cyan-500/15',
    border: 'border-cyan-500/40',
  },
  MouseCalibration: {
    icon: 'i-tabler-target',
    bg: 'bg-yellow-500/15',
    border: 'border-yellow-500/40',
  },
  WindowTarget: {
    icon: 'i-tabler-app-window',
    bg: 'bg-sky-500/15',
    border: 'border-sky-500/40',
  },
  PlayClip: { icon: 'i-tabler-vinyl', bg: 'bg-pink-500/15', border: 'border-pink-500/40' },
  // v3 Phase C — 检测沿用 cyan (跟 DetectColor 同组), 长按沿用 orange (跟 KeyPress/ClickAt 同组),
  // Try/Throw 用 red 系突出错误语义, Stopwatch 用 zinc 跟 Sleep 同"时间/中性"语义.
  DetectColorHSV: {
    icon: 'i-tabler-palette',
    bg: 'bg-cyan-500/15',
    border: 'border-cyan-500/40',
  },
  ROIColorScan: {
    icon: 'i-tabler-scan-eye',
    bg: 'bg-cyan-500/15',
    border: 'border-cyan-500/40',
  },
  Screenshot: {
    icon: 'i-tabler-camera',
    bg: 'bg-violet-500/15',
    border: 'border-violet-500/40',
  },
  KeyHoldStart: {
    icon: 'i-tabler-keyboard',
    bg: 'bg-orange-500/15',
    border: 'border-orange-500/40',
  },
  KeyHoldStop: {
    icon: 'i-tabler-keyboard-off',
    bg: 'bg-orange-500/15',
    border: 'border-orange-500/40',
  },
  MouseHoldStart: {
    icon: 'i-tabler-hand-click',
    bg: 'bg-orange-500/15',
    border: 'border-orange-500/40',
  },
  MouseHoldStop: {
    icon: 'i-tabler-hand-off',
    bg: 'bg-orange-500/15',
    border: 'border-orange-500/40',
  },
  Try: {
    icon: 'i-tabler-shield-exclamation',
    bg: 'bg-red-500/15',
    border: 'border-red-500/40',
  },
  Throw: {
    icon: 'i-tabler-bolt',
    bg: 'bg-red-500/15',
    border: 'border-red-500/40',
  },
  StopwatchStart: {
    icon: 'i-tabler-player-play',
    bg: 'bg-zinc-500/15',
    border: 'border-zinc-500/40',
  },
  StopwatchStop: {
    icon: 'i-tabler-player-stop',
    bg: 'bg-zinc-500/15',
    border: 'border-zinc-500/40',
  },
  StopwatchRead: {
    icon: 'i-tabler-stopwatch',
    bg: 'bg-zinc-500/15',
    border: 'border-zinc-500/40',
  },
}

/**
 * resolveSubgraphCallExecOut 给定 Subgraph 调用节点 + 被调子图的 OutputPins，
 * 返回该节点应该展示的 exec-out pin id 列表（用于 vue-flow handle key）。
 * 找不到子图 → 返 ['__missing__'] 让节点显示一个错误 handle。
 */
export function resolveSubgraphCallExecOut(
  node: { config?: { subgraphId?: string } },
  allSubgraphs: { id: string; outputPins: { id: string; name: string }[] }[],
): { id: string; name: string }[] {
  const sgID = node.config?.subgraphId ?? ''
  const sg = allSubgraphs.find((s) => s.id === sgID)
  if (!sg) return [{ id: '__missing__', name: '(子图未找到)' }]
  if (sg.outputPins.length === 0) return [{ id: '__empty__', name: '(无出口)' }]
  return sg.outputPins.map((p) => ({ id: p.id, name: p.name }))
}
