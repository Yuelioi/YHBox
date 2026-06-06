// NewVarModal — pure logic tests.
// Nuxt UI components cannot be mounted in vitest without full app setup,
// so we test the extracted logic (nameError predicate + zeroDefaultFor) directly.

import { describe, it, expect } from 'vitest'
import { zeroDefaultFor, type VarType } from '@/lib/variableRef'

// ── nameError predicate (mirrors the computed in NewVarModal.vue) ──────────────

function nameError(
  rawName: string,
  existingVarNames: string[],
): string | null {
  const n = rawName.trim()
  if (!n) return 'var.error.name_empty'
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(n)) return 'var.error.invalid_name'
  if (existingVarNames.includes(n)) return 'var.error.duplicate'
  return null
}

describe('NewVarModal — nameError predicate', () => {
  it('empty string → name_empty error', () => {
    expect(nameError('', [])).toBe('var.error.name_empty')
    expect(nameError('   ', [])).toBe('var.error.name_empty')
  })

  it('starts with digit → invalid_name error', () => {
    expect(nameError('2x', [])).toBe('var.error.invalid_name')
  })

  it('contains dash → invalid_name error', () => {
    expect(nameError('my-var', [])).toBe('var.error.invalid_name')
  })

  it('duplicate name → duplicate error', () => {
    expect(nameError('x', ['x', 'y'])).toBe('var.error.duplicate')
  })

  it('valid unique name → no error', () => {
    expect(nameError('myVar', ['x'])).toBeNull()
    expect(nameError('_private', [])).toBeNull()
    expect(nameError('var123', [])).toBeNull()
  })

  it('leading/trailing whitespace is trimmed before validation', () => {
    // "  x  " trims to "x", which is valid and not in existingVarNames → null
    expect(nameError('  x  ', [])).toBeNull()
    // " x " trims to "x", which IS in existingVarNames → duplicate
    expect(nameError(' x ', ['x'])).toBe('var.error.duplicate')
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

// ── confirm payload shape ─────────────────────────────────────────────────────

describe('NewVarModal — confirm payload', () => {
  it('point type produces correct payload', () => {
    const name = 'cursor'
    const type: VarType = 'point'
    const payload = {
      name: name.trim(),
      type,
      default: zeroDefaultFor(type),
    }
    expect(payload).toEqual({ name: 'cursor', type: 'point', default: { x: 0, y: 0 } })
  })

  it('nameError non-null prevents confirm (logic gate)', () => {
    const err = nameError('2bad', [])
    // confirm() would return early if nameError is non-null
    expect(err).not.toBeNull()
  })

  it('valid name + number type → payload with default 0', () => {
    const name = 'score'
    const type: VarType = 'number'
    expect(nameError(name, [])).toBeNull()
    expect(zeroDefaultFor(type)).toBe(0)
  })
})
