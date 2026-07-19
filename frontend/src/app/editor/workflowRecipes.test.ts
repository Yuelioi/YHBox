import { describe, expect, it } from 'vitest'
import type { NodeProjection } from '../../../../contracts/node/3.1/authoring-projection'
import { buildWorkflowRecipe } from './workflowRecipes'

function projection(typeID: string): NodeProjection {
  return {
    nodeRef: { nodeTypeId: typeID, semanticDigest: `sha256:${typeID}`, version: '1.0.0' },
  } as NodeProjection
}

describe('workflow visual recipes', () => {
  it('builds color analysis from ordinary capture, analysis, comparison and branch nodes', () => {
    const recipe = buildWorkflowRecipe('analyze-color', (typeID) => projection(typeID))
    expect(recipe.nodes.map((node) => node.id)).toEqual(['capture', 'analyze', 'compare', 'branch'])
    expect(recipe.edges).toContainEqual({
      channel: 'data',
      from: { nodeId: 'capture', portId: 'image' },
      to: { nodeId: 'analyze', portId: 'image' },
    })
    expect(recipe.edges).toContainEqual({
      channel: 'exec',
      from: { nodeId: 'capture', portId: 'completed' },
      to: { nodeId: 'branch', portId: 'in' },
    })
  })

  it('builds blob location with an explicit list length decision chain', () => {
    const recipe = buildWorkflowRecipe('find-color-blobs', (typeID) => projection(typeID))
    expect(recipe.nodes.map((node) => node.id)).toEqual([
      'capture',
      'find',
      'length',
      'compare',
      'branch',
    ])
    expect(recipe.edges).toHaveLength(5)
  })
})
