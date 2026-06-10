import { describe, it, expect } from 'vitest'
import { EXPR_FUNCTIONS, tokenAtCaret, unknownFnsIn } from './exprFunctions'

describe('EXPR_FUNCTIONS', () => {
  // 与 Go expr.Builtins() 同一预期表 (internal/services/expr/builtins_test.go) — 两侧测试用字面常量互锁.
  const WANT: Record<string, [number, number]> = {
    abs: [1, 1],
    min: [2, 2],
    max: [2, 2],
    now: [0, 0],
    floor: [1, 1],
    ceil: [1, 1],
    sqrt: [1, 1],
    round: [1, 2],
    pow: [2, 2],
    clamp: [3, 3],
    rand: [0, 0],
    randint: [2, 2],
  }

  it('matches the Go builtin set and arity', () => {
    expect(EXPR_FUNCTIONS.map(f => f.name).sort()).toEqual(Object.keys(WANT).sort())
    for (const f of EXPR_FUNCTIONS) {
      expect([f.minArgs, f.maxArgs], f.name).toEqual(WANT[f.name])
    }
  })

  it('every function has a signature starting with its name', () => {
    for (const f of EXPR_FUNCTIONS) {
      expect(f.sig.startsWith(`${f.name}(`), f.name).toBe(true)
    }
  })
})

describe('tokenAtCaret', () => {
  it('token at end of text', () => {
    expect(tokenAtCaret('1 + cla', 7)).toEqual({ token: 'cla', start: 4 })
  })
  it('token in the middle (caret inside word)', () => {
    expect(tokenAtCaret('clamp(x) + 1', 3)).toEqual({ token: 'cla', start: 0 })
  })
  it('empty at separator', () => {
    expect(tokenAtCaret('1 + ', 4)).toEqual({ token: '', start: 4 })
  })
  it('caret clamped to text length', () => {
    expect(tokenAtCaret('abs', 99)).toEqual({ token: 'abs', start: 0 })
  })
})

describe('unknownFnsIn', () => {
  it('flags typo function', () => {
    expect(unknownFnsIn('clmap(1, 2, 3)')).toEqual(['clmap'])
  })
  it('known functions pass', () => {
    expect(unknownFnsIn('clamp(abs(x), 0, max(1, 2))')).toEqual([])
  })
  it('ignores calls inside string literals', () => {
    expect(unknownFnsIn('"foo(1)" == s')).toEqual([])
  })
  it('identifier without paren is not a call', () => {
    expect(unknownFnsIn('clmap + 1')).toEqual([])
  })
  it('dedupes repeats', () => {
    expect(unknownFnsIn('foo(1) + foo(2)')).toEqual(['foo'])
  })
  it('allows space between name and paren', () => {
    expect(unknownFnsIn('foo (1)')).toEqual(['foo'])
  })
})
