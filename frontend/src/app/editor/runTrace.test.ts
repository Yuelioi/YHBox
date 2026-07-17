import { describe, expect, it } from 'vitest'
import type { RunView } from '@/app/transport/workflow'
import { nodeRunStatuses } from './runTrace'

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
