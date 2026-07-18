import { describe, expect, it } from 'vitest'
import authoringDocument from '../../../../contracts/node/3.1/builtin-authoring.json'
import type {
  NodeProjection,
  YottaNodeAuthoringProjection,
} from '../../../../contracts/node/3.1/authoring-projection'
import {
  compatibleCandidatePorts,
  projectedConnectionCompatibility,
} from './connectionCompatibility'

const authoring = authoringDocument as unknown as YottaNodeAuthoringProjection
const types = new Map(authoring.body.types.map((type) => [type.typeRef.typeId, type]))
const projection = (suffix: string): NodeProjection => {
  const result = authoring.body.nodes.find((node) => node.nodeRef.nodeTypeId.endsWith(suffix))
  if (!result) throw new Error(`missing projection ${suffix}`)
  return result
}

describe('connection compatibility', () => {
  it('finds compatible data inputs in stable port order', () => {
    const concat = projection('/text/concat')
    const ports = compatibleCandidatePorts(
      concat,
      { channel: 'data', direction: 'output', portId: 'result' },
      concat,
      types,
    )
    expect(ports.map((port) => port.handle.portId)).toEqual(['a', 'b'])
    expect(ports.every((port) => port.exact)).toBe(true)
    expect(ports.every((port) => port.match === 'exact')).toBe(true)
  })

  it('mirrors compiler resource lease narrowing', () => {
    const source = projection('/conversion/blob-to-stream')
    const target = structuredClone(projection('/conversion/stream-to-blob'))
    const valid = projectedConnectionCompatibility(
      source,
      { channel: 'data', direction: 'output', portId: 'stream' },
      target,
      { channel: 'data', direction: 'input', portId: 'stream' },
      types,
    )
    expect(valid.valid).toBe(true)

    target.dataInputs[0]!.resourceLease!.operations.push('stream/send')
    const widened = projectedConnectionCompatibility(
      source,
      { channel: 'data', direction: 'output', portId: 'stream' },
      target,
      { channel: 'data', direction: 'input', portId: 'stream' },
      types,
    )
    expect(widened).toMatchObject({ valid: false, issue: 'resource-lease' })
  })

  it('returns concrete source and target types for an incompatible drop', () => {
    const concat = projection('/text/concat')
    const greater = projection('/comparison/greater-than')
    expect(
      projectedConnectionCompatibility(
        concat,
        { channel: 'data', direction: 'output', portId: 'result' },
        greater,
        { channel: 'data', direction: 'input', portId: 'a' },
        types,
        authoring.body.nodes,
      ),
    ).toMatchObject({
      valid: false,
      issue: 'type',
      sourceType: 'string',
      targetType: 'number',
      disposition: 'conversion',
      reason: 'conversion-required',
      conversions: [
        expect.objectContaining({
          nodeTypeId: expect.stringMatching(/\/conversion\/string-to-integer$/),
          inputPort: 'text',
          outputPort: 'result',
          kind: 'parser',
          total: false,
          autoInsert: false,
        }),
        expect.objectContaining({
          nodeTypeId: expect.stringMatching(/\/conversion\/string-to-number$/),
          inputPort: 'text',
          outputPort: 'result',
          kind: 'parser',
          total: false,
          autoInsert: false,
        }),
      ],
    })
  })

  it('keeps exec and error channels distinct and instruction-aware', () => {
    const delay = projection('/control/delay')
    const retry = projection('/control/retry')
    expect(
      compatibleCandidatePorts(
        delay,
        { channel: 'error', direction: 'output', portId: 'failed' },
        retry,
        types,
      ).map((port) => port.handle),
    ).toEqual([{ channel: 'error', direction: 'input', portId: 'retry' }])
    expect(
      projectedConnectionCompatibility(
        delay,
        { channel: 'error', direction: 'output', portId: 'failed' },
        retry,
        { channel: 'exec', direction: 'input', portId: 'entry' },
        types,
      ),
    ).toMatchObject({ valid: false, issue: 'channel' })
  })

  it('uses Catalog relations and generic binding for a Repeat integer index', () => {
    const repeat = projection('/control/repeat')
    const greater = projection('/comparison/greater-than')
    const integerAdd = projection('/math/integer-add')
    const log = projection('/observability/log')
    const toString = projection('/conversion/to-string')
    const anchor = { channel: 'data', direction: 'output', portId: 'index' } as const

    expect(compatibleCandidatePorts(repeat, anchor, integerAdd, types)).toEqual([
      { handle: { channel: 'data', direction: 'input', portId: 'a' }, exact: true, match: 'exact' },
      { handle: { channel: 'data', direction: 'input', portId: 'b' }, exact: true, match: 'exact' },
    ])

    expect(compatibleCandidatePorts(repeat, anchor, greater, types)).toEqual([
      {
        handle: { channel: 'data', direction: 'input', portId: 'a' },
        exact: false,
        match: 'assignable',
      },
      {
        handle: { channel: 'data', direction: 'input', portId: 'b' },
        exact: false,
        match: 'assignable',
      },
    ])
    expect(compatibleCandidatePorts(repeat, anchor, toString, types)).toEqual([
      {
        handle: { channel: 'data', direction: 'input', portId: 'value' },
        exact: false,
        match: 'generic-bind',
      },
    ])
    expect(compatibleCandidatePorts(repeat, anchor, log, types)).toEqual([
      {
        handle: { channel: 'data', direction: 'input', portId: 'message' },
        exact: false,
        match: 'generic-bind',
      },
    ])
  })

  it('offers Catalog-generated break nodes for structured values', () => {
    const makePoint = projection('/geometry/make-point')
    const breakPoint = projection('/structure/break-point')
    expect(
      compatibleCandidatePorts(
        makePoint,
        { channel: 'data', direction: 'output', portId: 'result' },
        breakPoint,
        types,
      ),
    ).toEqual([
      {
        handle: { channel: 'data', direction: 'input', portId: 'value' },
        exact: true,
        match: 'exact',
      },
    ])
  })
})
