// Single source of truth for node Kind metadata on the frontend.
// 须与后端 node.Spec 结构一致 — structural fields (kind/category/pins/types).
//
// Adding a kind = 1 register({...}) call in specs/<group>.ts. Nothing else.
// pinSpec.ts / nodeFieldSchemas.ts / NodeExplorerModal.vue all derive views over this.

export type PinType = 'number' | 'bool' | 'string' | 'point' | 'any' | 'list'

/** Typed data pin 颜色表. vue-flow Handle background 用. */
export const TYPE_COLOR: Record<PinType, string> = {
  number: '#60a5fa', // blue
  bool: '#f87171', // red
  string: '#a78bfa', // purple
  point: '#34d399', // green
  any: '#9ca3af', // gray
  list: '#818cf8', // indigo
}

/**
 * Pin type compatibility — mirrors backend runtime.PinTypeCompat.
 * @returns allow=can connect, warn=allowed but coerced (UI gives hint)
 *
 * PARITY: 必须跟 `internal/services/container/runtime/pin_types.go` PinTypeCompat 一致
 * — Go TestRegistryParity 跨语言 diff 抓 drift.
 */
export function pinTypeCompat(from: PinType, to: PinType): { allow: boolean; warn: boolean } {
  if (from === to || from === 'any' || to === 'any') return { allow: true, warn: false }
  if (from === 'number' && (to === 'bool' || to === 'string')) return { allow: true, warn: true }
  if (from === 'bool' && (to === 'number' || to === 'string')) return { allow: true, warn: true }
  return { allow: false, warn: false }
}

/** 镜像后端 node.FieldSchema — 结构化输入的递归数据 schema. StructuredInput.vue 据此渲染. */
export interface NodeFieldSchema {
  type: 'object' | 'tuple' | 'array' | 'number' | 'string' | 'bool' | 'enum'
  widget?: string // '' | 'geometry'
  fields?: { key: string; schema: NodeFieldSchema; required?: boolean }[]
  /** type=array (同质变长列表) 的每元素 schema. */
  items?: NodeFieldSchema
  options?: { value: unknown }[]
}

/** 坐标点存储值 (x/y; unit 空=比例 0-1, 'px'=像素). */
export interface PointValue {
  x: number
  y: number
  unit?: 'px'
}

/** 几何输入存储值 (pct ratio 0-1 + 可选每分辨率像素覆盖). */
export interface GeometryValue {
  pct: { x: number; y: number; w: number; h: number }
  overrides?: { resolution: { w: number; h: number }; px: { x: number; y: number; w: number; h: number } }[]
}

/** Field schema for Inspector form (replaces nodeFieldSchemas.NODE_FIELD_SCHEMAS). */
export interface FieldSchema {
  key: string
  label: string
  type:
    | 'select'
    | 'text'
    | 'number'
    | 'bool'
    | 'template-picker'
    | 'key-capture'
    | 'color-preset'
    | 'icon-preset'
  /** 原始 backend widget kind (text/textarea/json/duration/slider/number/checkbox/dropdown/
   * async-dropdown/password/rect-editor) — `type` 把这些收敛成 4 类后丢了 json/textarea/duration
   * 等区分; PinInput 用 widgetKind 还原正确控件 (修 JSON [object Object] 等)。 */
  widgetKind?: string
  /** dropdown 选项. label 是 i18n key (node.<kind>.input.<name>.option.<value>), consumer 走 t(). */
  options?: Array<{ value: string; labelKey: string }>
  /** async-dropdown 数据源名. PinInput 调 NodeService.AsyncOptions 懒加载, 仍允许手输兜底。 */
  asyncSource?: string
  /** async option meta 应用映射: option.meta[key] → config.literal[targetPin]. */
  applyMeta?: Record<string, string>
  placeholder?: string
  /** number/slider 的 min/max/step (后端 SliderProps). 透到 UInputNumber, 否则小数步进 (如阈值 0.01) 失效, 只能整数. */
  min?: number
  max?: number
  step?: number
  /** Inline help text shown below the input. Migrated from old nodeFieldSchemas.ts. */
  hint?: string
  /** 结构化输入的递归 schema (后端 InputSpec.schema 透传); 非空 → NodeInspector 用 StructuredInput 渲染. */
  schema?: NodeFieldSchema
  /** backend InputSpec.Advanced — 进阶/不常用输入. */
  advanced?: boolean
  /** backend InputSpec.Semantic — UI 语义标签 (e.g. 'varname' = 变量名输入框). */
  semantic?: string
}

/** Group name for palette categorization. 对应后端 Spec.Category (lowercase'd).
 * 'variables' FE 用复数, backend 是 'Variable'; adapter 映射. */
export type NodeGroup =
  | 'control'
  | 'variables'
  | 'purefunc'
  | 'detect'
  | 'image'
  | 'input'
  | 'target'
  | 'system'
  | 'io'
  | 'stopwatch'
  | 'mock'
  | 'test'
  | 'event'
  | 'random'
  | 'list'
  | 'window'

/**
 * Single-source-of-truth descriptor for a node kind.
 * 须与后端 node.Spec 结构一致 — structural fields only; display text lives in i18n.
 */
export interface NodeKindSpec {
  kind: string
  group: NodeGroup

  /**
   * Inspector + palette display.
   * 字段名是历史包袱 — 现在存 i18n key 'node.<kind>.label' 不是中文字面值.
   * consumer 必须 t(labelZh) 渲染. fallback (i18n 未配置 key) 走 vue-i18n 默认行为 (返 raw key).
   */
  labelZh: string
  description: string
  /** i18n key 'node.<kind>.example' — 可选使用场景示例; te() 没翻译则 Inspector 不显示示例折叠区。 */
  example: string

  /** Renders icon + tailwind class in node body */
  visual: { icon: string; bg: string; border: string }

  /** Pin metadata — empty arrays/maps for no pins */
  execIn: string[]
  execOut: string[]
  /** exec-out 名集合, 其中 backend OutputSpec.Semantic==='error' 的失败出口 (Fail).
   * 节点画布据此把这些 exec 引脚渲染成红色. 普通 exec 出口不在此列. */
  errorOut?: string[]
  /** Dynamic exec-out (Switch/Parallel/Race). When set, takes precedence over execOut. */
  execOutFn?: (cfg: Record<string, unknown> | null | undefined) => string[]
  dataIn: Record<string, PinType>
  dataOut: Record<string, PinType>
  /** Dynamic data-in (Expr.inputs[], Subgraph inputParams). Merged with dataIn at lookup time. */
  dataInDynamicFn?: (cfg: Record<string, unknown> | null | undefined) => Record<string, PinType>
  /** backend Spec.DynamicInputs — 动态 data-in pin 由 config.Inputs[] (PascalCase Name/Type)
   * 声明 (Expr / Script). 驱动 Inspector 声明编辑区 + pinLiterals 动态 pin 合并. */
  dynamicInputs?: boolean
  /** Dynamic data-out (AI.config.Outputs[]). Merged with dataOut at lookup time → 可绑字段。 */
  dataOutDynamicFn?: (cfg: Record<string, unknown> | null | undefined) => Record<string, PinType>
  /** backend Spec.DynamicDataFields — 动态 Data 出口字段由 config.Outputs[] 声明 (AI).
   * 驱动 Inspector 输出声明编辑区 + bindableFields 动态字段合并. */
  dynamicDataFields?: boolean

  /** Config schema for Inspector form (replaces nodeFieldSchemas.ts NODE_FIELD_SCHEMAS[kind]). */
  fields: FieldSchema[]

  /** Defaults filled into node.config on creation */
  defaults: Record<string, any>

  /** Flags — match backend Spec semantics.
   * NOTE: backend Spec.IsYield (loop-body infinite-loop check) is validator-only,
   * no frontend equivalent — D2 parity test skips that field. */
  isPureData?: boolean // no exec pins, evaluated on-demand by data_pull
  isVisualOnly?: boolean // CommentBox — no runtime, no pin checks

  /** Hide from the node palette (NodeExplorerModal) list. The node is still a valid kind
   * (runtime + validator know it), but users create it via a specific UI
   * flow rather than drag-and-drop: SubgraphInput/Output are auto-managed
   * by the subgraph editor; CollapsedNode is created via "fold selection"
   * right-click action. Frontend-only metadata — no backend equivalent. */
  excludeFromPalette?: boolean
}
