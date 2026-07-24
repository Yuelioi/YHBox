import { describe, expect, it } from 'vitest'
import type { RunView } from '@/app/transport/workflow'
import type { YottaWorkflowSource } from '../../../../contracts/workflow/current/workflow-source'
import { activeRunAttempt, nodeRunStatuses, runRouteKey, unhandledExecRouteKeys } from './runTrace'

describe('node run statuses', () => {
  it('projects the latest node-attempt facts for the visible graph', () => {
    const run = {
      status: 'RUNNING',
      timeline: [
        attempt(1, 'main', 'root', 'started'),
        attempt(2, 'child', 'hidden', 'failed'),
        attempt(3, 'main', 'root', 'succeeded'),
        attempt(4, 'main', 'next', 'started'),
      ],
    } as RunView

    expect(Object.fromEntries(nodeRunStatuses(run, 'main'))).toEqual({
      root: 'succeeded',
      next: 'running',
    })
  })

  it('uses the terminal run failure when the journal has no matching terminal attempt', () => {
    const run = {
      status: 'FAILED',
      failure: { graphId: 'main', nodeId: 'broken' },
      timeline: [attempt(1, 'main', 'broken', 'started')],
    } as RunView

    expect(nodeRunStatuses(run, 'main').get('broken')).toBe('failed')
  })

  it('projects the active attempt with its latest waiting fact and timeout budget', () => {
    const run = {
      status: 'RUNNING',
      timeline: [
        attempt(1, 'main', 'click', 'started'),
        {
          sequence: 2,
          kind: 'node-status',
          graphPath: ['main'],
          nodeId: 'click',
          attempt: 1,
          statusCode: 'automation.template.waiting',
          statusCategory: 'waiting',
          occurredAt: '2026-07-17T00:00:01Z',
          summary: { code: 'automation.template.waiting', counters: { timeout_ms: 5000 } },
        },
      ],
    } as RunView

    expect(activeRunAttempt(run)).toEqual({
      graphPath: ['main'],
      nodeId: 'click',
      attempt: 1,
      startedAt: '2026-07-17T00:00:00Z',
      statusCode: 'automation.template.waiting',
      statusCategory: 'waiting',
      counters: { timeout_ms: 5000 },
    })
    expect(nodeRunStatuses(run, 'main').get('click')).toBe('waiting')
  })

  it('identifies an unconnected timeout route without guessing from run status', () => {
    const source = {
      graphs: [
        {
          id: 'main',
          nodes: [{ id: 'click' }, { id: 'next' }],
          edges: [
            {
              channel: 'exec',
              from: { nodeId: 'click', portId: 'completed' },
              to: { nodeId: 'next', portId: 'in' },
            },
          ],
        },
      ],
    } as unknown as YottaWorkflowSource

    expect(unhandledExecRouteKeys(source).has(runRouteKey(['main'], 'click', 'timeout'))).toBe(true)
  })
})

function attempt(sequence: number, graphId: string, nodeId: string, attemptOutcome: string) {
  return {
    sequence,
    kind: 'node-attempt',
    graphPath: [graphId],
    nodeId,
    attempt: 1,
    attemptOutcome,
    occurredAt: '2026-07-17T00:00:00Z',
    summary: { code: 'test', counters: {} },
  }
}
