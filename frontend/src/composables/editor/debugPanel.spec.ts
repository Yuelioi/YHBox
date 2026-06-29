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

  it('summarizes output and variable snapshots compactly', () => {
    const s = summarizeDebugSession({
      sessionId: 's1',
      status: 'paused',
      lastNodeId: 'expr',
      lastNodeKind: 'Expr',
      lastOutput: { z: 3, a: 'ready', obj: { ok: true }, b: false },
      vars: { hp: 100, name: 'player' },
    })

    expect(s.lastOutputPreview).toBe('a="ready", b=false, obj={"ok":true} +1')
    expect(s.varsPreview).toBe('hp=100, name="player"')
  })

  it('summarizes queued debug tokens compactly', () => {
    const s = summarizeDebugSession({
      sessionId: 's1',
      status: 'paused',
      queue: [
        { nodeId: 'click-primary-button', nodeKind: 'ClickAt', inPin: 'In' },
        { nodeId: 'wait', nodeKind: 'Sleep', inPin: 'In' },
        { nodeId: 'log', nodeKind: 'Log', inPin: 'In' },
        { nodeId: 'stop', nodeKind: 'Stop', inPin: 'In' },
      ],
    })

    expect(s.queuePreview).toBe('ClickAt:click-primary-but....In -> Sleep:wait.In -> Log:log.In +1')
  })

  it('hides after a stopped debug session is cleared', () => {
    const s = summarizeDebugSession({
      sessionId: 's1',
      status: 'stopped',
    })

    expect(s.visible).toBe(false)
    expect(s.active).toBe(false)
  })
})
