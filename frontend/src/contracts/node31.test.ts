import { describe, expect, it } from 'vitest'
import { builtinNodeContracts31, builtinNodeProjections31, projectNodeContract31 } from './node31'
import { edgeKindOf } from '@/components/containers/nodeRegistry/registry'
import { pinsFor } from '@/components/containers/pinSpec'

const concatID = 'https://schemas.yotta.dev/nodes/text/concat/v1'
const stringID = 'https://schemas.yotta.dev/types/core/string/v1'

describe('generated Node Contract 3.1 authoring projection', () => {
  it('projects Concat with nominal type and binding hints but no control pins', () => {
    const contract = builtinNodeContracts31.get(concatID)
    expect(contract).toBeDefined()
    const projection = projectNodeContract31(contract!)

    expect(
      projection.dataInputs.map(({ id, typeLabel, bindingHint }) => ({
        id,
        typeLabel,
        bindingHint,
      })),
    ).toEqual([
      { id: 'a', typeLabel: stringID, bindingHint: 'required' },
      { id: 'b', typeLabel: stringID, bindingHint: 'required' },
    ])
    expect(projection.dataOutputs.map((port) => port.id)).toEqual(['result'])
    expect(projection.execInputs).toEqual([])
    expect(projection.execOutputs).toEqual([])
    expect(projection.errorOutputs).toEqual([])
    expect(projection.statusOutputs).toEqual([])
  })

  it('drives the production canvas pin and edge selectors without inventing out', () => {
    expect(pinsFor(concatID)).toEqual({
      execIn: [],
      execOut: [],
      dataIn: ['a', 'b'],
      dataOut: ['result'],
    })
    expect(builtinNodeProjections31.get(concatID)?.dataInputs[0].typeLabel).toBe(stringID)
    expect(edgeKindOf(concatID, 'result')).toBe('data')
    expect(edgeKindOf(concatID, 'out')).toBe('')
  })
})
