// frontend/src/components/containers/nodeRegistry/index.ts
// Single source of truth for node Kind metadata on the frontend.
// Mirrors backend internal/services/container/nodekind/spec.go.
//
// Adding a kind = 1 register({...}) call in specs/<group>.ts. Nothing else.
// pinSpec.ts / nodeFieldSchemas.ts / NodePalette.vue all derive views over this.

export type PinType = 'number' | 'bool' | 'string' | 'point' | 'any'

/** v4 §2.2: color map for typed data pins. vue-flow Handle uses this for background. */
export const TYPE_COLOR: Record<PinType, string> = {
  number: '#60a5fa', // blue
  bool: '#f87171', // red
  string: '#a78bfa', // purple
  point: '#34d399', // green
  any: '#9ca3af', // gray
}

/**
 * v4 pin type compatibility (spec §2.3) — mirrors backend runtime.PinTypeCompat.
 * @returns allow=can connect, warn=allowed but coerced (UI gives hint)
 *
 * PARITY: must match `internal/services/container/runtime/pin_types.go` PinTypeCompat
 * — Phase D2 cross-check test asserts FE↔BE registry-derived behavior aligns.
 */
export function pinTypeCompat(from: PinType, to: PinType): { allow: boolean; warn: boolean } {
  if (from === to || from === 'any' || to === 'any') return { allow: true, warn: false }
  if (from === 'number' && (to === 'bool' || to === 'string')) return { allow: true, warn: true }
  if (from === 'bool' && (to === 'number' || to === 'string')) return { allow: true, warn: true }
  return { allow: false, warn: false }
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
    | 'var-name-select'
    | 'template-picker'
    | 'key-capture'
    | 'monaco'
    | 'subgraph-picker'
  options?: Array<{ value: string; label: string }>
  placeholder?: string
  /** Inline help text shown below the input. Migrated from old nodeFieldSchemas.ts. */
  hint?: string
}

/** Group name for palette categorization. Mirrors backend Spec.Group. */
export type NodeGroup = 'control' | 'variables' | 'purefunc' | 'detect' | 'input' | 'system'

/**
 * Single-source-of-truth descriptor for a node kind.
 * Mirrors backend nodekind.Spec (Go) field-for-field where applicable.
 */
export interface NodeKindSpec {
  kind: string
  group: NodeGroup

  /** Inspector + palette display */
  labelZh: string
  description: string

  /** Renders icon + tailwind class in node body */
  visual: { icon: string; bg: string; border: string }

  /** Pin metadata — empty arrays/maps for no pins */
  execIn: string[]
  execOut: string[]
  /** Dynamic exec-out (Switch/Parallel/Race). When set, takes precedence over execOut. */
  execOutFn?: (cfg: Record<string, unknown> | null | undefined) => string[]
  dataIn: Record<string, PinType>
  dataOut: Record<string, PinType>
  /** Dynamic data-in (Expr.inputs[], Subgraph inputParams). Merged with dataIn at lookup time. */
  dataInDynamicFn?: (cfg: Record<string, unknown> | null | undefined) => Record<string, PinType>

  /** Config schema for Inspector form (replaces nodeFieldSchemas.ts NODE_FIELD_SCHEMAS[kind]). */
  fields: FieldSchema[]

  /** Defaults filled into node.config on creation */
  defaults: Record<string, any>

  /** Flags — match backend Spec semantics.
   * NOTE: backend Spec.IsYield (loop-body infinite-loop check) is validator-only,
   * no frontend equivalent — D2 parity test skips that field. */
  isPureData?: boolean // no exec pins, evaluated on-demand by data_pull
  isVisualOnly?: boolean // CommentBox — no runtime, no pin checks
}
