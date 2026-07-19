export const WORKFLOW_CANVAS_INTERACTION = {
  selectionKeyCode: true,
  multiSelectionKeyCode: 'Control',
  panActivationKeyCode: 'Space',
  panOnDrag: [0, 1] as number[],
  selectNodesOnDrag: true,
}

export interface CanvasViewport {
  x: number
  y: number
  zoom: number
}

export function zoomViewportAtPoint(
  viewport: CanvasViewport,
  point: { x: number; y: number },
  deltaY: number,
  deltaMode: number,
  minZoom = 0.2,
  maxZoom = 2,
): CanvasViewport {
  const multiplier = deltaMode === 1 ? 0.05 : deltaMode ? 1 : 0.002
  const zoom = Math.max(minZoom, Math.min(maxZoom, viewport.zoom * 2 ** (-deltaY * multiplier)))
  if (Math.abs(zoom - viewport.zoom) < 0.000001) return viewport
  const graphX = (point.x - viewport.x) / viewport.zoom
  const graphY = (point.y - viewport.y) / viewport.zoom
  return {
    x: point.x - graphX * zoom,
    y: point.y - graphY * zoom,
    zoom,
  }
}

export function mergeMarqueeSelection(
  selectionBeforeDrag: ReadonlySet<string>,
  marqueeSelection: ReadonlySet<string>,
): Set<string> {
  return new Set([...selectionBeforeDrag, ...marqueeSelection])
}
