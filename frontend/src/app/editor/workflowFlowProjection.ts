import type { Node as FlowNode } from '@vue-flow/core'
import { shallowReactive } from 'vue'
import type { Node, NodeProjection } from './EditorSession'

export interface WorkflowNodeData {
  node: Node
  projection: NodeProjection
}

export const WORKFLOW_NODE_DRAG_HANDLE = '.workflow-node-drag-handle'

export interface WorkflowNodeGestureState {
  positions: ReadonlyMap<string, Node['position']>
  track: (nodeId: string, position: Node['position']) => void
  clear: (nodeId: string) => void
}

export function createWorkflowNodeGestureState(): WorkflowNodeGestureState {
  const positions = shallowReactive(new Map<string, Node['position']>())
  return {
    positions,
    track(nodeId, position) {
      positions.set(nodeId, { ...position })
    },
    clear(nodeId) {
      positions.delete(nodeId)
    },
  }
}

export function projectWorkflowFlowNodes(
  nodes: readonly Node[],
  projectionFor: (node: Node) => NodeProjection | undefined,
  livePositions: ReadonlyMap<string, Node['position']>,
): FlowNode<WorkflowNodeData, Record<string, never>, 'workflow'>[] {
  return nodes.flatMap((node) => {
    const projection = projectionFor(node)
    if (!projection) return []
    return [
      {
        id: node.id,
        type: 'workflow',
        position: livePositions.get(node.id) ?? node.position,
        dragHandle: WORKFLOW_NODE_DRAG_HANDLE,
        data: { node, projection },
      },
    ]
  })
}
