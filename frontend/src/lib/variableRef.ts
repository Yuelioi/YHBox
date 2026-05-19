// VariableRef — unified schema for variable references across GetVar/SetVar/IncVar nodes
// and editor surfaces (VarsPanel / FavoritesPanel / Promote-to-Var op).
// Spec: debug/docs/superpowers/specs/2026-05-19-editor-v2-vars-panel-design.md §4.7

export type VarType = 'number' | 'string' | 'bool' | 'point' | 'any'

export type VarScope = 'auto' | 'global' | 'local'

export interface VariableRef {
  name: string
  type: VarType
  // Only for GetVar/SetVar/IncVar node references; var decl itself has no scope.
  scope?: VarScope
}

export interface VarDeclLike {
  name: string
  type: string
  default?: unknown
}

export function declToRef(v: VarDeclLike): VariableRef {
  return { name: v.name, type: v.type as VarType }
}

export function isCompatibleType(src: VarType, dst: VarType): boolean {
  if (src === 'any' || dst === 'any') return true
  return src === dst
}
