// VariableRef — unified schema for variable references across GetVar/SetVar/IncVar nodes
// and editor surfaces (VarsPanel / FavoritesPanel / Promote-to-Var op).

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

// 枚举所有合法 VarType, 给 declToRef runtime guard 用 (system boundary 校验).
const VAR_TYPES = new Set<string>(['number', 'string', 'bool', 'point', 'any'])

export function declToRef(v: VarDeclLike): VariableRef {
  if (!VAR_TYPES.has(v.type)) {
    throw new Error(`declToRef: unknown VarType ${JSON.stringify(v.type)} for var ${JSON.stringify(v.name)}`)
  }
  return { name: v.name, type: v.type as VarType }
}

export function isCompatibleType(src: VarType, dst: VarType): boolean {
  if (src === 'any' || dst === 'any') return true
  return src === dst
}
