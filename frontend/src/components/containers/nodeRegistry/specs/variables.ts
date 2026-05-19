// frontend/src/components/containers/nodeRegistry/specs/variables.ts
// Variable / sys / param accessors (5): SetVar, IncVar, GetVar, GetSys, GetParam.
// Mirrors backend internal/services/container/nodekind/specs/variables.go.
import { register } from '../registry'

const SCOPE_OPTIONS = [
  { value: 'local', label: 'local (默认, 当前 frame 私有)' },
  { value: 'global', label: 'global (容器共享)' },
]
const SCOPE_OPTIONS_WITH_AUTO = [
  ...SCOPE_OPTIONS,
  { value: 'auto', label: 'auto (frame chain → global)' },
]

register({
  kind: 'SetVar',
  group: 'variables',
  labelZh: '赋值变量',
  description:
    '把 value (从下方 data-in pin 拿) 写到 varName。变量需先在容器属性面板声明。scope=local (默认) 写 frame 私有, global 跨 frame 共享。',
  visual: { icon: 'i-tabler-equal', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { value: 'any' },
  dataOut: {},
  fields: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS },
  ],
  defaults: { varName: '', scope: 'local', literal: { value: '' } },
})

register({
  kind: 'IncVar',
  group: 'variables',
  labelZh: '递增变量',
  description: '把 varName 的值加 delta (data-in pin, 默认 1)。仅 number 变量。scope 同 SetVar。',
  visual: { icon: 'i-tabler-circle-plus', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: { delta: 'number' },
  dataOut: {},
  fields: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS },
  ],
  defaults: { varName: '', scope: 'local', literal: { delta: 1 } },
})

register({
  kind: 'GetVar',
  group: 'variables',
  labelZh: '读取变量',
  description:
    '读取 varName 的当前值, 通过 data-out pin 暴露给下游 (Expr / 纯函数 / SetVar 等)。scope: local (默认, 当前 frame) / global (容器共享) / auto (frame chain → global)。',
  visual: { icon: 'i-tabler-variable', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: [],
  execOut: [],
  dataIn: {},
  dataOut: { value: 'any' },
  fields: [
    { key: 'varName', label: '变量名', type: 'var-name-select' },
    { key: 'scope', label: '作用域', type: 'select', options: SCOPE_OPTIONS_WITH_AUTO },
  ],
  defaults: { varName: '', scope: 'local' },
  isPureData: true,
})

register({
  kind: 'GetSys',
  group: 'variables',
  labelZh: '系统状态',
  description: '', // not in KIND_DESCRIPTION map — pinSpec.ts has no entry.
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
    },
  ],
  defaults: { path: '' },
  isPureData: true,
})

register({
  kind: 'GetParam',
  group: 'variables',
  labelZh: '子图入参',
  description: '',
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
