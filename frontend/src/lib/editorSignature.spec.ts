import { describe, it, expect } from 'vitest'
import { exprSigContext, scriptSigContext } from './editorSignature'

describe('exprSigContext', () => {
  it('inside first arg', () => {
    expect(exprSigContext('abs(', 4)).toEqual({ name: 'abs', argIndex: 0 })
  })
  it('counts commas to current arg', () => {
    expect(exprSigContext('clamp(1, ', 9)).toEqual({ name: 'clamp', argIndex: 1 })
  })
  it('skips closed nested call', () => {
    expect(exprSigContext('max(min(1,2), ', 14)).toEqual({ name: 'max', argIndex: 1 })
  })
  it('ignores comma inside string', () => {
    expect(exprSigContext('find("a, b", ', 12)).toEqual({ name: 'find', argIndex: 1 })
  })
  it('null when not in a call', () => {
    expect(exprSigContext('1 + 2', 5)).toBeNull()
  })
})

describe('scriptSigContext', () => {
  it('bare call first arg', () => {
    expect(scriptSigContext('ClickAt({', 9)).toEqual({ name: 'ClickAt', argIndex: 0 })
  })
  it('member call counts args', () => {
    expect(scriptSigContext('vars.get("hp", ', 15)).toEqual({ name: 'vars.get', argIndex: 1 })
  })
  it('outer call after closed inner', () => {
    expect(scriptSigContext('log.info(vars.get("a"), ', 24)).toEqual({ name: 'log.info', argIndex: 1 })
  })
  it('null outside any call', () => {
    expect(scriptSigContext('let x = 1', 9)).toBeNull()
  })
})
