// frontend/src/components/containers/nodeRegistry/specs/system.ts
// System / misc nodes (15): BringGameForeground, MouseCalibration, WindowTarget, CommentBox,
//   Try, Throw, Stopwatch{Start,Stop,Read}, Log, Toast,
//   Subgraph, SubgraphInput, SubgraphOutput, CollapsedNode.
// Mirrors backend internal/services/container/nodekind/specs/system.go.
import { register } from '../registry'

register({
  kind: 'BringGameForeground',
  group: 'system',
  labelZh: '游戏置前',
  description:
    '把当前检测到的游戏窗口置于前台。会自动重试 3 次（每次间隔 50ms），覆盖全屏独占 / 反作弊场景的偶发失败。',
  visual: { icon: 'i-tabler-app-window', bg: 'bg-cyan-500/15', border: 'border-cyan-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: {},
})

register({
  kind: 'MouseCalibration',
  group: 'system',
  labelZh: '鼠标校准',
  description:
    '声明本容器主图依赖鼠标校准值。config.counts360 持有本机转 360° 累积 HID counts。运行时 MouseMoveRel 根据该值缩放。仅允许放在主图。',
  visual: { icon: 'i-tabler-target', bg: 'bg-yellow-500/15', border: 'border-yellow-500/40' },
  execIn: [],
  execOut: [],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { counts360: 0 },
})

register({
  kind: 'WindowTarget',
  group: 'system',
  labelZh: '目标窗口',
  description: '声明 container 操作的目标游戏窗口 + input/capture backend. 必须在主图, 1 个.',
  visual: { icon: 'i-tabler-app-window', bg: 'bg-sky-500/15', border: 'border-sky-500/40' },
  execIn: [],
  execOut: [],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: {
    match: { title: '', class: '', processName: '', titleMatch: 'exact' },
    runtime: { inputBackend: 'postmessage', captureBackend: 'auto' },
  },
})

register({
  kind: 'CommentBox',
  group: 'system',
  labelZh: '注释框',
  description: '',
  visual: { icon: 'i-tabler-square-letter-c', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: [],
  execOut: [],
  dataIn: {},
  dataOut: {},
  fields: [
    { key: 'label', label: '标题', type: 'text', placeholder: '注释' },
    { key: 'color', label: '颜色 (hex)', type: 'text', placeholder: '#fbbf24' },
    { key: 'width', label: '宽度 (px)', type: 'text', placeholder: '200' },
    { key: 'height', label: '高度 (px)', type: 'text', placeholder: '150' },
  ],
  defaults: { label: '注释', color: '#fbbf24', width: 200, height: 150 },
  isVisualOnly: true,
})

register({
  kind: 'Try',
  group: 'system',
  labelZh: '错误捕获子图',
  description:
    '跑 subgraphId 子图, 三路出口: 正常完成走 done, 超时走 timeout, 子图内节点 Throw 或 Backend 异常走 error (errorMsg 数据口拿消息).',
  visual: { icon: 'i-tabler-shield-exclamation', bg: 'bg-red-500/15', border: 'border-red-500/40' },
  execIn: ['in'],
  execOut: ['done', 'timeout', 'error'],
  dataIn: { timeoutMs: 'number' },
  dataOut: { errorMsg: 'string' },
  fields: [],
  defaults: { subgraphId: '', literal: { timeoutMs: 0 } },
})

register({
  kind: 'Throw',
  group: 'system',
  labelZh: '抛出错误',
  description:
    '立刻触发最近 Try 的 error 出口, 携带 message (data-in pin 字符串字面量, 不解析占位符 — 想插变量先用 Concat 节点拼).',
  visual: { icon: 'i-tabler-bolt', bg: 'bg-red-500/15', border: 'border-red-500/40' },
  execIn: ['in'],
  execOut: [],
  dataIn: { message: 'string' },
  dataOut: {},
  fields: [],
  defaults: { literal: { message: '' } },
})

register({
  kind: 'StopwatchStart',
  group: 'system',
  labelZh: '计时开始',
  description: '按 key 启动/重置计时器. key 命名空间独立于变量表, 防与 SetVar 重名.',
  visual: { icon: 'i-tabler-player-play', bg: 'bg-zinc-500/15', border: 'border-zinc-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { key: 'default' },
})

register({
  kind: 'StopwatchStop',
  group: 'system',
  labelZh: '计时停止',
  description: '按 key 停止计时器 (key 不存在静默 warn, 不报错).',
  visual: { icon: 'i-tabler-player-stop', bg: 'bg-zinc-500/15', border: 'border-zinc-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { key: 'default' },
})

register({
  kind: 'StopwatchRead',
  group: 'system',
  labelZh: '计时读取',
  description: '按 key 读取经过毫秒数到 elapsedMs 数据口. running → now-start; stopped → stop-start.',
  visual: { icon: 'i-tabler-stopwatch', bg: 'bg-zinc-500/15', border: 'border-zinc-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: { elapsedMs: 'number' },
  fields: [],
  defaults: { key: 'default' },
})

register({
  kind: 'Log',
  group: 'system',
  labelZh: '日志',
  description: 'message 从 data-in pin 取 (可连数据边或填字面量字符串), 写到运行日志面板。level 控制颜色。',
  visual: { icon: 'i-tabler-file-text', bg: 'bg-slate-500/15', border: 'border-slate-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { message: 'any' },
  dataOut: {},
  fields: [
    {
      key: 'level',
      label: '级别',
      type: 'select',
      options: [
        { value: 'info', label: 'info' },
        { value: 'warn', label: 'warn' },
        { value: 'error', label: 'error' },
      ],
    },
  ],
  defaults: { level: 'info', literal: { message: 'hello' } },
})

register({
  kind: 'Toast',
  group: 'system',
  labelZh: '弹窗提示',
  description: 'title / message 都是 data-in pin (可连数据边或填字面量)。前端订阅 container:toast 显示。',
  visual: { icon: 'i-tabler-bell', bg: 'bg-slate-500/15', border: 'border-slate-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { title: 'any', message: 'any' },
  dataOut: {},
  fields: [
    {
      key: 'color',
      label: '颜色',
      type: 'select',
      options: [
        { value: 'neutral', label: '中性 (neutral)' },
        { value: 'primary', label: '主色 (primary)' },
        { value: 'success', label: '成功 (success)' },
        { value: 'warning', label: '警告 (warning)' },
        { value: 'error', label: '错误 (error)' },
      ],
    },
  ],
  defaults: { color: 'primary', literal: { title: '提示', message: '' } },
})

// Subgraph call node — exec-out + data-in are dynamic, derived from the called
// subgraph's OutputPins / InputParams. Backend Spec leaves them empty; callers must
// special-case kind === 'Subgraph' to resolve pins.
register({
  kind: 'Subgraph',
  group: 'system',
  labelZh: '子图',
  description: '调用一个子图（容器内已存在的 Subgraph）。双击该节点进入子图编辑。',
  visual: { icon: 'i-tabler-package', bg: 'bg-fuchsia-500/15', border: 'border-fuchsia-500/40' },
  execIn: ['in'],
  execOut: [], // dynamic — caller queries subgraph.outputPins
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { subgraphId: '' },
})

register({
  kind: 'SubgraphInput',
  group: 'system',
  labelZh: '子图入口',
  description: '子图唯一入口节点。每个子图自动有一个，不可删除。',
  visual: {
    icon: 'i-tabler-arrow-bar-to-right',
    bg: 'bg-emerald-500/15',
    border: 'border-emerald-500/40',
  },
  execIn: [],
  execOut: ['out'],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: {},
})

register({
  kind: 'SubgraphOutput',
  group: 'system',
  labelZh: '子图出口',
  description:
    '子图命名出口。declID 引用所属子图 OutputPins 中的 Decl ID；父图的 Subgraph 调用节点会以此派生 exec-out pin。',
  visual: {
    icon: 'i-tabler-arrow-bar-to-left',
    bg: 'bg-rose-500/15',
    border: 'border-rose-500/40',
  },
  execIn: ['in'],
  execOut: [],
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { declID: '' },
})

register({
  kind: 'CollapsedNode',
  group: 'system',
  labelZh: '折叠节点',
  description: '',
  visual: { icon: 'i-tabler-fold', bg: 'bg-fuchsia-500/15', border: 'border-fuchsia-500/40' },
  execIn: ['in'],
  execOut: [], // dynamic — same as Subgraph (bound to anonymous backing subgraph)
  dataIn: {},
  dataOut: {},
  fields: [],
  defaults: { subgraphId: '', label: 'Collapsed' },
})
