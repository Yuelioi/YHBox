import { describe, it, expect } from 'vitest'
import { declToRef, isCompatibleType, type VariableRef, type VarType } from './variableRef'

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
})
