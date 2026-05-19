// frontend/src/components/containers/nodeFieldSchemas.ts
//
// Thin compatibility shell. All field schemas live in nodeRegistry/specs/*.ts
// (each kind's `fields: FieldSchema[]`). This file exports the legacy
// NODE_FIELD_SCHEMAS map by deriving it from the registry, so existing
// callers (NodeInspector.vue etc.) keep working without edits.
//
// DO NOT add new entries here — put them in nodeRegistry/specs/<group>.ts.
import '@/components/containers/nodeRegistry/specs'
import { allSpecs } from '@/components/containers/nodeRegistry/registry'
import type { FieldSchema } from '@/components/containers/nodeRegistry/index'

export type { FieldSchema }
/** Legacy alias — NodeInspector.vue imports `Field`. */
export type Field = FieldSchema

export const NODE_FIELD_SCHEMAS: Record<string, FieldSchema[]> = (() => {
  const out: Record<string, FieldSchema[]> = {}
  for (const s of allSpecs()) out[s.kind] = s.fields
  return out
})()
