// 从 nodeRegistry/ 派生命名 map (PIN_SPECS / KIND_LABEL_ZH / ...). 正源在 registry,
// 新代码直接 import '@/components/containers/nodeRegistry'.
//
// DO NOT add new entries here — register them in nodeRegistry/specs/*.ts.

// nodeRegistry/specs is imported once at app entry (main.ts). No side-effect needed here.
import {
  allSpecs,
  getSpec,
  edgeKindOf,
  execOutPinsFor as registryExecOutPinsFor,
} from '@/components/containers/nodeRegistry/registry'
import { TYPE_COLOR, pinTypeCompat } from '@/components/containers/nodeRegistry/index'
import type { PinType, FieldSchema, NodeKindSpec } from '@/components/containers/nodeRegistry/index'
import { i18n } from '@/i18n'
import { builtinNodeProjections31 } from '@/contracts/node31'

// Re-exports from nodeRegistry.
export { TYPE_COLOR, pinTypeCompat }
export type { PinType, FieldSchema }

/** PinDataType 别名. */
export type PinDataType = PinType

/** DataPinSpec — GetVar/SetVar/Expr/pure-func 节点用. */
export type DataPinSpec = { name: string; type: PinType }

/** Legacy shape for PIN_SPECS entries. */
export interface PinSpec {
  execIn: string[]
  execOut: string[]
  dataIn: Record<string, PinType>
  dataOut: Record<string, PinType>
  execOutFn?: (cfg: Record<string, unknown> | null | undefined) => string[]
}

// 5 个 const map: 改成 mutable empty 对象, 由 rebuildPinSpecMaps() 在 RPC populate
// 完后填. 原 module-init 写法 (allSpecs() 顶层执行) 在 RPC-driven 启动模式下时序
// 错位 — module 加载早于 main.ts await populateRegistryFromBackend(), allSpecs()
// 返空 → 节点显示全白块没 label.
//
// const 是引用, 内容 mutate. consumer (ContainerFlowNode.vue 等) 用 KIND_VISUAL[kind]
// 访问语义不变 (Vue reactivity 跨原始对象 mutate 也能触发再 render).

export const PIN_SPECS: Record<string, PinSpec> = {}
/**
 * Map<kind, i18n key>. 值现在是 i18n key 字符串 (e.g. 'node.Add.label'), 不是中文字面值.
 * Consumer 必须用 t(KIND_LABEL_ZH[kind]) 渲染. 名字 ZH 是历史包袱.
 */
export const KIND_LABEL_ZH: Record<string, string> = {}
/** Map<kind, i18n key>. 值是 'node.<kind>.description'. consumer 走 t() 渲染. */
export const KIND_DESCRIPTION: Record<string, string> = {}
/** Map<kind, i18n key>. 值是 'node.<kind>.example'. consumer 走 te()/t() — 没翻译则不渲染示例区. */
export const KIND_EXAMPLE: Record<string, string> = {}
export const KIND_DEFAULTS: Record<string, Record<string, any>> = {}
export const KIND_VISUAL: Record<string, { icon: string; bg: string; border: string }> = {}

/** RPC populate 完后调一次. main.ts 在 await populateRegistryFromBackend() 之后调. */
export function rebuildPinSpecMaps(): void {
  for (const k of Object.keys(PIN_SPECS)) delete PIN_SPECS[k]
  for (const k of Object.keys(KIND_LABEL_ZH)) delete KIND_LABEL_ZH[k]
  for (const k of Object.keys(KIND_DESCRIPTION)) delete KIND_DESCRIPTION[k]
  for (const k of Object.keys(KIND_EXAMPLE)) delete KIND_EXAMPLE[k]
  for (const k of Object.keys(KIND_DEFAULTS)) delete KIND_DEFAULTS[k]
  for (const k of Object.keys(KIND_VISUAL)) delete KIND_VISUAL[k]

  for (const s of allSpecs()) {
    PIN_SPECS[s.kind] = {
      execIn: s.execIn,
      execOut: s.execOut,
      dataIn: s.dataIn,
      dataOut: s.dataOut,
      execOutFn: s.execOutFn,
    }
    KIND_LABEL_ZH[s.kind] = s.labelZh
    KIND_DESCRIPTION[s.kind] = s.description
    KIND_EXAMPLE[s.kind] = s.example
    KIND_DEFAULTS[s.kind] = s.defaults
    KIND_VISUAL[s.kind] = s.visual
  }

  // B2: SubgraphInput/Output 不再 backend register, 但编辑器仍要渲染 virtual marker.
  // 手动加 PIN_SPECS + visual 让 ContainerFlowNode 能画 (1 exec-out / 1 exec-in).
  // entry marker 的 exec-out 名必须是 'Done' — runtime 从 entryID+".Done" 播种 (dispatch_v5.go),
  // 与 SubgraphOutput.execIn='In' 同走 PascalCase exec 约定; 用 'out' 会让边渲染错位 + 运行时不播种.
  PIN_SPECS['SubgraphInput'] = { execIn: [], execOut: ['Done'], dataIn: {}, dataOut: {} }
  KIND_LABEL_ZH['SubgraphInput'] = 'node.SubgraphInput.label'
  KIND_DESCRIPTION['SubgraphInput'] = 'node.SubgraphInput.description'
  KIND_DEFAULTS['SubgraphInput'] = {}
  KIND_VISUAL['SubgraphInput'] = {
    icon: 'i-tabler-circle-arrow-right',
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/40',
  }

  PIN_SPECS['SubgraphOutput'] = { execIn: ['In'], execOut: [], dataIn: {}, dataOut: {} }
  KIND_LABEL_ZH['SubgraphOutput'] = 'node.SubgraphOutput.label'
  KIND_DESCRIPTION['SubgraphOutput'] = 'node.SubgraphOutput.description'
  KIND_DEFAULTS['SubgraphOutput'] = {}
  KIND_VISUAL['SubgraphOutput'] = {
    icon: 'i-tabler-circle-arrow-left',
    bg: 'bg-rose-500/10',
    border: 'border-rose-500/40',
  }
}

/**
 * Returns the pin lists for a node kind given its current config.
 * Replaces the old PIN_SPECS[kind] + execOutFn(config) pattern.
 */
export function pinsFor(
  kind: string,
  config?: Record<string, unknown> | null,
): { execIn: string[]; execOut: string[]; dataIn: string[]; dataOut: string[] } {
  const projection31 = builtinNodeProjections31.get(kind)
  if (projection31) {
    return {
      execIn: projection31.execInputs,
      execOut: projection31.execOutputs,
      dataIn: projection31.dataInputs.map((port) => port.id),
      dataOut: projection31.dataOutputs.map((port) => port.id),
    }
  }
  const s = getSpec(kind)
  if (!s) {
    // virtual marker (SubgraphInput/Output) 不在 registry, pin 定义在 PIN_SPECS —
    // 必须从这里取 (entry exec-out='Done' / output exec-in='In'), 否则 fallback 的
    // in/out 跟边 + runtime 的 Done/In 对不上 → 边渲染掉到节点底部 + 运行时不播种.
    const marker = PIN_SPECS[kind]
    if (marker) {
      return {
        execIn: marker.execIn,
        execOut: marker.execOut,
        dataIn: Object.keys(marker.dataIn),
        dataOut: Object.keys(marker.dataOut),
      }
    }
    return { execIn: [], execOut: [], dataIn: [], dataOut: [] }
  }
  const dynDataIn = s.dataInDynamicFn ? s.dataInDynamicFn(config) : {}
  const dynDataOut = s.dataOutDynamicFn ? s.dataOutDynamicFn(config) : {}
  return {
    execIn: s.execIn,
    execOut: registryExecOutPinsFor(kind, config),
    dataIn: [...Object.keys(s.dataIn), ...Object.keys(dynDataIn)],
    dataOut: [...Object.keys(s.dataOut), ...Object.keys(dynDataOut)],
  }
}

/** Re-exported for legacy callers. */
export const execOutPinsFor = registryExecOutPinsFor

/** Edge type derived from pin (kind, fromPin). 须与后端 edge-kind 推导一致. */
export function edgeKind(fromKind: string, fromPin: string): 'exec' | 'data' | '' {
  return edgeKindOf(fromKind, fromPin)
}

/**
 * 子图未解析时 resolveSubgraphCallExecOut 的渲染兜底 pin —— 纯**显示哨兵**。
 * 绝不能连成持久化边 (会被后端校验拒 "不存在 out pin __missing__"); 由 useGraphMutations.onConnect 拦截。
 */
export const MISSING_PIN = '__missing__'
export const EMPTY_PIN = '__empty__'
export function isSentinelPin(pin: string | null | undefined): boolean {
  return pin === MISSING_PIN || pin === EMPTY_PIN
}

/**
 * Subgraph 调用节点 exec-out 是动态查 subgraph.outputPins 派生, 不在 registry spec 表达范围内.
 * 保留此 helper — 调用方 (ContainerFlowNode.vue 等) 直接使用.
 * 子图查不到 → 返回 MISSING_PIN 哨兵 (仅渲染, onConnect 不让它连成边)。
 */
export function resolveSubgraphCallExecOut(
  node: { config?: { SubgraphID?: string } },
  allSubgraphs: { id: string; outputPins: { id: string; name: string }[] }[],
): { id: string; name: string }[] {
  const sgID = node.config?.SubgraphID ?? ''
  const sg = allSubgraphs.find((s) => s.id === sgID)
  const t = i18n.global.t
  if (!sg) return [{ id: MISSING_PIN, name: t('node.Subgraph.fallback_missing') }]
  if (sg.outputPins.length === 0)
    return [{ id: EMPTY_PIN, name: t('node.Subgraph.fallback_empty') }]
  return sg.outputPins.map((p) => ({ id: p.id, name: p.name }))
}

/** Export the spec type for callers that need it. */
export type { NodeKindSpec }
