import { describe, expect, it } from 'vitest'
import {
  colorRangeValueFromTarget,
  pointValueFromTarget,
  regionValueFromTarget,
} from './targetValueMapping'

describe('target value mapping', () => {
  it('maps point and region picker results into explicit ratio or pixel values', () => {
    const point = { x: 640, y: 360, xRatio: 0.5, yRatio: 0.5, screenW: 1280, screenH: 720 }
    const region = {
      x: 320,
      y: 180,
      w: 640,
      h: 360,
      region: [0.25, 0.25, 0.5, 0.5] as [number, number, number, number],
      screenW: 1280,
      screenH: 720,
    }

    expect(pointValueFromTarget(point, 'ratio')).toEqual({ x: 0.5, y: 0.5, unit: 'ratio' })
    expect(pointValueFromTarget(point, 'px')).toEqual({ x: 640, y: 360, unit: 'px' })
    expect(regionValueFromTarget(region, 'ratio')).toEqual({
      x: 0.25,
      y: 0.25,
      width: 0.5,
      height: 0.5,
      unit: 'ratio',
    })
    expect(regionValueFromTarget(region, 'px')).toEqual({
      x: 320,
      y: 180,
      width: 640,
      height: 360,
      unit: 'px',
    })
  })

  it('maps the picker interleaved color range without mutating the result', () => {
    const picked = {
      range: [10, 20, 30, 40, 50, 60] as [number, number, number, number, number, number],
      hueWrap: false,
    }

    expect(colorRangeValueFromTarget(picked, 'hsv')).toEqual({
      space: 'hsv',
      minimum: [10, 30, 50],
      maximum: [20, 40, 60],
    })
    expect(picked.range).toEqual([10, 20, 30, 40, 50, 60])
  })
})
