/** 吸管 range[6] → config.literal 值. tuple 直传数组拷贝; object 映射 hsv 六字段. */
export function fillColorLiteral(
  range: number[],
  schemaType: 'tuple' | 'object',
): number[] | Record<string, number> {
  if (schemaType === 'tuple') return [...range]
  return {
    hMin: range[0],
    hMax: range[1],
    sMin: range[2],
    sMax: range[3],
    vMin: range[4],
    vMax: range[5],
  }
}
