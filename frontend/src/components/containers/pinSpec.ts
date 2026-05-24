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

/** Derived from nodeRegistry. */
export const PIN_SPECS: Record<string, PinSpec> = (() => {
  const out: Record<string, PinSpec> = {}
  for (const s of allSpecs()) {
    out[s.kind] = {
      execIn: s.execIn,
      execOut: s.execOut,
      dataIn: s.dataIn,
      dataOut: s.dataOut,
      execOutFn: s.execOutFn,
    }
  }
  return out
})()

/** Derived from nodeRegistry. */
export const KIND_LABEL_ZH: Record<string, string> = (() => {
  const out: Record<string, string> = {}
  for (const s of allSpecs()) out[s.kind] = s.labelZh
  return out
})()

/** Derived from nodeRegistry. */
export const KIND_DESCRIPTION: Record<string, string> = (() => {
  const out: Record<string, string> = {}
  for (const s of allSpecs()) out[s.kind] = s.description
  return out
})()

/** Derived from nodeRegistry. */
export const KIND_DEFAULTS: Record<string, Record<string, any>> = (() => {
  const out: Record<string, Record<string, any>> = {}
  for (const s of allSpecs()) out[s.kind] = s.defaults
  return out
})()

/** Derived from nodeRegistry. */
export const KIND_VISUAL: Record<string, { icon: string; bg: string; border: string }> = (() => {
  const out: Record<string, { icon: string; bg: string; border: string }> = {}
  for (const s of allSpecs()) out[s.kind] = s.visual
  return out
})()

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
  if (!sg) return [{ id: '__missing__', name: '(子图未找到)' }]
  if (sg.outputPins.length === 0) return [{ id: '__empty__', name: '(无出口)' }]
  return sg.outputPins.map((p) => ({ id: p.id, name: p.name }))
}

/** Export the spec type for callers that need it. */
export type { NodeKindSpec }
