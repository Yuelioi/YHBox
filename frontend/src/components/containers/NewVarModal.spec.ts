// NewVarModal — pure logic tests.
// Nuxt UI components cannot be mounted in vitest without full app setup,
// so we test the extracted logic (validateVarName + zeroDefaultFor) directly.

import { describe, it, expect } from 'vitest'
import { validateVarName, zeroDefaultFor, type VarType } from '@/lib/variableRef'

// ── validateVarName ───────────────────────────────────────────────────────────

describe('validateVarName', () => {
  it('empty string → name_empty', () => {
    expect(validateVarName('', [])).toBe('var.error.name_empty')
    expect(validateVarName('   ', [])).toBe('var.error.name_empty')
  })

  it('starts with digit → invalid_name', () => {
    expect(validateVarName('2x', [])).toBe('var.error.invalid_name')
  })

  it('contains dash → invalid_name', () => {
    expect(validateVarName('my-var', [])).toBe('var.error.invalid_name')
  })

  it('duplicate name → duplicate', () => {
    expect(validateVarName('x', ['x', 'y'])).toBe('var.error.duplicate')
  })

  it('valid unique name → null', () => {
    expect(validateVarName('myVar', ['x'])).toBeNull()
    expect(validateVarName('_private', [])).toBeNull()
    expect(validateVarName('var123', [])).toBeNull()
  })

  it('leading/trailing whitespace is trimmed before validation', () => {
    expect(validateVarName('  x  ', [])).toBeNull()
    expect(validateVarName(' x ', ['x'])).toBe('var.error.duplicate')
  })

  it('allowSelf: self name does not count as duplicate', () => {
    expect(validateVarName('x', ['x', 'y'], 'x')).toBeNull()
  })

  it('allowSelf: other duplicate still caught', () => {
    expect(validateVarName('y', ['x', 'y'], 'x')).toBe('var.error.duplicate')
  })
})

// ── zeroDefaultFor ────────────────────────────────────────────────────────────

describe('zeroDefaultFor', () => {
  const cases: [VarType, unknown][] = [
    ['number', 0],
    ['string', ''],
    ['bool', false],
    ['point', { x: 0, y: 0 }],
    ['any', null],
  ]

  it.each(cases)('type=%s → %j', (type, expected) => {
    expect(zeroDefaultFor(type)).toEqual(expected)
  })
})
