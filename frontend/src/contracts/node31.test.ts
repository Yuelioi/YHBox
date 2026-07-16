import { describe, expect, it } from 'vitest'
import { builtinNodeProjections31, builtinTypeProjections31 } from './node31'

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

  it('does not invent an exec out port for a pure-data node', () => {
    expect(builtinNodeProjections31.get(concatID)?.dataInputs[0].type.label).toBe(stringID)
    expect(builtinNodeProjections31.get(concatID)?.dataInputs[0].type.color).toBe('#8b5cf6')
    expect(builtinTypeProjections31.get(stringID)?.lifecycle).toBe('durable')
    expect(builtinNodeProjections31.get(concatID)?.dataOutputs.map((port) => port.id)).toEqual([
      'result',
    ])
    expect(builtinNodeProjections31.get(concatID)?.signals).toEqual([])
  })

  it('keeps exec and error as distinct canvas channels while status remains run metadata', () => {
    const syntheticID = 'https://schemas.yotta.dev/nodes/test/signals/v1'
    const projection = structuredClone(builtinNodeProjections31.get(concatID)!)
    projection.nodeRef.nodeTypeId = syntheticID
    projection.signals = [
      { id: 'next', channel: 'exec', direction: 'output' },
      { id: 'failed', channel: 'error', direction: 'output' },
    ]
    projection.statusEvents = [{ code: 'test.progress', category: 'progress' }]
    builtinNodeProjections31.set(syntheticID, projection)
    try {
      expect(builtinNodeProjections31.get(syntheticID)?.signals).toEqual([
        { id: 'next', channel: 'exec', direction: 'output' },
        { id: 'failed', channel: 'error', direction: 'output' },
      ])
      expect(projection.statusEvents).toEqual([{ code: 'test.progress', category: 'progress' }])
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
