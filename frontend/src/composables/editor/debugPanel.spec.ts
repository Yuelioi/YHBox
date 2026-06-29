import { describe, expect, it } from 'vitest'
import { summarizeDebugSession } from './debugPanel'

describe('debugPanel summary', () => {
  it('focuses the next queued node while paused', () => {
    const s = summarizeDebugSession({
      sessionId: 's1',
      status: 'paused',
      currentNodeId: 'n1',
      currentNodeKind: 'Click',
      queue: [{ nodeId: 'n1', nodeKind: 'Click', inPin: 'In' }],
    })

    expect(s.visible).toBe(true)
    expect(s.active).toBe(true)
    expect(s.tone).toBe('primary')
    expect(s.focusNodeID).toBe('n1')
    expect(s.focusRoleKey).toBe('editor.debug_panel.focus_next')
    expect(s.queueCount).toBe(1)
  })

  it('focuses the running node during continue', () => {
    const s = summarizeDebugSession({
      sessionId: 's1',
      status: 'running',
      runningNodeId: 'wait',
      runningNodeKind: 'Sleep',
      currentNodeId: 'next',
      currentNodeKind: 'Click',
    })

    expect(s.tone).toBe('warning')
    expect(s.focusNodeID).toBe('wait')
    expect(s.focusRoleKey).toBe('editor.debug_panel.focus_running')
  })

  it('keeps failed sessions visible with error tone', () => {
    const s = summarizeDebugSession({
      sessionId: 's1',
      status: 'failed',
      currentNodeId: 'bad',
      currentNodeKind: 'Expr',
      error: { message: 'boom' },
    })

    expect(s.visible).toBe(true)
    expect(s.active).toBe(false)
    expect(s.tone).toBe('error')
    expect(s.focusNodeID).toBe('bad')
    expect(s.error?.message).toBe('boom')
  })
})
