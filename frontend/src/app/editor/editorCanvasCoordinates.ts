export interface CanvasViewport {
  x: number
  y: number
  zoom: number
}

export function flowGuideToCanvasCoordinate(
  guide: number,
  viewport: CanvasViewport,
  axis: 'x' | 'y',
): number {
  return guide * viewport.zoom + viewport[axis]
}

export function canvasCenteredInsertionPosition(
  bounds: { left: number; top: number; width: number; height: number },
  screenToFlowCoordinate: (point: { x: number; y: number }) => { x: number; y: number },
): { x: number; y: number } {
  return screenToFlowCoordinate({
    x: bounds.left + bounds.width / 2,
    y: bounds.top + bounds.height * 0.38,
  })
}
