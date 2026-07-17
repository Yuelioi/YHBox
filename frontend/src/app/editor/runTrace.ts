import type { RunView } from '@/app/transport/workflow'

export type NodeRunStatus = 'running' | 'succeeded' | 'failed' | 'cancelled' | 'routed'

export function nodeRunStatuses(
  run: RunView | null,
  graphId: string,
): ReadonlyMap<string, NodeRunStatus> {
  const statuses = new Map<string, NodeRunStatus>()
  if (!run) return statuses

  for (const entry of run.timeline) {
    if (entry.kind !== 'node-attempt' || !entry.nodeId || entry.graphPath.at(-1) !== graphId)
      continue
    const status = nodeRunStatus(entry.attemptOutcome)
    if (status) statuses.set(entry.nodeId, status)
  }

  if (run.failure?.nodeId && (!run.failure.graphId || run.failure.graphId === graphId)) {
    statuses.set(run.failure.nodeId, 'failed')
  }
  return statuses
}

function nodeRunStatus(value: string | undefined): NodeRunStatus | undefined {
  if (!value) return undefined
  if (value === 'started') return 'running'
  if (['succeeded', 'failed', 'cancelled', 'routed'].includes(value)) {
    return value as NodeRunStatus
  }
  return undefined
}
