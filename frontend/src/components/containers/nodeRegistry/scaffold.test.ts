// Empty-registry sanity for B1. Real spec tests land in B2.
// beforeEach resets module state so tests don't depend on each other's order.
import { describe, it, expect, beforeEach } from 'vitest'
import {
  register,
  getSpec,
  allKinds,
  edgeKindOf,
  dataOutTypeFor,
  __resetForTests,
} from './registry'
import type { NodeKindSpec } from './index'

const minimalSpec: NodeKindSpec = {
  kind: '__TestKind',
  group: 'control',
  labelZh: 'test',
  description: '',
  example: '',
  visual: { icon: 'x', bg: '', border: '' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: { v: 'any' },
  fields: [],
  defaults: {},
}

describe('nodeRegistry scaffold', () => {
  beforeEach(() => {
    __resetForTests()
  })

  it('register + getSpec round-trip', () => {
    register(minimalSpec)
    expect(getSpec('__TestKind')).toEqual(minimalSpec)
    expect(allKinds()).toContain('__TestKind')
  })

  it('duplicate register throws', () => {
    register(minimalSpec)
    expect(() => register(minimalSpec)).toThrow(/duplicate kind/)
  })

  it('empty kind throws', () => {
    expect(() => register({ ...minimalSpec, kind: '' })).toThrow(/empty kind/)
  })

  it('edgeKindOf returns data for data-out pin', () => {
    register(minimalSpec)
    expect(edgeKindOf('__TestKind', 'v')).toBe('data')
    expect(edgeKindOf('__TestKind', 'out')).toBe('exec') // not a data-out
    expect(dataOutTypeFor('__TestKind', 'v')).toBe('any')
  })

  it('edgeKindOf returns exec for unknown kind', () => {
    expect(edgeKindOf('UnknownKind', 'pin')).toBe('exec')
  })
})
