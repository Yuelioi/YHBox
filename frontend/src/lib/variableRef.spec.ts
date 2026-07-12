import { describe, it, expect } from 'vitest'
import {
  declToRef,
  isCompatibleType,
  parseListDraft,
  zeroDefaultFor,
  VAR_TYPE_VALUES,
} from './variableRef'

describe('VariableRef', () => {
  it('declToRef strips runtime fields', () => {
    const decl = { name: 'x', type: 'number' as const, default: 1 }
    expect(declToRef(decl)).toEqual({ name: 'x', type: 'number' })
  })

  it('isCompatibleType: any matches everything', () => {
    expect(isCompatibleType('any', 'number')).toBe(true)
    expect(isCompatibleType('number', 'any')).toBe(true)
  })

  it('isCompatibleType: exact match', () => {
    expect(isCompatibleType('number', 'number')).toBe(true)
    expect(isCompatibleType('string', 'number')).toBe(false)
  })

  it('declToRef throws on unknown type (system boundary guard)', () => {
    expect(() => declToRef({ name: 'x', type: 'integer' as unknown as string })).toThrow(
      /unknown VarType/i,
    )
  })

  it('list is a declared VarType with [] zero default', () => {
    expect(VAR_TYPE_VALUES).toContain('list')
    expect(zeroDefaultFor('list')).toEqual([])
    expect(declToRef({ name: 'items', type: 'list', default: [1, 2] })).toEqual({
      name: 'items',
      type: 'list',
    })
  })

  it('isCompatibleType: list matches list/any only', () => {
    expect(isCompatibleType('list', 'list')).toBe(true)
    expect(isCompatibleType('list', 'any')).toBe(true)
    expect(isCompatibleType('any', 'list')).toBe(true)
    expect(isCompatibleType('list', 'number')).toBe(false)
    expect(isCompatibleType('string', 'list')).toBe(false)
  })

  it('parseListDraft: valid JSON array commits, anything else rejects', () => {
    expect(parseListDraft('[1, 2, "a"]')).toEqual({ ok: true, value: [1, 2, 'a'] })
    expect(parseListDraft('[]')).toEqual({ ok: true, value: [] })
    expect(parseListDraft('[1,2')).toEqual({ ok: false })
    expect(parseListDraft('{"a":1}')).toEqual({ ok: false })
    expect(parseListDraft('"text"')).toEqual({ ok: false })
    expect(parseListDraft('')).toEqual({ ok: false })
  })
})
