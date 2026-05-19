// frontend/src/components/containers/nodeRegistry/export.ts
//
// Mirror of backend internal/services/container/nodekind/export.go ExportJSON.
// Both sides produce byte-identical JSON for the same set of registered specs
// (modulo the fields each side carries — IsYield is BE-only).
//
// D2 parity test consumes this: a vitest writes frontend dump to a file, Go
// test reads it + compares to backend nodekind.ExportJSON() output.
//
// CRITICAL: any field added here MUST be added to backend ExportedSpec too,
// AND the parity test's skip list updated if it's not meant to match across.

import { allSpecs } from './registry'
import type { PinType } from './index'

export interface ExportedSpec {
  kind: string
  group: string
  /** null when empty (matches Go's `omitempty` + json.Marshal of nil slice). */
  execIn: string[] | null
  execOut: string[] | null
  hasExecOutFn: boolean
  /** null when empty (matches Go side). */
  dataIn: Record<string, PinType> | null
  hasDataInDynamicFn: boolean
  dataOut: Record<string, PinType> | null
  isPureData: boolean
  isVisualOnly: boolean
}

/**
 * exportJSON returns all registered specs as a stable JSON string,
 * sorted by kind ascending. Designed to byte-match backend nodekind.ExportJSON.
 */
export function exportJSON(): string {
  const out: ExportedSpec[] = allSpecs().map((s) => ({
    kind: s.kind,
    group: s.group,
    execIn: s.execIn.length > 0 ? s.execIn : null,
    execOut: s.execOut.length > 0 ? s.execOut : null,
    hasExecOutFn: !!s.execOutFn,
    dataIn: Object.keys(s.dataIn).length > 0 ? sortKeys(s.dataIn) : null,
    hasDataInDynamicFn: !!s.dataInDynamicFn,
    dataOut: Object.keys(s.dataOut).length > 0 ? sortKeys(s.dataOut) : null,
    isPureData: !!s.isPureData,
    isVisualOnly: !!s.isVisualOnly,
  }))
  out.sort((a, b) => (a.kind < b.kind ? -1 : a.kind > b.kind ? 1 : 0))
  // JSON.stringify with 2-space indent matches Go's json.MarshalIndent("", "  ").
  // Go marshals maps with keys sorted alphabetically; we mirror via sortKeys above.
  return JSON.stringify(out, null, 2)
}

/**
 * sortKeys returns a new object with keys sorted alphabetically — necessary
 * because Go's encoding/json marshals maps with sorted keys, but
 * JavaScript's JSON.stringify uses insertion order. Without this, FE and BE
 * dumps diff on map key order even with identical content.
 */
function sortKeys<T>(obj: Record<string, T>): Record<string, T> {
  const result: Record<string, T> = {}
  for (const k of Object.keys(obj).sort()) result[k] = obj[k]
  return result
}
