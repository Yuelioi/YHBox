// frontend/src/components/containers/pinSpec.ts
//
// Thin compatibility shell. All data lives in nodeRegistry/. This file
// exports the legacy named maps (PIN_SPECS / KIND_LABEL_ZH / ...) by
// deriving them from the registry. Existing call sites keep working
// without changes; new code should import from
// '@/components/containers/nodeRegistry' directly.
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

// Re-exports — preserves the v3/v4 API surface for existing callers.
export { TYPE_COLOR, pinTypeCompat }
export type { PinType, FieldSchema }

/** Legacy alias for callers that use PinDataType. */
export type PinDataType = PinType

/** Legacy alias used by GetVar/SetVar/Expr/pure-func nodes. */
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
export const KIND_DEFAULTS: Record<string, Record<string, any>> = {}
export const KIND_VISUAL: Record<string, { icon: string; bg: string; border: string }> = {}

/** RPC populate 完后调一次. main.ts 在 await populateRegistryFromBackend() 之后调. */
export function rebuildPinSpecMaps(): void {
  for (const k of Object.keys(PIN_SPECS)) delete PIN_SPECS[k]
  for (const k of Object.keys(KIND_LABEL_ZH)) delete KIND_LABEL_ZH[k]
  for (const k of Object.keys(KIND_DESCRIPTION)) delete KIND_DESCRIPTION[k]
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
    KIND_DEFAULTS[s.kind] = s.defaults
    KIND_VISUAL[s.kind] = s.visual
  }

  // B2: SubgraphInput/Output 不再 backend register, 但编辑器仍要渲染 virtual marker.
  // 手动加 PIN_SPECS + visual 让 ContainerFlowNode 能画 (1 exec-out / 1 exec-in).
  PIN_SPECS['SubgraphInput'] = { execIn: [], execOut: ['out'], dataIn: {}, dataOut: {} }
  KIND_LABEL_ZH['SubgraphInput'] = 'node.SubgraphInput.label'
  KIND_DESCRIPTION['SubgraphInput'] = 'node.SubgraphInput.description'
  KIND_DEFAULTS['SubgraphInput'] = {}
  KIND_VISUAL['SubgraphInput'] = { icon: 'i-tabler-circle-arrow-right', bg: 'bg-emerald-500/10', border: 'border-emerald-500/40' }

  PIN_SPECS['SubgraphOutput'] = { execIn: ['In'], execOut: [], dataIn: {}, dataOut: {} }
  KIND_LABEL_ZH['SubgraphOutput'] = 'node.SubgraphOutput.label'
  KIND_DESCRIPTION['SubgraphOutput'] = 'node.SubgraphOutput.description'
  KIND_DEFAULTS['SubgraphOutput'] = {}
  KIND_VISUAL['SubgraphOutput'] = { icon: 'i-tabler-circle-arrow-left', bg: 'bg-rose-500/10', border: 'border-rose-500/40' }
}

/**
 * Returns the pin lists for a node kind given its current config.
 * Replaces the old PIN_SPECS[kind] + execOutFn(config) pattern.
 */
export function pinsFor(
  kind: string,
  config?: Record<string, unknown> | null,
): { execIn: string[]; execOut: string[]; dataIn: string[]; dataOut: string[] } {
  const s = getSpec(kind)
  if (!s) {
    return { execIn: ['in'], execOut: ['out'], dataIn: [], dataOut: [] }
  }
  const dynDataIn = s.dataInDynamicFn ? s.dataInDynamicFn(config) : {}
  return {
    execIn: s.execIn,
    execOut: registryExecOutPinsFor(kind, config),
    dataIn: [...Object.keys(s.dataIn), ...Object.keys(dynDataIn)],
    dataOut: Object.keys(s.dataOut),
  }
}

/** Re-exported for legacy callers. */
export const execOutPinsFor = registryExecOutPinsFor

/** Edge type derived from pin (kind, fromPin). Mirrors backend logic. */
export function edgeKind(fromKind: string, fromPin: string): 'exec' | 'data' {
  return edgeKindOf(fromKind, fromPin)
}

/**
 * Subgraph 调用节点 exec-out 是动态查 subgraph.outputPins 派生, 不在 registry spec 表达范围内.
 * 保留此 helper — 调用方 (ContainerFlowNode.vue 等) 直接使用.
 */
export function resolveSubgraphCallExecOut(
  node: { config?: { SubgraphID?: string } },
  allSubgraphs: { id: string; outputPins: { id: string; name: string }[] }[],
): { id: string; name: string }[] {
  const sgID = node.config?.SubgraphID ?? ''
  const sg = allSubgraphs.find((s) => s.id === sgID)
  const t = i18n.global.t
  if (!sg) return [{ id: '__missing__', name: t('node.Subgraph.fallback_missing') }]
  if (sg.outputPins.length === 0) return [{ id: '__empty__', name: t('node.Subgraph.fallback_empty') }]
  return sg.outputPins.map((p) => ({ id: p.id, name: p.name }))
}

/** Export the spec type for callers that need it. */
export type { NodeKindSpec }
