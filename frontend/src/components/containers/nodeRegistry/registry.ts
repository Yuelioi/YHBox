// frontend/src/components/containers/nodeRegistry/registry.ts
import type { NodeKindSpec, PinType } from './index'

const byKind: Record<string, NodeKindSpec> = {}

/**
 * __resetForTests clears the registry. ONLY use from test setup (beforeEach).
 * Production code must never call this — registry is build-time immutable once
 * specs/index.ts barrel finishes import. Underscored to discourage accidental use.
 */
export function __resetForTests(): void {
  for (const k of Object.keys(byKind)) delete byKind[k]
}

/**
 * Register a NodeKindSpec. Must be called from module-init top level (i.e. from a
 * specs/*.ts file imported by the barrel). Duplicate registrations throw — that's
 * a programmer error, not a runtime condition.
 */
export function register(spec: NodeKindSpec): void {
  if (!spec.kind) throw new Error('register: empty kind')
  if (byKind[spec.kind]) throw new Error(`register: duplicate kind ${spec.kind}`)
  byKind[spec.kind] = spec
}

export function getSpec(kind: string): NodeKindSpec | undefined {
  return byKind[kind]
}

/** Returns all registered kind names (unspecified order). */
export function allKinds(): string[] {
  return Object.keys(byKind)
}

/** Returns all registered specs (unspecified order). */
export function allSpecs(): NodeKindSpec[] {
  return Object.values(byKind)
}

/** v4 replacement for old PIN_SPECS[kind].execOutFn || execOut. */
export function execOutPinsFor(
  kind: string,
  cfg: Record<string, unknown> | null | undefined,
): string[] {
  const s = byKind[kind]
  if (!s) return ['out']
  if (s.execOutFn) return s.execOutFn(cfg)
  return s.execOut.length > 0 ? s.execOut : ['out']
}

/** Returns the PinType of (kind, dataIn pin) or '' if missing. Considers dynamic. */
export function dataInTypeFor(
  kind: string,
  pin: string,
  cfg: Record<string, unknown> | null | undefined,
): PinType | '' {
  const s = byKind[kind]
  if (!s) return ''
  if (pin in s.dataIn) return s.dataIn[pin]
  if (s.dataInDynamicFn) {
    const dyn = s.dataInDynamicFn(cfg)
    if (pin in dyn) return dyn[pin]
  }
  return ''
}

/** Returns the PinType of (kind, dataOut pin) or '' if missing. Static only. */
export function dataOutTypeFor(kind: string, pin: string): PinType | '' {
  const s = byKind[kind]
  if (!s) return ''
  return s.dataOut[pin] ?? ''
}

/**
 * Derives edge type from (from-kind, from-pin) — never stored as edge field.
 * Mirrors backend validator logic. A future Phase C task deletes GraphEdge.kind
 * field entirely; this helper makes that change trivial.
 */
export function edgeKindOf(fromKind: string, fromPin: string): 'data' | 'exec' {
  return dataOutTypeFor(fromKind, fromPin) !== '' ? 'data' : 'exec'
}
