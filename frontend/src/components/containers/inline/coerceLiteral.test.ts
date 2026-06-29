import { describe, it, expect } from 'vitest'
import { coerceLiteral, safeCoerceForFix } from './coerceLiteral'

describe('coerceLiteral', () => {
  it('number: 字符串数字 → number (杜绝 Sleep.Duration "13000" 类保存失败)', () => {
    expect(coerceLiteral('13000', 'number')).toBe(13000)
    expect(coerceLiteral(3000, 'number')).toBe(3000)
    expect(coerceLiteral('1.5', 'number')).toBe(1.5)
  })

  it('number: 非数字 / 空 → 0 (不放 NaN 进 literal)', () => {
    expect(coerceLiteral('abc', 'number')).toBe(0)
    expect(coerceLiteral('', 'number')).toBe(0)
    expect(coerceLiteral(null, 'number')).toBe(0)
  })

  it('bool → boolean', () => {
    expect(coerceLiteral('true', 'bool')).toBe(true)
    expect(coerceLiteral('', 'bool')).toBe(false)
    expect(coerceLiteral(1, 'bool')).toBe(true)
  })

  it('string: 标量 → String; null → ""', () => {
    expect(coerceLiteral(42, 'string')).toBe('42')
    expect(coerceLiteral(null, 'string')).toBe('')
    expect(coerceLiteral('hi', 'string')).toBe('hi')
  })

  it('string: 对象不糊成 [object Object], 原样透传 (防呆)', () => {
    const o = { x: 1 }
    expect(coerceLiteral(o, 'string')).toBe(o)
  })

  it('结构化 / any 类型 → 原样透传, 不碰', () => {
    const pt = { x: 0.5, y: 0.5 }
    expect(coerceLiteral(pt, 'point')).toBe(pt)
    expect(coerceLiteral('anything', 'any')).toBe('anything')
    expect(coerceLiteral(['a'], 'list')).toEqual(['a'])
  })
})

describe('safeCoerceForFix', () => {
  it('number: 干净数字字符串 → number', () => {
    expect(safeCoerceForFix('13000', 'number')).toBe(13000)
    expect(safeCoerceForFix('13.5', 'number')).toBe(13.5)
    expect(safeCoerceForFix('-5', 'number')).toBe(-5)
    expect(safeCoerceForFix(' 42 ', 'number')).toBe(42)
  })

  it('number: 非干净数字串不碰 → undefined (留给用户改, 绝不静默改 0)', () => {
    expect(safeCoerceForFix('500ms', 'number')).toBeUndefined()
    expect(safeCoerceForFix('abc', 'number')).toBeUndefined()
    expect(safeCoerceForFix('13000px', 'number')).toBeUndefined()
    expect(safeCoerceForFix('', 'number')).toBeUndefined()
    expect(safeCoerceForFix(13000, 'number')).toBeUndefined() // 已是 number, 不该被 flag
  })

  it('bool: 明确真假串/数字 → bool; 含糊 → undefined', () => {
    expect(safeCoerceForFix('true', 'bool')).toBe(true)
    expect(safeCoerceForFix('false', 'bool')).toBe(false)
    expect(safeCoerceForFix('1', 'bool')).toBe(true)
    expect(safeCoerceForFix(0, 'bool')).toBe(false)
    expect(safeCoerceForFix('yes', 'bool')).toBeUndefined()
  })

  it('string: 标量 → 字符串; 对象/空 不碰', () => {
    expect(safeCoerceForFix(42, 'string')).toBe('42')
    expect(safeCoerceForFix(true, 'string')).toBe('true')
    expect(safeCoerceForFix({ x: 1 }, 'string')).toBeUndefined()
    expect(safeCoerceForFix(null, 'string')).toBeUndefined()
  })

  it('结构化 / 未知期望类型 → 不机械修', () => {
    expect(safeCoerceForFix('1,2', 'point')).toBeUndefined()
    expect(safeCoerceForFix('x', 'any')).toBeUndefined()
  })
})
