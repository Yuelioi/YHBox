// VarNameInput — 纯逻辑单测
// 策略: 从组件文件导出 4 个纯决策函数, 直接 import 测试.
// Nuxt UI 组件无法在 vitest 环境 mount, 与 NewVarModal.spec.ts 保持同一惯例.

import { describe, it, expect } from 'vitest'
import { filterVars, shouldShowCreate, computeConflict, resolveOnBlur } from './VarNameInput.helpers'
import type { VarType } from '@/lib/variableRef'

// ── 测试数据 ─────────────────────────────────────────────────────────────────

const vars: { name: string; type: VarType }[] = [
  { name: 'count', type: 'number' },
  { name: 'label', type: 'string' },
  { name: 'flag', type: 'bool' },
  { name: 'pos', type: 'point' },
]

// ── §1 scope=local 退化 ───────────────────────────────────────────────────────
// scope=local 时组件渲染纯 UInput, 无下拉/新建/冲突警告.
// 纯函数层面: 不调用这 4 个函数. 通过 shouldShowCreate/filterVars 不被用来验证隔离效果.
// 此处用间接验证: scope=local 时这些纯函数若被误调用也不应崩溃.
describe('§1 scope=local 安全性 (纯函数层面不崩)', () => {
  it('filterVars 空 declaredVars 不崩', () => {
    expect(filterVars([], 'any')).toEqual([])
  })
  it('shouldShowCreate 空 names 不崩', () => {
    expect(shouldShowCreate('x', [])).toBe(true)
  })
})

// ── §2 filterVars: 下拉列表按 draft 子串过滤 ─────────────────────────────────
describe('§2 filterVars', () => {
  it('draft 空串 → 返回全部', () => {
    expect(filterVars(vars, '')).toHaveLength(4)
  })

  it('draft 空白串 → 返回全部', () => {
    expect(filterVars(vars, '   ')).toHaveLength(4)
  })

  it('精确匹配 → 只返回该项', () => {
    const r = filterVars(vars, 'count')
    expect(r).toHaveLength(1)
    expect(r[0].name).toBe('count')
  })

  it('大小写不敏感', () => {
    const r = filterVars(vars, 'COUNT')
    expect(r).toHaveLength(1)
    expect(r[0].name).toBe('count')
  })

  it('子串匹配 → 返回含该子串的多个项', () => {
    // 'la' 匹配 label; 'l' 匹配 label + flag
    const r = filterVars(vars, 'l')
    expect(r.map(v => v.name)).toContain('label')
    expect(r.map(v => v.name)).toContain('flag')
  })

  it('无匹配 → 返回空数组', () => {
    expect(filterVars(vars, 'zzz')).toHaveLength(0)
  })
})

// ── §3 shouldShowCreate: 新建项显隐 ──────────────────────────────────────────
describe('§3 shouldShowCreate', () => {
  const names = vars.map(v => v.name)

  it('合法且不重名 → true (显示新建)', () => {
    expect(shouldShowCreate('newVar', names)).toBe(true)
    expect(shouldShowCreate('_priv', names)).toBe(true)
  })

  it('重名 → false (不显示新建, 已有项走列表选)', () => {
    expect(shouldShowCreate('count', names)).toBe(false)
    expect(shouldShowCreate('label', names)).toBe(false)
  })

  it('空 draft → false', () => {
    expect(shouldShowCreate('', names)).toBe(false)
    expect(shouldShowCreate('   ', names)).toBe(false)
  })

  it('非法名 (数字开头) → false', () => {
    expect(shouldShowCreate('1bad', names)).toBe(false)
  })

  it('非法名 (含连字符) → false', () => {
    expect(shouldShowCreate('my-var', names)).toBe(false)
  })

  it('合法名 + 空 existingNames → true', () => {
    expect(shouldShowCreate('myVar', [])).toBe(true)
  })
})

// ── §4 captureType 新建 → zeroDefaultFor 正确 ────────────────────────────────
// computeConflict 不覆盖 declare 路径; 通过 zeroDefaultFor 直接验证 default 值正确性
import { zeroDefaultFor } from '@/lib/variableRef'

describe('§4 zeroDefaultFor (captureType 路径的 default)', () => {
  it('number → 0', () => expect(zeroDefaultFor('number')).toBe(0))
  it('string → ""', () => expect(zeroDefaultFor('string')).toBe(''))
  it('bool → false', () => expect(zeroDefaultFor('bool')).toBe(false))
  it('point → {x:0,y:0}', () => expect(zeroDefaultFor('point')).toEqual({ x: 0, y: 0 }))
  it('any → null', () => expect(zeroDefaultFor('any')).toBeNull())
})

// ── §5 无 captureType 新建 → 走 modal 路径 ───────────────────────────────────
// modal 路径由 captureType===undefined 控制; 纯函数层面通过 shouldShowCreate 验证入口,
// modalOpen=true 是组件内 onCreate() 中 else 分支的逻辑.
// 这里验证: 名合法+无 captureType 时 shouldShowCreate=true (onCreate 会触发 modal).
describe('§5 无 captureType → modal 路径入口条件', () => {
  it('合法新名 + 无 captureType → shouldShowCreate 为 true (组件走 modal)', () => {
    expect(shouldShowCreate('brand_new', ['x'])).toBe(true)
  })

  it('重名时 shouldShowCreate=false → onCreate 提前 return, 不开 modal', () => {
    expect(shouldShowCreate('x', ['x'])).toBe(false)
  })
})

// ── §6 resolveOnBlur: blur 时的还原/确认逻辑 ─────────────────────────────────
describe('§6 resolveOnBlur', () => {
  it('draft 精确匹配已有变量 → 返回该名 (应 emit update)', () => {
    expect(resolveOnBlur('count', vars)).toBe('count')
  })

  it('前后有空白的 draft 也能匹配 (trim)', () => {
    expect(resolveOnBlur('  label  ', vars)).toBe('label')
  })

  it('draft 不匹配任何变量 → 返 null (应还原为 modelValue)', () => {
    expect(resolveOnBlur('nonexistent', vars)).toBeNull()
  })

  it('空 draft → 返 null (不声明半截名)', () => {
    expect(resolveOnBlur('', vars)).toBeNull()
  })

  it('大小写不匹配 → 返 null (变量名区分大小写)', () => {
    // 'Count' !== 'count' → 不算精确匹配
    expect(resolveOnBlur('Count', vars)).toBeNull()
  })
})

// ── §7 computeConflict: 类型冲突警告 ─────────────────────────────────────────
describe('§7 computeConflict', () => {
  it('无 captureType → null (不警告)', () => {
    expect(computeConflict('count', vars, undefined)).toBeNull()
  })

  it('modelValue 不在 declaredVars → null', () => {
    expect(computeConflict('unknown', vars, 'string')).toBeNull()
  })

  it('captureType 与已有类型相同 → null (兼容)', () => {
    expect(computeConflict('count', vars, 'number')).toBeNull()
  })

  it('captureType=any → null (any 与任何类型兼容)', () => {
    expect(computeConflict('count', vars, 'any')).toBeNull()
  })

  it('已有类型=any → null (any 与任何类型兼容)', () => {
    const anyVars = [{ name: 'x', type: 'any' as VarType }]
    expect(computeConflict('x', anyVars, 'number')).toBeNull()
  })

  it('captureType=string, 已有 count:number → 返冲突对象', () => {
    const c = computeConflict('count', vars, 'string')
    expect(c).not.toBeNull()
    expect(c!.cap).toBe('string')
    expect(c!.dst).toBe('number')
  })

  it('captureType=bool, 已有 label:string → 返冲突对象', () => {
    const c = computeConflict('label', vars, 'bool')
    expect(c).not.toBeNull()
    expect(c!.cap).toBe('bool')
    expect(c!.dst).toBe('string')
  })

  it('captureType=point, 已有 flag:bool → 返冲突对象', () => {
    const c = computeConflict('flag', vars, 'point')
    expect(c).not.toBeNull()
    expect(c!.cap).toBe('point')
    expect(c!.dst).toBe('bool')
  })
})
