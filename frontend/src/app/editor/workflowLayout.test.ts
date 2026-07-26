import { describe, expect, it } from 'vitest'
import {
  alignNodePositions,
  autoLayoutNodePositions,
  distributeNodePositions,
  snapNodePosition,
  type SizedWorkflowNode,
} from './workflowLayout'

const nodes: SizedWorkflowNode[] = [
  { id: 'a', position: { x: 0, y: 0 }, width: 100, height: 40 },
  { id: 'b', position: { x: 180, y: 80 }, width: 140, height: 60 },
  { id: 'c', position: { x: 420, y: 180 }, width: 80, height: 80 },
]

describe('workflow layout geometry', () => {
  it('aligns physical edges for differently sized nodes', () => {
    expect(alignNodePositions(nodes.slice(0, 2), 'right')).toEqual([
      { nodeId: 'a', position: { x: 220, y: 0 } },
      { nodeId: 'b', position: { x: 180, y: 80 } },
    ])
  })

  it('distributes equal edge-to-edge gaps while preserving endpoints', () => {
    const positions = distributeNodePositions(nodes, 'horizontal')
    expect(positions).toEqual([
      { nodeId: 'a', position: { x: 0, y: 0 } },
      { nodeId: 'b', position: { x: 190, y: 80 } },
      { nodeId: 'c', position: { x: 420, y: 180 } },
    ])
  })

  it('snaps matching centers and edges within the threshold', () => {
    const snapped = snapNodePosition(
      { id: 'dragged', position: { x: 104, y: 56 }, width: 100, height: 40 },
      [{ id: 'target', position: { x: 200, y: 100 }, width: 100, height: 40 }],
    )
    expect(snapped).toEqual({ position: { x: 100, y: 60 }, guideX: 200, guideY: 100 })
  })

  it('lays a connected chain in the requested direction without moving its centroid', async () => {
    const positions = await autoLayoutNodePositions(
      nodes,
      [
        {
          channel: 'exec',
          from: { nodeId: 'a', portId: 'out' },
          to: { nodeId: 'b', portId: 'in' },
        },
        {
          channel: 'exec',
          from: { nodeId: 'b', portId: 'out' },
          to: { nodeId: 'c', portId: 'in' },
        },
      ],
      'LR',
    )
    expect(positions[0]!.position.x).toBeLessThan(positions[1]!.position.x)
    expect(positions[1]!.position.x).toBeLessThan(positions[2]!.position.x)
    const oldCenterX = nodes.reduce((sum, node) => sum + node.position.x + node.width / 2, 0) / 3
    const newCenterX =
      positions.reduce((sum, item, index) => sum + item.position.x + nodes[index]!.width / 2, 0) / 3
    expect(newCenterX).toBeCloseTo(oldCenterX)
  })
})
