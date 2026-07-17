import type { Edge } from './EditorSession'

export type AlignMode =
  | 'left'
  | 'right'
  | 'top'
  | 'bottom'
  | 'horizontal-center'
  | 'vertical-center'
export type DistributeMode = 'horizontal' | 'vertical'

export interface SizedWorkflowNode {
  id: string
  position: { x: number; y: number }
  width: number
  height: number
}

export interface SnappedPosition {
  position: { x: number; y: number }
  guideX?: number
  guideY?: number
}

export function alignNodePositions(
  nodes: SizedWorkflowNode[],
  mode: AlignMode,
): Array<{ nodeId: string; position: { x: number; y: number } }> {
  if (nodes.length < 2) return []
  const left = Math.min(...nodes.map((node) => node.position.x))
  const right = Math.max(...nodes.map((node) => node.position.x + node.width))
  const top = Math.min(...nodes.map((node) => node.position.y))
  const bottom = Math.max(...nodes.map((node) => node.position.y + node.height))
  const centerX =
    nodes.reduce((sum, node) => sum + node.position.x + node.width / 2, 0) / nodes.length
  const centerY =
    nodes.reduce((sum, node) => sum + node.position.y + node.height / 2, 0) / nodes.length
  return nodes.map((node) => ({
    nodeId: node.id,
    position: {
      x:
        mode === 'left'
          ? left
          : mode === 'right'
            ? right - node.width
            : mode === 'vertical-center'
              ? centerX - node.width / 2
              : node.position.x,
      y:
        mode === 'top'
          ? top
          : mode === 'bottom'
            ? bottom - node.height
            : mode === 'horizontal-center'
              ? centerY - node.height / 2
              : node.position.y,
    },
  }))
}

export function distributeNodePositions(
  nodes: SizedWorkflowNode[],
  mode: DistributeMode,
): Array<{ nodeId: string; position: { x: number; y: number } }> {
  if (nodes.length < 3) return []
  const horizontal = mode === 'horizontal'
  const sorted = [...nodes].sort((left, right) =>
    horizontal ? left.position.x - right.position.x : left.position.y - right.position.y,
  )
  const first = sorted[0]!
  const last = sorted.at(-1)!
  const span = horizontal
    ? last.position.x + last.width - first.position.x
    : last.position.y + last.height - first.position.y
  const occupied = sorted.reduce((sum, node) => sum + (horizontal ? node.width : node.height), 0)
  const gap = (span - occupied) / (sorted.length - 1)
  let cursor = horizontal ? first.position.x : first.position.y
  return sorted.map((node) => {
    const position = horizontal
      ? { x: cursor, y: node.position.y }
      : { x: node.position.x, y: cursor }
    cursor += (horizontal ? node.width : node.height) + gap
    return { nodeId: node.id, position }
  })
}

export function snapNodePosition(
  dragged: SizedWorkflowNode,
  others: SizedWorkflowNode[],
  threshold = 6,
): SnappedPosition {
  const x = bestSnap(
    [
      dragged.position.x,
      dragged.position.x + dragged.width / 2,
      dragged.position.x + dragged.width,
    ],
    others.flatMap((node) => [
      node.position.x,
      node.position.x + node.width / 2,
      node.position.x + node.width,
    ]),
    threshold,
  )
  const y = bestSnap(
    [
      dragged.position.y,
      dragged.position.y + dragged.height / 2,
      dragged.position.y + dragged.height,
    ],
    others.flatMap((node) => [
      node.position.y,
      node.position.y + node.height / 2,
      node.position.y + node.height,
    ]),
    threshold,
  )
  return {
    position: {
      x: dragged.position.x + (x?.delta ?? 0),
      y: dragged.position.y + (y?.delta ?? 0),
    },
    guideX: x?.target,
    guideY: y?.target,
  }
}

export async function autoLayoutNodePositions(
  nodes: SizedWorkflowNode[],
  edges: Edge[],
  direction: 'LR' | 'TB',
): Promise<Array<{ nodeId: string; position: { x: number; y: number } }>> {
  if (!nodes.length) return []
  const { default: ELK } = await import('elkjs/lib/elk.bundled.js')
  const elk = new ELK()
  const result = await elk.layout({
    id: 'workflow',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': direction === 'LR' ? 'RIGHT' : 'DOWN',
      'elk.edgeRouting': 'ORTHOGONAL',
      'elk.layered.spacing.nodeNodeBetweenLayers': '80',
      'elk.spacing.nodeNode': '40',
    },
    children: nodes.map((node) => ({ id: node.id, width: node.width, height: node.height })),
    edges: edges.map((edge, index) => ({
      id: `edge-${index}`,
      sources: [edge.from.nodeId],
      targets: [edge.to.nodeId],
    })),
  })
  const oldCenter = centroid(
    nodes.map((node) => ({ ...node.position, width: node.width, height: node.height })),
  )
  const laidOut = (result.children ?? []).map((node) => ({
    id: node.id,
    x: node.x ?? 0,
    y: node.y ?? 0,
    width: node.width ?? 230,
    height: node.height ?? 90,
  }))
  const newCenter = centroid(laidOut)
  return laidOut.map((node) => ({
    nodeId: node.id,
    position: { x: node.x + oldCenter.x - newCenter.x, y: node.y + oldCenter.y - newCenter.y },
  }))
}

function bestSnap(
  draggedAnchors: number[],
  targets: number[],
  threshold: number,
): { delta: number; target: number } | undefined {
  let best: { delta: number; target: number } | undefined
  for (const dragged of draggedAnchors) {
    for (const target of targets) {
      const delta = target - dragged
      if (Math.abs(delta) > threshold || (best && Math.abs(best.delta) <= Math.abs(delta))) continue
      best = { delta, target }
    }
  }
  return best
}

function centroid(nodes: Array<{ x: number; y: number; width: number; height: number }>): {
  x: number
  y: number
} {
  return {
    x: nodes.reduce((sum, node) => sum + node.x + node.width / 2, 0) / nodes.length,
    y: nodes.reduce((sum, node) => sum + node.y + node.height / 2, 0) / nodes.length,
  }
}
