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
//  - dynamicPorts: backend descriptor 驱动 config-derived pins.
//
// app 入口 main.ts boot 时 await populateRegistryFromBackend() 把 RPC spec 注册到
// 老 byKind, 各 consumer (ContextMenu / NodeExplorerModal /
// FindReferencesModal / nodeFieldSchemas) 不变.

import { useNodeRegistryStore } from '@/stores/nodeRegistry'
import { setExprFunctions } from '@/lib/exprFunctions'
import { __resetForTests, register } from './registry'
import type { FieldSchema, NodeFieldSchema, NodeGroup, NodeKindSpec, PinType } from './index'
import {
  DynamicPortRole,
  DynamicPortShape,
  type DynamicPortSpec,
  type InputSpec,
  type OutputSpec,
  type Spec,
} from '@bindings/github.com/yottaapp/yotta/internal/node'
import { groupVisual, resolvePalette } from '../visualRegistry'
import { rebuildPinSpecMaps } from '@/components/containers/pinSpec'
import { rebuildNodeFieldSchemas } from '@/components/containers/nodeFieldSchemas'

// Backend Category → FE NodeGroup. backend 用 TitleCase, FE 历史 lowercase + 'variables'.
const GROUP_MAP: Record<string, NodeGroup> = {
  Control: 'control',
  Detect: 'detect',
  Image: 'image',
  Input: 'input',
  Target: 'target',
  System: 'system',
  Variable: 'variables',
  PureFunc: 'purefunc',
  IO: 'io',
  Stopwatch: 'stopwatch',
  Mock: 'mock',
  Test: 'test',
  Event: 'event',
  Random: 'random',
  List: 'list',
  Window: 'window',
}

// 按 group 取 visual (icon + tailwind color) — 从视觉注册中心 (visualRegistry) 派生,
// 不再本地 hardcode (消除与 useNodeGroupColor GROUP_TONE 的漂移).
function visualForGroup(group: NodeGroup): { icon: string; bg: string; border: string } {
  const gv = groupVisual(group)
  const p = resolvePalette(gv.color)
  return { icon: gv.icon, bg: p.bg, border: p.border }
}

// backend Input.Widget.Kind → FE FieldSchema.type. 1:1 直传 (11 种 widget kind).
function widgetKindToFieldType(k: string): FieldSchema['type'] {
  // FE FieldSchema.type 类型 union: select/text/number/bool/key-capture.
  // backend Widget.Kind 是: async-dropdown/checkbox/key-capture/
  // dropdown/duration/json/number/password/rect-editor/slider/text/textarea. 映射:
  switch (k) {
    case 'key-capture':
      return 'key-capture'
    case 'color-preset':
      return 'color-preset'
    case 'icon-preset':
      return 'icon-preset'
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

// backend Type ("Number" / "String" / "Bool" / "Point" / "Duration" / "*" / 等) → FE PinType.
function backendTypeToPinType(t: string): PinType {
  switch (t.toLowerCase()) {
    case 'number':
    case 'integer':
    case 'duration': // Duration 值就是毫秒数 (Go 把裸数字按 ms 解析); 当 number 渲染可填数字
      return 'number'
    case 'bool':
    case 'boolean':
      return 'bool'
    case 'string':
      return 'string'
    case 'point':
      return 'point'
    case 'list':
      return 'list'
    case 'file':
      return 'file'
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
  errorOut: string[]
} {
  const execOut: string[] = []
  const dataOut: Record<string, PinType> = {}
  // Semantic==='error' 的 exec 出口名 (后端 Fail 出口) — FE 上色成红引脚.
  const errorOut: string[] = []
  for (const o of outputs) {
    if (o.type === 'Exec') {
      execOut.push(o.name)
      if (o.semantic === 'error') errorOut.push(o.name)
      // exec 出口的 data 字段也算 dataOut (FE 拖数据边从 exec 出口的 data field 拉)
      for (const f of o.data ?? []) {
        dataOut[f.name] = backendTypeToPinType(f.type)
      }
    } else {
      dataOut[o.name] = backendTypeToPinType(o.type)
    }
  }
  return { execOut, dataOut, errorOut }
}

// Inspector form schema: 从 backend Inputs[] 派生. exec pin 跳过 (它是连边不是 form 项).
// label / hint / option label 全是 i18n key 字符串 (node.<kind>.input.<name>.{label,hint,option.<value>}),
// 不再是 backend 字面值 — backend Spec 已不带任何展示文案. consumer 必须 t() 渲染.
// hint 对所有 field 都出 key; 渲染层用 te() 判存在性 (key 没配 → 不渲染), 取代旧的 backend doc gate.
// 导出供测试; app 内部仍通过 adaptSpec → populateRegistryFromBackend 调用.
export function deriveFields(kind: string, inputs: InputSpec[]): FieldSchema[] {
  const fields: FieldSchema[] = []
  for (const i of inputs) {
    if (i.type === 'Exec') continue
    const widget = i.widget
    const f: FieldSchema = {
      key: i.name,
      label: `node.${kind}.input.${i.name}.label`,
      type: widgetKindToFieldType(widget?.kind ?? 'text'),
      widgetKind: widget?.kind ?? 'text',
      hint: `node.${kind}.input.${i.name}.hint`,
    }
    // schema 透传: 后端 InputSpec.schema 非空 → FE FieldSchema.schema (StructuredInput 渲染路径).
    if (i.schema) f.schema = i.schema as unknown as NodeFieldSchema
    // advanced / semantic 透传: Inspector 用 semantic==='varname' 选 VarNameInput 控件.
    if (i.advanced) f.advanced = true
    if (i.semantic) f.semantic = i.semantic
    // dropdown options 从 Widget.Props.options 抽 (backend MarshalProps 写来的, 静态 dropdown 只有 value).
    // label 走 i18n key node.<kind>.input.<name>.option.<value> — backend 不再带 enum 中文.
    const props = (widget?.props ?? {}) as Record<string, unknown>
    // placeholder (e.g. JSONProps.Placeholder) 透到 PinInput 的 textarea/input 占位示例.
    if (typeof props.placeholder === 'string') f.placeholder = props.placeholder
    // number/slider 的 min/max/step (SliderProps) → 透到 PinInput 的 UInputNumber, 让小数步进生效.
    if (typeof props.min === 'number') f.min = props.min
    if (typeof props.max === 'number') f.max = props.max
    if (typeof props.step === 'number') f.step = props.step
    if (typeof props.asyncSource === 'string') f.asyncSource = props.asyncSource
    if (props.applyMeta && typeof props.applyMeta === 'object' && !Array.isArray(props.applyMeta)) {
      f.applyMeta = Object.fromEntries(
        Object.entries(props.applyMeta as Record<string, unknown>).filter(
          ([, v]) => typeof v === 'string',
        ),
      ) as Record<string, string>
    }
    if (f.type === 'select' && Array.isArray(props.options)) {
      f.options = (props.options as Array<{ value: unknown }>).map((o) => ({
        value: String(o.value),
        labelKey: `node.${kind}.input.${i.name}.option.${String(o.value)}`,
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

function dynamicPortsOf(spec: Spec): DynamicPortSpec[] {
  return spec.dynamicPorts ?? []
}

// Parallel/Race still use their legacy numeric branch contract. Config-derived node
// contracts such as Switch are handled from backend dynamicPorts below.
const DYNAMIC_EXEC_OUT: Record<
  string,
  (cfg: Record<string, unknown> | null | undefined) => string[]
> = {
  Parallel: parallelBranchPins,
  Race: parallelBranchPins,
}

function namedDynamicExecOutputs(
  descriptor: DynamicPortSpec,
  staticOutputs: string[],
): (cfg: Record<string, unknown> | null | undefined) => string[] {
  return (cfg) => {
    const values = Array.isArray(cfg?.[descriptor.configKey])
      ? (cfg?.[descriptor.configKey] as unknown[])
      : []
    const out: string[] = []
    const seen = new Set(staticOutputs)
    for (const value of values.slice(0, descriptor.maxItems)) {
      if (
        typeof value !== 'string' ||
        value === '' ||
        value.trim() !== value ||
        value.includes('.') ||
        seen.has(value)
      )
        continue
      seen.add(value)
      out.push(value)
    }
    return [...out, ...staticOutputs]
  }
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

// Name/type record descriptors read PascalCase Name/Type keys from their declared config key.
function parseNameTypeRecordsCfg(
  configKey: string,
  cfg: Record<string, unknown> | null | undefined,
): Record<string, PinType> {
  const records = Array.isArray(cfg?.[configKey])
    ? (cfg?.[configKey] as Array<Record<string, unknown>>)
    : []
  const out: Record<string, PinType> = {}
  for (const record of records) {
    const name = typeof record.Name === 'string' ? record.Name : ''
    const type = typeof record.Type === 'string' ? record.Type : 'any'
    if (name) out[name] = backendTypeToPinType(type)
  }
  return out
}

// 主转换: backend Spec → NodeKindSpec.
export function adaptSpec(s: Spec): NodeKindSpec {
  const group = GROUP_MAP[s.category] ?? 'system'
  const visual = visualForGroup(group)
  const { execIn, dataIn } = splitInputs(s.inputs ?? [])
  const { execOut, dataOut, errorOut } = splitOutputs(s.outputs ?? [])
  const dynamicPorts = dynamicPortsOf(s)

  const out: NodeKindSpec = {
    kind: s.kind,
    group,
    // labelZh / description 字段语义改: 现在存 i18n key (node.<kind>.label / node.<kind>.description).
    // consumer 走 t(). backend Spec.displayName/description 不再被 FE 用 — FE 单源 (zh.ts node.* lookup).
    labelZh: `node.${s.kind}.label`,
    description: `node.${s.kind}.description`,
    example: `node.${s.kind}.example`,
    visual,
    execIn,
    execOut,
    errorOut,
    dataIn,
    dataOut,
    fields: deriveFields(s.kind, s.inputs ?? []),
    supportedTargets: ((s as any).supportedTargets ?? []) as NodeKindSpec['supportedTargets'],
    defaults: deriveDefaults(s.inputs ?? []),
    isPureData: s.isPureData,
    isVisualOnly: s.isVisualOnly,
  }
  // 标记节点 (SubgraphInput/Output, CollapsedNode 等) 不在 palette 里显示;
  // CommentBox 虽 visual-only 但要能从面板拖出来 (sticky note), 例外.
  if ((s.isGraphMarker || s.isVisualOnly) && s.kind !== 'CommentBox') {
    out.excludeFromPalette = true
  }
  const namedOutputs = dynamicPorts.find(
    (port) =>
      port.role === DynamicPortRole.DynamicPortOutput &&
      port.shape === DynamicPortShape.DynamicPortNames &&
      port.fixedType === 'Exec',
  )
  if (namedOutputs) out.execOutFn = namedDynamicExecOutputs(namedOutputs, execOut)
  else if (DYNAMIC_EXEC_OUT[s.kind]) out.execOutFn = DYNAMIC_EXEC_OUT[s.kind]
  const dynamicInputs = dynamicPorts.find(
    (port) =>
      port.role === DynamicPortRole.DynamicPortInput &&
      port.shape === DynamicPortShape.DynamicPortNameTypeRecords,
  )
  if (dynamicInputs) {
    out.dynamicInputs = true
    out.dataInDynamicFn = (cfg) => parseNameTypeRecordsCfg(dynamicInputs.configKey, cfg)
  }
  const dynamicOutputData = dynamicPorts.find(
    (port) =>
      port.role === DynamicPortRole.DynamicPortOutputData &&
      port.shape === DynamicPortShape.DynamicPortNameTypeRecords,
  )
  if (dynamicOutputData) {
    out.dynamicDataFields = true
    out.dataOutDynamicFn = (cfg) => parseNameTypeRecordsCfg(dynamicOutputData.configKey, cfg)
  }
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
  // Expr 函数元数据 (补全/未知函数检查) — RPC 单源, 喂给 exprFunctions 模块级表.
  setExprFunctions(store.exprFunctions)
  // pinSpec/nodeFieldSchemas derive module-level maps from the populated registry.
  rebuildPinSpecMaps()
  rebuildNodeFieldSchemas()
}
