import type { Edge } from './EditorSession'
import type { NodeRunStatus } from './runTrace'

export interface WorkflowEdgeVisualState {
  traced: boolean
  animated: boolean
  stroke: string
  strokeWidth: number
}

export function workflowEdgeVisualState(
  edge: Edge,
  statuses: ReadonlyMap<string, NodeRunStatus>,
): WorkflowEdgeVisualState {
  const from = statuses.get(edge.from.nodeId)
  const to = statuses.get(edge.to.nodeId)
  const traced = edge.channel !== 'data' && Boolean(from && to)
  const animated = traced && to === 'running'
  return {
    traced,
    animated,
    stroke: traced ? traceStroke(edge, to) : baseStroke(edge),
    strokeWidth: traced ? 3 : 2,
  }
}

function traceStroke(edge: Edge, target?: NodeRunStatus): string {
  if (edge.channel === 'error' || target === 'failed') return 'var(--ui-error)'
  if (target === 'cancelled' || target === 'routed') return 'var(--ui-warning)'
  if (target === 'running') return 'var(--ui-primary)'
  return 'var(--ui-success)'
}

function baseStroke(edge: Edge): string {
  if (edge.channel === 'error') return 'var(--ui-error)'
  if (edge.channel === 'data') return 'var(--ui-primary)'
  return 'var(--ui-text-dimmed)'
}
