// VariableRef — unified schema for variable references across GetVar/SetVar/IncVar nodes
// and editor surfaces (VarsPanel / FavoritesPanel / Promote-to-Var op).

export type VarType = 'number' | 'string' | 'bool' | 'point' | 'list' | 'any'

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
const VAR_TYPES = new Set<string>(['number', 'string', 'bool', 'point', 'list', 'any'])

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

// 变量名校验 — 返 i18n key (未翻译) 或 null。
// allowSelf: 编辑已有变量时传当前名, 自身不算重名。
export function validateVarName(
  raw: string,
  existingNames: string[],
  allowSelf?: string,
): 'var.error.name_empty' | 'var.error.invalid_name' | 'var.error.duplicate' | null {
  const n = raw.trim()
  if (!n) return 'var.error.name_empty'
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(n)) return 'var.error.invalid_name'
  if (n !== allowSelf && existingNames.includes(n)) return 'var.error.duplicate'
  return null
}

export const VAR_TYPE_VALUES = ['number', 'string', 'bool', 'point', 'list', 'any'] as const satisfies VarType[]
export const VAR_TYPE_OPTIONS = VAR_TYPE_VALUES.map(v => ({ value: v, label: v }))

// 新建变量的零值 default — 对齐 VarDecl.Default (any 类型) 现有约定 (后端 model.go: point 是 map{x,y})。
export function zeroDefaultFor(varType: VarType): unknown {
  switch (varType) {
    case 'number': return 0
    case 'string': return ''
    case 'bool': return false
    case 'point': return { x: 0, y: 0 }
    case 'list': return []
    case 'any': return null
  }
}

// list 变量默认值草稿解析 — 只认 JSON 数组; 非法 JSON / 非数组一律拒 (调用方红框不落盘)。
export function parseListDraft(raw: string): { ok: true; value: unknown[] } | { ok: false } {
  try {
    const v = JSON.parse(raw)
    return Array.isArray(v) ? { ok: true, value: v } : { ok: false }
  } catch {
    return { ok: false }
  }
}
