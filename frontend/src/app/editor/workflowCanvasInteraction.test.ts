import { describe, expect, it } from 'vitest'
import { mergeMarqueeSelection, WORKFLOW_CANVAS_INTERACTION } from './workflowCanvasInteraction'

describe('workflow canvas interaction contract', () => {
  it('reserves blank left drag for marquee and keeps space or middle drag for panning', () => {
    expect(WORKFLOW_CANVAS_INTERACTION.selectionKeyCode).toBe(true)
    expect(WORKFLOW_CANVAS_INTERACTION.panActivationKeyCode).toBe('Space')
    expect(WORKFLOW_CANVAS_INTERACTION.panOnDrag).toEqual([0, 1])
    expect(WORKFLOW_CANVAS_INTERACTION.multiSelectionKeyCode).toBe('Control')
  })

  it('adds a shift marquee to the selection captured before Vue Flow clears it', () => {
    expect(mergeMarqueeSelection(new Set(['first']), new Set(['second']))).toEqual(
      new Set(['first', 'second']),
    )
  })

  it('uses the marquee result directly when no additive selection was captured', () => {
    expect(mergeMarqueeSelection(new Set(), new Set(['second']))).toEqual(new Set(['second']))
  })
})
