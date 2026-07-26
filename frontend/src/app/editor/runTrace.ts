import type { RunView } from '@/app/transport/workflow'
import type { YottaWorkflowSource } from '../../../../contracts/workflow/current/workflow-source'

export type NodeRunStatus = 'running' | 'waiting' | 'succeeded' | 'failed' | 'cancelled' | 'routed'

export interface ActiveRunAttempt {
  graphPath: string[]
  nodeId: string
  attempt: number
  startedAt: string
  statusCode?: string
  statusCategory?: string
  counters: Record<string, number>
}

export function nodeRunStatuses(
  run: RunView | null,
  graphId: string,
): ReadonlyMap<string, NodeRunStatus> {
  const statuses = new Map<string, NodeRunStatus>()
  if (!run) return statuses

  const active = new Set<string>()
  for (const entry of run.timeline) {
    if (!entry.nodeId || entry.graphPath.at(-1) !== graphId) continue
    const key = attemptKey(entry.graphPath, entry.nodeId, entry.attempt)
    if (entry.kind === 'node-attempt') {
      const status = nodeRunStatus(entry.attemptOutcome)
      if (status) statuses.set(entry.nodeId, status)
      if (entry.attemptOutcome === 'started') active.add(key)
      else if (status) active.delete(key)
    } else if (
      entry.kind === 'node-status' &&
      entry.statusCategory === 'waiting' &&
      active.has(key)
    ) {
      statuses.set(entry.nodeId, 'waiting')
    }
  }

  if (run.failure?.nodeId && (!run.failure.graphId || run.failure.graphId === graphId)) {
    statuses.set(run.failure.nodeId, 'failed')
  }
  return statuses
}

export function activeRunAttempt(run: RunView): ActiveRunAttempt | null {
  const active = new Map<string, ActiveRunAttempt & { sequence: number }>()
  for (const entry of run.timeline) {
    if (!entry.nodeId || entry.attempt < 1) continue
    const key = attemptKey(entry.graphPath, entry.nodeId, entry.attempt)
    if (entry.kind === 'node-attempt' && entry.attemptOutcome === 'started') {
      active.set(key, {
        graphPath: [...entry.graphPath],
        nodeId: entry.nodeId,
        attempt: entry.attempt,
        startedAt: entry.occurredAt,
        counters: {},
        sequence: entry.sequence,
      })
      continue
    }
    if (entry.kind === 'node-attempt' && nodeRunStatus(entry.attemptOutcome)) {
      active.delete(key)
      continue
    }
    const attempt = active.get(key)
    if (attempt && entry.kind === 'node-status') {
      attempt.statusCode = entry.statusCode
      attempt.statusCategory = entry.statusCategory
      attempt.counters = Object.fromEntries(
        Object.entries(entry.summary.counters).filter(
          (item): item is [string, number] => typeof item[1] === 'number',
        ),
      )
    }
  }
  const latest = [...active.values()].sort((left, right) => right.sequence - left.sequence)[0]
  if (!latest) return null
  const { sequence: _, ...attempt } = latest
  return attempt
}

export function unhandledExecRouteKeys(source: YottaWorkflowSource): ReadonlySet<string> {
  const result = new Set<string>()
  for (const graph of source.graphs) {
    const connected = new Set(
      graph.edges
        .filter((edge) => edge.channel === 'exec')
        .map((edge) => `${edge.from.nodeId}\u0000${edge.from.portId}`),
    )
    for (const node of graph.nodes) {
      if (!connected.has(`${node.id}\u0000timeout`)) {
        result.add(runRouteKey([graph.id], node.id, 'timeout'))
      }
    }
  }
  return result
}

export function runRouteKey(graphPath: string[], nodeId: string, portId: string): string {
  return `${graphPath.at(-1) ?? ''}\u0000${nodeId}\u0000${portId}`
}

export function statusRoutePort(statusCode: string | undefined): string | undefined {
  return statusCode === 'automation.template.timeout' ? 'timeout' : undefined
}

function attemptKey(graphPath: string[], nodeId: string, attempt: number): string {
  return `${graphPath.join('/')}\u0000${nodeId}\u0000${attempt}`
}

function nodeRunStatus(value: string | undefined): NodeRunStatus | undefined {
  if (!value) return undefined
  if (value === 'started') return 'running'
  if (['succeeded', 'failed', 'cancelled', 'routed'].includes(value)) {
    return value as NodeRunStatus
  }
  return undefined
}
