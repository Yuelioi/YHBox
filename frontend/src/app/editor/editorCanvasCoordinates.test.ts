import { describe, expect, it } from 'vitest'
import {
  canvasCenteredInsertionPosition,
  centerElementAtFlowPosition,
  flowGuideToCanvasCoordinate,
} from './editorCanvasCoordinates'

describe('editor canvas coordinates', () => {
  it('projects a flow guide into the canvas-local overlay coordinate once', () => {
    expect(flowGuideToCanvasCoordinate(320, { x: 40, y: -20, zoom: 0.75 }, 'x')).toBe(280)
    expect(flowGuideToCanvasCoordinate(200, { x: 40, y: -20, zoom: 0.75 }, 'y')).toBe(130)
  })

  it('centers toolbar-created elements in the upper canvas viewport', () => {
    const position = canvasCenteredInsertionPosition(
      { left: 300, top: 100, width: 1000, height: 600 },
      (point) => ({ x: (point.x - 300 - 50) / 2, y: (point.y - 100 + 30) / 2 }),
    )

    expect(position).toEqual({ x: 225, y: 129 })
    expect(centerElementAtFlowPosition(position, { width: 260, height: 140 })).toEqual({
      x: 95,
      y: 59,
    })
  })
})
