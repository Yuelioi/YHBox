import { describe, it, expect } from 'vitest'
import { fillColorLiteral } from './colorRange'

describe('fillColorLiteral', () => {
  const r = [0, 30, 40, 100, 50, 100]
  it('tuple → 原数组拷贝', () => {
    expect(fillColorLiteral(r, 'tuple')).toEqual(r)
    expect(fillColorLiteral(r, 'tuple')).not.toBe(r)
  })
  it('object → hsv 六字段', () => {
    expect(fillColorLiteral(r, 'object')).toEqual({
      hMin: 0, hMax: 30, sMin: 40, sMax: 100, vMin: 50, vMax: 100,
    })
  })
})
