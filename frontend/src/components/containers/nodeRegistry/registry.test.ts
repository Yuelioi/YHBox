import { beforeEach, describe, expect, it } from 'vitest'
import { __resetForTests, execOutPinsFor, register } from './registry'
import type { NodeKindSpec } from './index'

describe('Node Contract 3.1 exec output projection', () => {
  beforeEach(__resetForTests)

  it('preserves an explicit empty exec output set for pure-data nodes', () => {
    register({
      kind: 'Concat',
      group: 'purefunc',
      labelZh: 'node.Concat.label',
      description: 'node.Concat.description',
      example: 'node.Concat.example',
      visual: { icon: 'function', bg: '', border: '' },
      execIn: [],
      execOut: [],
      dataIn: { A: 'string', B: 'string' },
      dataOut: { Result: 'string' },
      fields: [],
      defaults: {},
      isPureData: true,
    } satisfies NodeKindSpec)

    expect(execOutPinsFor('Concat', {})).toEqual([])
    expect(execOutPinsFor('missing-contract', {})).toEqual([])
  })
})
