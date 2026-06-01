// 从 nodeRegistry/specs/*.ts (各 kind 的 `fields: FieldSchema[]`) 派生 NODE_FIELD_SCHEMAS map.
// field schemas 的正源在 registry, 这里只 re-export.
//
// DO NOT add new entries here — put them in nodeRegistry/specs/<group>.ts.
// nodeRegistry/specs is imported once at app entry (main.ts). No side-effect needed here.
import { allSpecs } from '@/components/containers/nodeRegistry/registry'
import type { FieldSchema } from '@/components/containers/nodeRegistry/index'

export type { FieldSchema }
/** `Field` 别名 (NodeInspector.vue 用). */
export type Field = FieldSchema

// mutable empty, 由 rebuildNodeFieldSchemas() 在 RPC populate 后填.
// 原 module-init 写法在 RPC-driven 启动下时序错位 (allSpecs() 返空).
export const NODE_FIELD_SCHEMAS: Record<string, FieldSchema[]> = {}

export function rebuildNodeFieldSchemas(): void {
  for (const k of Object.keys(NODE_FIELD_SCHEMAS)) delete NODE_FIELD_SCHEMAS[k]
  for (const s of allSpecs()) NODE_FIELD_SCHEMAS[s.kind] = s.fields
}
