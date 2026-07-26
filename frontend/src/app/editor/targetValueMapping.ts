import type { TargetColorRange, TargetPoint, TargetRegion } from './useTargetPicker'

export type CoordinateUnit = 'ratio' | 'px'
export type PointValue = { x: number; y: number; unit: CoordinateUnit }
export type RegionValue = {
  x: number
  y: number
  width: number
  height: number
  unit: CoordinateUnit
}
export type ColorSpace = 'rgb' | 'hsv'
export type ColorRangeValue = {
  space: ColorSpace
  minimum: [number, number, number]
  maximum: [number, number, number]
}

export function pointValueFromTarget(picked: TargetPoint, unit: CoordinateUnit): PointValue {
  return unit === 'px'
    ? { x: picked.x, y: picked.y, unit }
    : { x: picked.xRatio, y: picked.yRatio, unit }
}

export function regionValueFromTarget(picked: TargetRegion, unit: CoordinateUnit): RegionValue {
  return unit === 'px'
    ? { x: picked.x, y: picked.y, width: picked.w, height: picked.h, unit }
    : {
        x: picked.region[0],
        y: picked.region[1],
        width: picked.region[2],
        height: picked.region[3],
        unit,
      }
}

export function colorRangeValueFromTarget(
  picked: TargetColorRange,
  space: ColorSpace,
): ColorRangeValue {
  return {
    space,
    minimum: [picked.range[0], picked.range[2], picked.range[4]],
    maximum: [picked.range[1], picked.range[3], picked.range[5]],
  }
}
