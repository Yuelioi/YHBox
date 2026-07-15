import { describe, expect, it } from 'vitest'
import { builtinNodeProjections31, builtinTypeProjections31 } from './node31'
import { edgeKindOf } from '@/components/containers/nodeRegistry/registry'
import { pinsFor } from '@/components/containers/pinSpec'

const concatID = 'https://schemas.yotta.dev/nodes/text/concat/v1'
const stringID = 'https://schemas.yotta.dev/types/core/string/v1'

describe('generated Node Contract 3.1 authoring projection', () => {
  it('projects Concat with nominal type and binding hints but no control pins', () => {
    const projection = builtinNodeProjections31.get(concatID)
    expect(projection).toBeDefined()

    expect(
      projection!.dataInputs.map(({ id, type, binding }) => ({
        id,
        typeLabel: type.label,
        bindingHint: binding,
      })),
    ).toEqual([
      { id: 'a', typeLabel: stringID, bindingHint: 'required' },
      { id: 'b', typeLabel: stringID, bindingHint: 'required' },
    ])
    expect(projection!.dataOutputs.map((port) => port.id)).toEqual(['result'])
    expect(projection!.signals).toEqual([])
    expect(projection!.availability).toBe('portable')
  })

  it('drives the production canvas pin and edge selectors without inventing out', () => {
    expect(pinsFor(concatID)).toEqual({
      execIn: [],
      execOut: [],
      errorOut: [],
      statusOut: [],
      dataIn: ['a', 'b'],
      dataOut: ['result'],
    })
    expect(builtinNodeProjections31.get(concatID)?.dataInputs[0].type.label).toBe(stringID)
    expect(builtinNodeProjections31.get(concatID)?.dataInputs[0].type.color).toBe('#8b5cf6')
    expect(builtinTypeProjections31.get(stringID)?.lifecycle).toBe('durable')
    expect(edgeKindOf(concatID, 'result')).toBe('data')
    expect(edgeKindOf(concatID, 'out')).toBe('')
  })

  it('keeps exec, error, and status as distinct canvas channels', () => {
    const syntheticID = 'https://schemas.yotta.dev/nodes/test/signals/v1'
    const projection = structuredClone(builtinNodeProjections31.get(concatID)!)
    projection.nodeRef.nodeTypeId = syntheticID
    projection.signals = [
      { id: 'next', channel: 'exec', direction: 'output' },
      { id: 'failed', channel: 'error', direction: 'output' },
      { id: 'progress', channel: 'status', direction: 'output' },
    ]
    builtinNodeProjections31.set(syntheticID, projection)
    try {
      expect(pinsFor(syntheticID)).toEqual(
        expect.objectContaining({
          execOut: ['next'],
          errorOut: ['failed'],
          statusOut: ['progress'],
        }),
      )
      expect(edgeKindOf(syntheticID, 'next')).toBe('exec')
      expect(edgeKindOf(syntheticID, 'failed')).toBe('error')
      expect(edgeKindOf(syntheticID, 'progress')).toBe('status')
    } finally {
      builtinNodeProjections31.delete(syntheticID)
    }
  })

  it('projects config constraints, runtime lifecycle, and exact capability hints', () => {
    const conversionID = 'https://schemas.yotta.dev/nodes/conversion/stream-to-blob/v1'
    const projection = builtinNodeProjections31.get(conversionID)
    expect(projection?.configFields).toEqual([
      expect.objectContaining({
        id: 'mediaType',
        control: 'text',
        required: true,
        hasDefault: false,
        constraints: expect.objectContaining({ minLength: 3, maxLength: 255 }),
      }),
    ])
    expect(projection?.availability).toBe('target-required')
    expect(projection?.dataInputs[0]).toEqual(expect.objectContaining({ carrier: 'runtime' }))
    expect(projection?.dataOutputs[0]).toEqual(expect.objectContaining({ carrier: 'durable' }))
    expect(projection?.capabilities.map((entry) => entry.targetSlot)).toEqual([
      'blob-store',
      'stream-session',
    ])
  })
})
