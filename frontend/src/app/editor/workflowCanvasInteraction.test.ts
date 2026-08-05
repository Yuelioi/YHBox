import { describe, expect, it } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import { VueFlow } from '@vue-flow/core'
import {
  canvasOwnsWheelTarget,
  mergeMarqueeSelection,
  WORKFLOW_CANVAS_INTERACTION,
  zoomViewportAtPoint,
} from './workflowCanvasInteraction'

describe('workflow canvas interaction contract', () => {
  it('uses left drag for marquee and reserves Space, middle, or right drag for panning', () => {
    expect(WORKFLOW_CANVAS_INTERACTION.selectionKeyCode).toBe(true)
    expect(WORKFLOW_CANVAS_INTERACTION.panActivationKeyCode).toBe('Space')
    expect(WORKFLOW_CANVAS_INTERACTION.panOnDrag).toEqual([0, 1, 2])
    expect(WORKFLOW_CANVAS_INTERACTION.multiSelectionKeyCode).toBe('Control')
  })

  it('starts a Vue Flow marquee from an unmodified left drag', async () => {
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp({
      render: () =>
        h(VueFlow, {
          nodes: [],
          edges: [],
          selectionKeyCode: WORKFLOW_CANVAS_INTERACTION.selectionKeyCode,
          panActivationKeyCode: WORKFLOW_CANVAS_INTERACTION.panActivationKeyCode,
          panOnDrag: WORKFLOW_CANVAS_INTERACTION.panOnDrag,
        }),
    })

    try {
      app.mount(root)
      await nextTick()
      const pane = root.querySelector<HTMLElement>('.vue-flow__pane')
      expect(pane).not.toBeNull()
      if (!pane) return
      pane.getBoundingClientRect = () =>
        ({ left: 0, top: 0, right: 800, bottom: 600, width: 800, height: 600 }) as DOMRect

      pane.dispatchEvent(
        new PointerEvent('pointerdown', {
          bubbles: true,
          button: 0,
          clientX: 20,
          clientY: 20,
          pointerId: 1,
        }),
      )
      pane.dispatchEvent(
        new PointerEvent('pointermove', {
          bubbles: true,
          button: 0,
          buttons: 1,
          clientX: 180,
          clientY: 140,
          pointerId: 1,
        }),
      )
      await nextTick()

      expect(root.querySelector('.vue-flow__selection')).not.toBeNull()
    } finally {
      app.unmount()
      root.remove()
    }
  })

  it('adds a shift marquee to the selection captured before Vue Flow clears it', () => {
    expect(mergeMarqueeSelection(new Set(['first']), new Set(['second']))).toEqual(
      new Set(['first', 'second']),
    )
  })

  it('uses the marquee result directly when no additive selection was captured', () => {
    expect(mergeMarqueeSelection(new Set(), new Set(['second']))).toEqual(new Set(['second']))
  })

  it('zooms around the pointer while preserving the graph coordinate underneath it', () => {
    const viewport = { x: -40, y: 20, zoom: 2 }
    const point = { x: 300, y: 220 }
    const graphPoint = {
      x: (point.x - viewport.x) / viewport.zoom,
      y: (point.y - viewport.y) / viewport.zoom,
    }

    const next = zoomViewportAtPoint(viewport, point, 320, 0)

    expect(next.zoom).toBeLessThan(viewport.zoom)
    expect((point.x - next.x) / next.zoom).toBeCloseTo(graphPoint.x)
    expect((point.y - next.y) / next.zoom).toBeCloseTo(graphPoint.y)
  })

  it('clamps node wheel zoom to the workflow canvas limits', () => {
    expect(zoomViewportAtPoint({ x: 0, y: 0, zoom: 0.2 }, { x: 0, y: 0 }, 320, 0).zoom).toBe(0.2)
    expect(zoomViewportAtPoint({ x: 0, y: 0, zoom: 2 }, { x: 0, y: 0 }, -320, 0).zoom).toBe(2)
  })

  it('leaves wheel input inside an independent overlay instead of zooming the canvas', () => {
    const canvasNode = document.createElement('article')
    const connectionMenu = document.createElement('div')
    const menuCandidate = document.createElement('button')
    connectionMenu.classList.add('nowheel')
    connectionMenu.append(menuCandidate)

    expect(canvasOwnsWheelTarget(canvasNode)).toBe(true)
    expect(canvasOwnsWheelTarget(menuCandidate)).toBe(false)
  })
})
