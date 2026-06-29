// VarNameInput 核心决策纯函数 — 从组件解耦以支持 vitest 单测.

import { validateVarName, isCompatibleType, type VarType } from '@/lib/variableRef'

/** 下拉列表过滤: 按 draft 子串(大小写不敏感)过滤 declaredVars.  draft 空串返全部. */
export function filterVars(
  declaredVars: { name: string; type: VarType }[],
  draft: string,
): { name: string; type: VarType }[] {
  const q = draft.trim().toLowerCase()
  if (!q) return declaredVars
  return declaredVars.filter(v => v.name.toLowerCase().includes(q))
}

/**
 * 是否显示「新建」项:
 * draft trim 非空 + validateVarName===null (合法 + 不重名)
 */
export function shouldShowCreate(draft: string, existingNames: string[]): boolean {
  return validateVarName(draft, existingNames) === null
}

/**
 * 类型冲突检测:有 captureType、modelValue 命中 declaredVars、且类型不兼容时返 {cap, dst}.
 * 否则返 null.
 */
export function computeConflict(
  modelValue: string,
  declaredVars: { name: string; type: VarType }[],
  captureType: string | undefined,
): { cap: string; dst: string } | null {
  if (!captureType) return null
  const v = declaredVars.find(x => x.name === modelValue)
  if (!v) return null
  if (!isCompatibleType(captureType as VarType, v.type)) return { cap: captureType, dst: v.type }
  return null
}

/**
 * blur 时的还原/确认逻辑:
 * - draft (trim) 精确匹配某 declaredVar → 返该名 (调用方应 emit update:modelValue)
 * - 否则返 null (调用方应还原 draft = modelValue, 不声明)
 */
export function resolveOnBlur(
  draft: string,
  declaredVars: { name: string; type: VarType }[],
): string | null {
  const n = draft.trim()
  const hit = declaredVars.find(v => v.name === n)
  return hit ? hit.name : null
}
