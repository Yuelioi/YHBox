import { describe, it, expect } from 'vitest'
import { setExprFunctions, allExprFunctions, exprFnNames, tokenAtCaret, unknownFnsIn, lintExpr, parseSigParams } from './exprFunctions'

// 函数表集合/arity 的权威测试在 Go 侧 (builtins_test.go TestFunctions_DTO) —
// FE 不再手写表, 这里只测 store 喂表 + 纯函数逻辑.

describe('setExprFunctions', () => {
  it('populates list and name set', () => {
    setExprFunctions([
      { name: 'clamp', sig: 'clamp(x, min, max)', minArgs: 3, maxArgs: 3 },
      { name: 'rand', sig: 'rand()', minArgs: 0, maxArgs: 0 },
    ])
    expect(allExprFunctions().map(f => f.name)).toEqual(['clamp', 'rand'])
    expect(exprFnNames().has('clamp')).toBe(true)
    expect(exprFnNames().has('clmap')).toBe(false)
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
  const KNOWN = new Set(['clamp', 'abs', 'max'])

  it('flags typo function', () => {
    expect(unknownFnsIn('clmap(1, 2, 3)', KNOWN)).toEqual(['clmap'])
  })
  it('known functions pass', () => {
    expect(unknownFnsIn('clamp(abs(x), 0, max(1, 2))', KNOWN)).toEqual([])
  })
  it('ignores calls inside string literals', () => {
    expect(unknownFnsIn('"foo(1)" == s', KNOWN)).toEqual([])
  })
  it('identifier without paren is not a call', () => {
    expect(unknownFnsIn('clmap + 1', KNOWN)).toEqual([])
  })
  it('dedupes repeats', () => {
    expect(unknownFnsIn('foo(1) + foo(2)', KNOWN)).toEqual(['foo'])
  })
  it('allows space between name and paren', () => {
    expect(unknownFnsIn('foo (1)', KNOWN)).toEqual(['foo'])
  })
})

describe('parseSigParams', () => {
  it('splits arg names', () => {
    expect(parseSigParams('clamp(x, lo, hi)')).toEqual(['x', 'lo', 'hi'])
  })
  it('zero args → empty', () => {
    expect(parseSigParams('now()')).toEqual([])
  })
  it('strips optional brackets/space', () => {
    expect(parseSigParams('vars.get(name, [scope])')).toEqual(['name', 'scope'])
  })
})

describe('lintExpr', () => {
  const KNOWN = new Set(['clamp', 'abs'])

  it('valid expression → no diagnostics', () => {
    expect(lintExpr('clamp(abs(x), 0, 10) > 5', KNOWN)).toEqual([])
  })
  it('extra closing paren points at the offending paren', () => {
    const d = lintExpr('abs(x))', KNOWN)
    expect(d).toHaveLength(1)
    expect(d[0]).toMatchObject({ from: 6, to: 7, messageKey: 'expression.error.paren_mismatch' })
  })
  it('missing closing paren points at end', () => {
    const d = lintExpr('clamp(1, 2', KNOWN)
    expect(d).toHaveLength(1)
    expect(d[0].messageKey).toBe('expression.error.paren_missing')
    expect(d[0].from).toBe(10)
  })
  it('unclosed string points at opening quote', () => {
    const d = lintExpr('x == "abc', KNOWN)
    expect(d).toHaveLength(1)
    expect(d[0]).toMatchObject({ from: 5, messageKey: 'expression.error.string_unclosed' })
  })
  it('unknown function underlines the name', () => {
    const d = lintExpr('1 + clmap(1)', KNOWN)
    expect(d).toHaveLength(1)
    expect(d[0]).toMatchObject({ from: 4, to: 9, messageKey: 'expression.error.unknown_fn' })
    expect(d[0].params).toEqual({ name: 'clmap' })
  })
  it('trailing operator points at end', () => {
    const d = lintExpr('1 +', KNOWN)
    expect(d).toHaveLength(1)
    expect(d[0].messageKey).toBe('expression.error.op_end')
  })
  it('bare word covers the whole token', () => {
    const d = lintExpr('hello', KNOWN)
    expect(d).toHaveLength(1)
    expect(d[0]).toMatchObject({ from: 0, to: 5, messageKey: 'expression.error.bare_word' })
  })
  it('empty text → no diagnostics', () => {
    expect(lintExpr('', KNOWN)).toEqual([])
    expect(lintExpr('   ', KNOWN)).toEqual([])
  })
})
