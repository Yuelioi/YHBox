// RGB↔HSV — 前端版, 跟 Go pkg/vision.RGBToHSV 同一套约定: H ∈ [0,360], S/V ∈ [0,100].
// Screenshot color-pick values use the same scale as Workflow color analysis.
// 改这里前先看 pkg/vision/color.go, 别让前后端漂移.

export interface HSV {
  h: number
  s: number
  v: number
}

// rgbToHsv 8-bit RGB (0-255) → HSV (H 0-360, S/V 0-100). 取整跟 Go 侧一致 (截断, 非四舍五入).
export function rgbToHsv(r: number, g: number, b: number): HSV {
  const rf = r / 255
  const gf = g / 255
  const bf = b / 255
  const maxC = Math.max(rf, gf, bf)
  const minC = Math.min(rf, gf, bf)
  const delta = maxC - minC

  const v = Math.trunc(maxC * 100)
  if (maxC === 0) return { h: 0, s: 0, v }
  const s = Math.trunc((delta / maxC) * 100)

  let hf: number
  if (delta === 0) hf = 0
  else if (maxC === rf) hf = 60 * (((gf - bf) / delta) % 6)
  else if (maxC === gf) hf = 60 * ((bf - rf) / delta + 2)
  else hf = 60 * ((rf - gf) / delta + 4)
  if (hf < 0) hf += 360

  return { h: Math.trunc(hf), s, v }
}

// rgbToHex 8-bit RGB → '#RRGGBB' (大写, 跟 Go 侧 fmt '%02X' 一致).
export function rgbToHex(r: number, g: number, b: number): string {
  const hx = (n: number) => n.toString(16).padStart(2, '0').toUpperCase()
  return `#${hx(r)}${hx(g)}${hx(b)}`
}
