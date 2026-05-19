// Empty-registry sanity for B1. Real spec tests land in B2.
import { describe, it, expect } from 'vitest'
import { register, getSpec, allKinds, edgeKindOf, dataOutTypeFor } from './registry'
import type { NodeKindSpec } from './index'

const minimalSpec: NodeKindSpec = {
  kind: '__TestKind',
  group: 'control',
  labelZh: 'test',
  description: '',
  visual: { icon: 'x', bg: '', border: '' },
  execIn: ['in'],
  execOut: ['out'],
  dataIn: {},
  dataOut: { v: 'any' },
  fields: [],
  defaults: {},
}

describe('nodeRegistry scaffold', () => {
  it('register + getSpec round-trip', () => {
    register(minimalSpec)
    expect(getSpec('__TestKind')).toEqual(minimalSpec)
    expect(allKinds()).toContain('__TestKind')
  })

  it('duplicate register throws', () => {
    expect(() => register(minimalSpec)).toThrow(/duplicate kind/)
  })

  it('empty kind throws', () => {
    expect(() => register({ ...minimalSpec, kind: '' })).toThrow(/empty kind/)
  })

  it('edgeKindOf returns data for data-out pin', () => {
    expect(edgeKindOf('__TestKind', 'v')).toBe('data')
    expect(edgeKindOf('__TestKind', 'out')).toBe('exec') // not a data-out
    expect(dataOutTypeFor('__TestKind', 'v')).toBe('any')
  })

  it('edgeKindOf returns exec for unknown kind', () => {
    expect(edgeKindOf('UnknownKind', 'pin')).toBe('exec')
  })
})
