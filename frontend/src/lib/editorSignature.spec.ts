import { describe, it, expect } from 'vitest'
import { exprSigContext, scriptPinValueContext, scriptSigContext } from './editorSignature'

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

describe('scriptPinValueContext', () => {
  it('cursor inside string value of a pin', () => {
    const doc = 'GetVar({Scope: "au"})'
    expect(scriptPinValueContext(doc, doc.indexOf('au') + 1)).toEqual({ kind: 'GetVar', pin: 'Scope' })
  })
  it('cursor right after colon (bare value position)', () => {
    const doc = 'GetVar({Scope: })'
    expect(scriptPinValueContext(doc, doc.indexOf(': ') + 2)).toEqual({ kind: 'GetVar', pin: 'Scope' })
  })
  it('varname pin value (VarName)', () => {
    const doc = 'SetVar({VarName: "h"})'
    expect(scriptPinValueContext(doc, doc.indexOf('h"') + 1)).toEqual({ kind: 'SetVar', pin: 'VarName' })
  })
  it('second pin in the object', () => {
    const doc = 'SetVar({VarName: "hp", Scope: ""})'
    expect(scriptPinValueContext(doc, doc.lastIndexOf('"'))).toEqual({ kind: 'SetVar', pin: 'Scope' })
  })
  it('null when cursor is on the key, not the value', () => {
    const doc = 'GetVar({Scope: ""})'
    expect(scriptPinValueContext(doc, doc.indexOf('Scope') + 2)).toBeNull()
  })
  it('null when object literal is not a call argument', () => {
    expect(scriptPinValueContext('let x = { a: 1 }', 13)).toBeNull()
  })
})
