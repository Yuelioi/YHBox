// frontend/src/components/containers/nodeRegistry/specs/variables.ts
// Variable / sys / param accessors (5): SetVar, IncVar, GetVar, GetSys, GetParam.
// Mirrors backend internal/services/container/nodekind/specs/variables.go.
import { register } from '../registry'

// 2026-05-19 默认 scope 改成 "auto" — local 没 UI 声明入口, 默认指向它会让容器变量
// 面板里声明的 var 在 GetVar 读不到 (反直觉). auto = 优先 frame.LocalVars, 命中不到
// fallback 容器变量, 跟用户对"变量面板里的就是默认变量"的认知一致.
const SCOPE_OPTIONS = [
  { value: 'auto', label: 'auto (默认, frame chain → 容器变量)' },
  { value: 'global', label: 'global (强制容器变量, 跳过 frame chain)' },
  { value: 'local', label: 'local (frame 私有, 子图隔离)' },
]

register({
  kind: 'SetVar',
  group: 'variables',
  labelZh: '赋值变量',
  description:
    '把 value (从下方 data-in pin 拿) 写到 varName。scope=auto (默认): 当前 frame 已有 → 更新 local; 否则写容器变量。global=强制容器变量; local=frame 私有 (子图隔离).',
  visual: { icon: 'i-tabler-equal', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { value: 'any' },
  dataOut: {},
  fields: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS, hint: 'auto=按是否已 local 选; global=容器变量; local=frame 私有' },
  ],
  defaults: { varName: '', scope: 'auto', literal: { value: '' } },
})

register({
  kind: 'IncVar',
  group: 'variables',
  labelZh: '递增变量',
  description: '把 varName 的值加 delta (data-in pin, 默认 1)。仅 number 变量。scope 同 SetVar (默认 auto)。',
  visual: { icon: 'i-tabler-circle-plus', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { delta: 'number' },
  dataOut: {},
  fields: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS, hint: 'auto=按是否已 local 选; global=容器变量; local=frame 私有' },
  ],
  defaults: { varName: '', scope: 'auto', literal: { delta: 1 } },
})

register({
  kind: 'GetVar',
  group: 'variables',
  labelZh: '读取变量',
  description:
    '读取 varName 的当前值, 通过 data-out pin 暴露给下游 (Expr / 纯函数 / SetVar 等)。scope=auto (默认): frame.LocalVars → 容器变量 fallback. global=只读容器变量; local=只读当前 frame.',
  visual: { icon: 'i-tabler-variable', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: [],
  execOut: [],
  dataIn: {},
  dataOut: { value: 'any' },
  fields: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS, hint: 'auto=frame chain → 容器变量; global=容器变量; local=只读 frame' },
  ],
  defaults: { varName: '', scope: 'auto' },
  isPureData: true,
})

register({
  kind: 'GetSys',
  group: 'variables',
  labelZh: '系统状态',
  description: '从 $sys 系统快照读字段 (e.g. iter, lastTemplate.point, winnerIdx)。纯数据节点, 通过 data-out pin 输出。',
  visual: { icon: 'i-tabler-cpu', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: [],
  execOut: [],
  dataIn: {},
  dataOut: { value: 'any' },
  fields: [
    {
      key: 'path',
      label: '系统路径',
      type: 'select',
      options: [
        { value: 'runId', label: 'runId (number)' },
        { value: 'iter', label: 'iter (number) — Loop 当前 iter' },
        { value: 'winnerIdx', label: 'winnerIdx (number) — Race 获胜分支' },
        { value: 'lastTemplate.found', label: 'lastTemplate.found (bool)' },
        { value: 'lastTemplate.point', label: 'lastTemplate.point (point)' },
        { value: 'lastTemplate.point.x', label: 'lastTemplate.point.x (number)' },
        { value: 'lastTemplate.point.y', label: 'lastTemplate.point.y (number)' },
        { value: 'lastTemplate.region', label: 'lastTemplate.region (any)' },
        { value: 'lastColor.count', label: 'lastColor.count (number)' },
        { value: 'lastColor.cx', label: 'lastColor.cx (number)' },
        { value: 'lastColor.cy', label: 'lastColor.cy (number)' },
        { value: 'lastColor.center', label: 'lastColor.center (point)' },
        { value: 'lastColor.center.x', label: 'lastColor.center.x (number)' },
        { value: 'lastColor.center.y', label: 'lastColor.center.y (number)' },
        { value: 'lastStopwatch.elapsedMs', label: 'lastStopwatch.elapsedMs (number)' },
        { value: 'lastTry.errorMsg', label: 'lastTry.errorMsg (string)' },
        { value: 'lastDetect.pixelCount', label: 'lastDetect.pixelCount (number)' },
        { value: 'lastDetect.pixelRatio', label: 'lastDetect.pixelRatio (number)' },
        { value: 'lastROIScan.clusterCount', label: 'lastROIScan.clusterCount (number)' },
        { value: 'lastROIScan.clusters', label: 'lastROIScan.clusters (any)' },
        { value: 'lastScreenshot.path', label: 'lastScreenshot.path (string)' },
      ],
      hint: '运行时 sys 状态 — 同一 tick 内一致 (per-tick snapshot)',
    },
  ],
  defaults: { path: '' },
  isPureData: true,
})

register({
  kind: 'GetParam',
  group: 'variables',
  labelZh: '子图入参',
  description: '读取当前子图的入参 (Subgraph 调用时传入). paramName 必须在该子图 inputParams 中声明。纯数据节点。',
  visual: { icon: 'i-tabler-input-search', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: [],
  execOut: [],
  dataIn: {},
  dataOut: { value: 'any' },
  fields: [
    { key: 'paramName', label: '参数名 (paramName)', type: 'text', placeholder: '当前子图 inputParams 里的 name' },
  ],
  defaults: { paramName: '' },
  isPureData: true,
})
