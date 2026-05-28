// frontend/src/components/containers/nodeRegistry/adapter.ts
//
// Backend node.Spec (RPC pull via NodeService.GetAllNodeSpecs) → 老 NodeKindSpec
// adapter. 替代手写 specs/*.ts 9 文件 1245 行.
//
// 设计:
//  - 字段 1:1 直接映射: kind / labelZh(=DisplayName) / description / isPureData / isVisualOnly
//  - exec/data pins: 从 InputSpec/OutputSpec Type==='Exec' vs 其他派生
//  - fields: 从 InputSpec.Widget.Kind 派生 FE FieldSchema (widget kind 1:1 复制)
//  - defaults: 从 InputSpec.Default 收集 (literal map)
//  - visual: 按 group 用 fallback mapping (没每节点装饰多样化, 后续 backend Spec
//    加 Visual 字段才能 fine-grain)
//  - execOutFn / dataInDynamicFn: 几个特殊 Kind hardcode (Switch/Parallel/Race/Expr/
//    Subgraph). 这些 dynamic 逻辑跟 backend validator 镜像, 必须人工保.
//
// app 入口 main.ts boot 时 await populateRegistryFromBackend() 把 RPC spec 注册到
// 老 byKind, 6 处 consumer (Palette / ContextMenu / NodePalette / NodeExplorerModal /
// FindReferencesModal / nodeFieldSchemas) 不变.

import { useNodeRegistryStore } from '@/stores/nodeRegistry'
import { __resetForTests, register } from './registry'
import type { FieldSchema, NodeGroup, NodeKindSpec, PinType } from './index'
import type { Spec, InputSpec, OutputSpec } from '@bindings/yhbox/internal/node'

// Backend Category → FE NodeGroup. backend 用 TitleCase, FE 历史 lowercase + 'variables'.
const GROUP_MAP: Record<string, NodeGroup> = {
  Control: 'control',
  Detect: 'detect',
  Input: 'input',
  System: 'system',
  Variable: 'variables',
  PureFunc: 'purefunc',
  IO: 'io',
  Stopwatch: 'stopwatch',
  Mock: 'mock',
  Test: 'test',
}

// 按 group fallback visual (icon + tailwind color). per-kind 装饰多样化目前不做,
// 真要 fine-grain 需 backend Spec 加 Visual{Icon,Color} 字段, 60 节点都填一遍.
const VISUAL_BY_GROUP: Record<NodeGroup, { icon: string; bg: string; border: string }> = {
  control: { icon: 'i-tabler-arrow-bounce', bg: 'bg-blue-500/15', border: 'border-blue-500/40' },
  variables: { icon: 'i-tabler-variable', bg: 'bg-cyan-500/15', border: 'border-cyan-500/40' },
  purefunc: { icon: 'i-tabler-math-function', bg: 'bg-lime-500/15', border: 'border-lime-500/40' },
  detect: { icon: 'i-tabler-eye', bg: 'bg-violet-500/15', border: 'border-violet-500/40' },
  input: { icon: 'i-tabler-device-gamepad', bg: 'bg-orange-500/15', border: 'border-orange-500/40' },
  system: { icon: 'i-tabler-settings', bg: 'bg-zinc-500/15', border: 'border-zinc-500/40' },
  io: { icon: 'i-tabler-message-circle', bg: 'bg-sky-500/15', border: 'border-sky-500/40' },
  stopwatch: { icon: 'i-tabler-clock', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  mock: { icon: 'i-tabler-test-pipe', bg: 'bg-pink-500/15', border: 'border-pink-500/40' },
  test: { icon: 'i-tabler-flask', bg: 'bg-pink-500/15', border: 'border-pink-500/40' },
}

// backend Input.Widget.Kind → FE FieldSchema.type. 1:1 直传 (11 种 widget kind).
function widgetKindToFieldType(k: string): FieldSchema['type'] {
  // FE FieldSchema.type 类型 union: select/text/number/bool/var-name-select/template-picker/
  // key-capture/monaco/subgraph-picker. backend Widget.Kind 是: async-dropdown/checkbox/
  // dropdown/duration/json/number/password/rect-editor/slider/text/textarea. 映射:
  switch (k) {
    case 'dropdown':
    case 'async-dropdown':
      return 'select'
    case 'checkbox':
      return 'bool'
    case 'number':
    case 'slider':
    case 'duration':
      return 'number'
    case 'text':
    case 'password':
    case 'textarea':
    case 'json':
    case 'rect-editor':
    default:
      return 'text'
  }
}

// backend Type ("Number" / "String" / "Bool" / "Point" / "*" / 等) → FE PinType.
function backendTypeToPinType(t: string): PinType {
  switch (t.toLowerCase()) {
    case 'number':
    case 'integer':
      return 'number'
    case 'bool':
    case 'boolean':
      return 'bool'
    case 'string':
      return 'string'
    case 'point':
      return 'point'
    default:
      return 'any'
  }
}

// 把 backend InputSpec[] / OutputSpec[] 分成 exec pin names vs data pin map.
function splitInputs(inputs: InputSpec[]): {
  execIn: string[]
  dataIn: Record<string, PinType>
} {
  const execIn: string[] = []
  const dataIn: Record<string, PinType> = {}
  for (const i of inputs) {
    if (i.type === 'Exec') execIn.push(i.name)
    else dataIn[i.name] = backendTypeToPinType(i.type)
  }
  return { execIn, dataIn }
}

function splitOutputs(outputs: OutputSpec[]): {
  execOut: string[]
  dataOut: Record<string, PinType>
} {
  const execOut: string[] = []
  const dataOut: Record<string, PinType> = {}
  for (const o of outputs) {
    if (o.type === 'Exec') {
      execOut.push(o.name)
      // exec 出口的 data 字段也算 dataOut (FE 拖数据边从 exec 出口的 data field 拉)
      for (const f of o.data ?? []) {
        dataOut[f.name] = backendTypeToPinType(f.type)
      }
    } else {
      dataOut[o.name] = backendTypeToPinType(o.type)
    }
  }
  return { execOut, dataOut }
}

// Inspector form schema: 从 backend Inputs[] 派生. exec pin 跳过 (它是连边不是 form 项).
function deriveFields(inputs: InputSpec[]): FieldSchema[] {
  const fields: FieldSchema[] = []
  for (const i of inputs) {
    if (i.type === 'Exec') continue
    const widget = i.widget
    const f: FieldSchema = {
      key: i.name,
      label: i.displayName || i.name,
      type: widgetKindToFieldType(widget?.kind ?? 'text'),
      hint: i.doc,
    }
    // dropdown options 从 Widget.Props.options 抽 (backend MarshalProps 写过来的).
    const props = (widget?.props ?? {}) as Record<string, unknown>
    if (f.type === 'select' && Array.isArray(props.options)) {
      f.options = (props.options as Array<{ value: unknown; label: string }>).map((o) => ({
        value: String(o.value),
        label: o.label,
      }))
    }
    fields.push(f)
  }
  return fields
}

// defaults: 从 InputSpec.Default 收集成 literal map. backend Default 是 any (json.Number /
// string / bool / map 等), 直接放 literal 里.
function deriveDefaults(inputs: InputSpec[]): Record<string, unknown> {
  const literal: Record<string, unknown> = {}
  for (const i of inputs) {
    if (i.type === 'Exec') continue
    if (i.default === undefined || i.default === null) continue
    literal[i.name] = i.default
  }
  return Object.keys(literal).length > 0 ? { literal } : {}
}

// dynamic exec out — 镜像 backend validator. Switch/Parallel/Race 等节点 exec 出口由
// config 推. 这部分目前 backend Spec 不暴露, FE hardcode.
const DYNAMIC_EXEC_OUT: Record<
  string,
  (cfg: Record<string, unknown> | null | undefined) => string[]
> = {
  Switch: (cfg) => {
    const cases = Array.isArray(cfg?.Cases)
      ? (cfg!.Cases as unknown[])
      : Array.isArray(cfg?.cases)
      ? (cfg!.cases as unknown[])
      : []
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
  Parallel: parallelBranchPins,
  Race: parallelBranchPins,
}

function parallelBranchPins(cfg: Record<string, unknown> | null | undefined): string[] {
  const lit = (cfg?.literal as Record<string, unknown> | undefined) ?? {}
  const raw = Number(lit.N ?? lit.n)
  const n = Math.max(2, Math.min(8, Number.isFinite(raw) && raw > 0 ? Math.floor(raw) : 2))
  const out: string[] = []
  for (let i = 0; i < n; i++) out.push(`branch${i}`)
  out.push('complete')
  return out
}

// dynamic data in — Expr.Inputs[] / Subgraph 参数. 跟 backend 镜像.
const DYNAMIC_DATA_IN: Record<
  string,
  (cfg: Record<string, unknown> | null | undefined) => Record<string, PinType>
> = {
  Expr: (cfg) => {
    const inputs = Array.isArray(cfg?.Inputs)
      ? (cfg!.Inputs as Array<Record<string, unknown>>)
      : []
    const out: Record<string, PinType> = {}
    for (const i of inputs) {
      const name = typeof i.Name === 'string' ? i.Name : ''
      const type = typeof i.Type === 'string' ? i.Type : 'any'
      if (name) out[name] = backendTypeToPinType(type)
    }
    return out
  },
}

// 主转换: backend Spec → NodeKindSpec.
function adaptSpec(s: Spec): NodeKindSpec {
  const group = GROUP_MAP[s.category] ?? 'system'
  const visual = VISUAL_BY_GROUP[group] ?? VISUAL_BY_GROUP.system
  const { execIn, dataIn } = splitInputs(s.inputs ?? [])
  const { execOut, dataOut } = splitOutputs(s.outputs ?? [])

  const out: NodeKindSpec = {
    kind: s.kind,
    group,
    // labelZh / description 字段语义改: 现在存 i18n key (node.<kind>.label / node.<kind>.description).
    // consumer 走 t(). backend Spec.displayName/description 不再被 FE 用 — FE 单源 (zh.ts node.* lookup).
    labelZh: `node.${s.kind}.label`,
    description: `node.${s.kind}.description`,
    visual,
    execIn,
    execOut,
    dataIn,
    dataOut,
    fields: deriveFields(s.inputs ?? []),
    defaults: deriveDefaults(s.inputs ?? []),
    isPureData: s.isPureData,
    isVisualOnly: s.isVisualOnly,
  }
  // 标记节点 (SubgraphInput/Output, CollapsedNode 等) 不在 palette 里显示
  if (s.isGraphMarker || s.isVisualOnly) {
    out.excludeFromPalette = true
  }
  if (DYNAMIC_EXEC_OUT[s.kind]) out.execOutFn = DYNAMIC_EXEC_OUT[s.kind]
  if (DYNAMIC_DATA_IN[s.kind]) out.dataInDynamicFn = DYNAMIC_DATA_IN[s.kind]
  return out
}

// app boot 时调一次. 启动期 RPC 拉 backend spec → 注册到老 byKind.
// 6 处 consumer (Palette / ContextMenu / etc) 不动.
// 末尾 rebuild pinSpec / nodeFieldSchemas 的 module-level const map (那俩老模块在
// 顶层用 allSpecs() 转 const, 跟 RPC-driven 启动时序错位, 必须 populate 后再填).
export async function populateRegistryFromBackend(): Promise<void> {
  const store = useNodeRegistryStore()
  if (!store.loaded) await store.loadAll()
  __resetForTests() // 清空, 防 dev hot reload 重复 register 触发 dup throw
  for (const spec of store.specs.values()) {
    register(adaptSpec(spec))
  }
  // 异步 import 避免循环: pinSpec/nodeFieldSchemas → registry, registry 不应反 import.
  const { rebuildPinSpecMaps } = await import('@/components/containers/pinSpec')
  const { rebuildNodeFieldSchemas } = await import(
    '@/components/containers/nodeFieldSchemas'
  )
  rebuildPinSpecMaps()
  rebuildNodeFieldSchemas()
}
