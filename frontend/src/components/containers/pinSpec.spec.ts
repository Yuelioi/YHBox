import { describe, expect, it, beforeEach } from 'vitest'
import { __resetForTests } from './nodeRegistry/registry'
import { pinsFor, rebuildPinSpecMaps } from './pinSpec'

describe('pinsFor unknown kinds', () => {
  beforeEach(() => {
    __resetForTests()
    rebuildPinSpecMaps()
  })

  it('does not invent lowercase in/out pins for unknown node kinds', () => {
    expect(pinsFor('UnknownNode')).toEqual({
      execIn: [],
      execOut: [],
      errorOut: [],
      statusOut: [],
      dataIn: [],
      dataOut: [],
    })
  })

  it('keeps virtual subgraph marker pins explicit', () => {
    expect(pinsFor('SubgraphInput').execOut).toEqual(['Done'])
    expect(pinsFor('SubgraphOutput').execIn).toEqual(['In'])
  })
})
