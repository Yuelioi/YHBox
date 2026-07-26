import { describe, expect, it } from 'vitest'
import { builtinNodeProjections, builtinTypeProjections } from './node'

const concatID = 'https://schemas.yotta.dev/nodes/text/concat'
const stringID = 'https://schemas.yotta.dev/types/core/string/v1'

describe('generated current Node Contract authoring projection', () => {
  it('projects Concat with nominal type and binding hints but no control pins', () => {
    const projection = builtinNodeProjections.get(concatID)
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
    expect(builtinNodeProjections.get(concatID)?.dataInputs[0].type.label).toBe(stringID)
    expect(builtinNodeProjections.get(concatID)?.dataInputs[0].type.color).toBe('#8b5cf6')
    expect(builtinTypeProjections.get(stringID)?.lifecycle).toBe('durable')
    expect(builtinNodeProjections.get(concatID)?.dataOutputs.map((port) => port.id)).toEqual([
      'result',
    ])
    expect(builtinNodeProjections.get(concatID)?.signals).toEqual([])
  })

  it('keeps exec and error as distinct canvas channels while status remains run metadata', () => {
    const syntheticID = 'https://schemas.yotta.dev/nodes/test/signals'
    const projection = structuredClone(builtinNodeProjections.get(concatID)!)
    projection.nodeRef.nodeTypeId = syntheticID
    projection.signals = [
      { id: 'next', channel: 'exec', direction: 'output' },
      { id: 'failed', channel: 'error', direction: 'output' },
    ]
    projection.statusEvents = [{ code: 'test.progress', category: 'progress' }]
    builtinNodeProjections.set(syntheticID, projection)
    try {
      expect(builtinNodeProjections.get(syntheticID)?.signals).toEqual([
        { id: 'next', channel: 'exec', direction: 'output' },
        { id: 'failed', channel: 'error', direction: 'output' },
      ])
      expect(projection.statusEvents).toEqual([{ code: 'test.progress', category: 'progress' }])
    } finally {
      builtinNodeProjections.delete(syntheticID)
    }
  })

  it('projects config constraints, runtime lifecycle, and exact capability hints', () => {
    const conversionID = 'https://schemas.yotta.dev/nodes/conversion/stream-to-blob'
    const projection = builtinNodeProjections.get(conversionID)
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
