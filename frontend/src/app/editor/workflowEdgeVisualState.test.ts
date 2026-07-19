import { describe, expect, it } from 'vitest'
import type { Edge } from './EditorSession'
import { workflowEdgeVisualState } from './workflowEdgeVisualState'

const edge = {
  channel: 'exec',
  from: { nodeId: 'first', portId: 'done' },
  to: { nodeId: 'second', portId: 'in' },
} as Edge

describe('workflow edge visual state', () => {
  it('keeps idle signal edges quiet', () => {
    const state = workflowEdgeVisualState(edge, new Map())

    expect(state.traced).toBe(false)
    expect(state.animated).toBe(false)
  })

  it('highlights only the observed execution path', () => {
    const state = workflowEdgeVisualState(
      edge,
      new Map([
        ['first', 'succeeded'],
        ['second', 'running'],
      ]),
    )

    expect(state.traced).toBe(true)
    expect(state.animated).toBe(true)
    expect(state.stroke).toBe('var(--ui-primary)')
  })

  it('shows a completed observed path without perpetual animation', () => {
    const state = workflowEdgeVisualState(
      edge,
      new Map([
        ['first', 'succeeded'],
        ['second', 'succeeded'],
      ]),
    )

    expect(state.traced).toBe(true)
    expect(state.animated).toBe(false)
    expect(state.stroke).toBe('var(--ui-success)')
  })
})
