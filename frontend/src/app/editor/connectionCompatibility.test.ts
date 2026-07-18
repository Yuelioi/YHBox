import { describe, expect, it } from 'vitest'
import authoringDocument from '../../../../contracts/node/3.1/builtin-authoring.json'
import parityDocument from '../../../../internal/workflow/compiler/testdata/connection_plan_parity.json'
import type {
  NodeProjection,
  TypeExpression,
  YottaNodeAuthoringProjection,
} from '../../../../contracts/node/3.1/authoring-projection'
import {
  compatibleCandidatePorts,
  projectedConnectionCompatibility,
  typeMatch,
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

  it('matches the Compiler direct-connection fixture', () => {
    expect(parityDocument.version).toBe(1)
    expect(parityDocument.cases.length).toBeGreaterThan(0)
    expect(parityDocument.cases.length).toBeLessThanOrEqual(64)
    for (const test of parityDocument.cases) {
      const output = resolveParityExpression(test.output)
      const input = resolveParityExpression(test.input)
      expect(typeMatch(output, input, types) ?? 'invalid', test.id).toBe(test.expected)
    }
  })
})

interface ParityExpression {
  kind: string
  name?: string
  variable?: string
  constraints?: string[]
  staleDigest?: boolean
  element?: ParityExpression
  members?: ParityExpression[]
}

function resolveParityExpression(source: ParityExpression, depth = 0): TypeExpression {
  if (depth > 16) throw new Error('connection parity fixture exceeds type depth budget')
  switch (source.kind) {
    case 'named': {
      const projection = authoring.body.types.find((type) =>
        type.typeRef.typeId.includes(`/core/${source.name}/`),
      )
      if (!projection) throw new Error(`unknown fixture type ${source.name}`)
      const ref = structuredClone(projection.typeRef)
      if (source.staleDigest)
        ref.semanticDigest =
          'sha256:0000000000000000000000000000000000000000000000000000000000000000'
      return { kind: 'ref', ref }
    }
    case 'variable':
      if (!source.variable) throw new Error('fixture variable omitted its name')
      return { kind: 'variable', variable: source.variable, constraints: source.constraints ?? [] }
    case 'list':
      if (!source.element) throw new Error('fixture list omitted its element')
      return { kind: 'list', element: resolveParityExpression(source.element, depth + 1) }
    case 'union': {
      const members = (source.members ?? []).map((member) =>
        resolveParityExpression(member, depth + 1),
      )
      if (members.length < 2) throw new Error('fixture union requires two members')
      return {
        kind: 'union',
        members: members as [TypeExpression, TypeExpression, ...TypeExpression[]],
      }
    }
    default:
      throw new Error(`unknown fixture expression kind ${source.kind}`)
  }
}
