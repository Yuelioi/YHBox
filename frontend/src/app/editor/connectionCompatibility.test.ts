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
    )
    expect(ports.map((port) => port.handle.portId)).toEqual(['a', 'b'])
    expect(ports.every((port) => port.exact)).toBe(true)
  })

  it('mirrors compiler resource lease narrowing', () => {
    const source = projection('/conversion/blob-to-stream')
    const target = structuredClone(projection('/conversion/stream-to-blob'))
    const valid = projectedConnectionCompatibility(
      source,
      { channel: 'data', direction: 'output', portId: 'stream' },
      target,
      { channel: 'data', direction: 'input', portId: 'stream' },
    )
    expect(valid.valid).toBe(true)

    target.dataInputs[0]!.resourceLease!.operations.push('stream/send')
    const widened = projectedConnectionCompatibility(
      source,
      { channel: 'data', direction: 'output', portId: 'stream' },
      target,
      { channel: 'data', direction: 'input', portId: 'stream' },
    )
    expect(widened).toMatchObject({ valid: false, issue: 'resource-lease' })
  })

  it('keeps exec and error channels distinct and instruction-aware', () => {
    const delay = projection('/control/delay')
    const retry = projection('/control/retry')
    expect(
      compatibleCandidatePorts(
        delay,
        { channel: 'error', direction: 'output', portId: 'failed' },
        retry,
      ).map((port) => port.handle),
    ).toEqual([{ channel: 'error', direction: 'input', portId: 'retry' }])
    expect(
      projectedConnectionCompatibility(
        delay,
        { channel: 'error', direction: 'output', portId: 'failed' },
        retry,
        { channel: 'exec', direction: 'input', portId: 'entry' },
      ),
    ).toMatchObject({ valid: false, issue: 'channel' })
  })
})
