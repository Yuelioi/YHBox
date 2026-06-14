import { describe, it, expect } from 'vitest'
import { resolveInspectorMode } from './inspectorMode'

describe('resolveInspectorMode', () => {
  it('selected node always wins (even in subgraph)', () => {
    expect(resolveInspectorMode({ hasSelectedNode: true, inSubgraph: false })).toBe('node')
    expect(resolveInspectorMode({ hasSelectedNode: true, inSubgraph: true })).toBe('node')
  })
  it('subgraph empty selection keeps subgraph panel', () => {
    expect(resolveInspectorMode({ hasSelectedNode: false, inSubgraph: true })).toBe('subgraph')
  })
  it('root empty selection collapses', () => {
    expect(resolveInspectorMode({ hasSelectedNode: false, inSubgraph: false })).toBe('collapsed')
  })
})
