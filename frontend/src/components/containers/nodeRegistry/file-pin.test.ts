import { describe, expect, it } from 'vitest'
import { adaptSpec } from './adapter'
import { TYPE_COLOR, pinTypeCompat } from './index'

describe('File pin type', () => {
  it('maps backend File pins to frontend file pins', () => {
    const spec = adaptSpec({
      kind: 'TestFileNode',
      category: 'IO',
      inputs: [{ name: 'File', type: 'File' }],
      outputs: [{ name: 'Done', type: 'Exec', data: [{ name: 'File', type: 'File' }] }],
    } as any)

    expect(spec.dataIn.File).toBe('file')
    expect(spec.dataOut.File).toBe('file')
    expect(TYPE_COLOR.file).toBeTruthy()
  })

  it('keeps file pins distinct except for any', () => {
    expect(pinTypeCompat('file', 'file')).toEqual({ allow: true, warn: false })
    expect(pinTypeCompat('file', 'string')).toEqual({ allow: false, warn: false })
    expect(pinTypeCompat('file', 'any')).toEqual({ allow: true, warn: false })
  })
})
