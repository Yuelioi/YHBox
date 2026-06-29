import { setActivePinia, createPinia } from 'pinia'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { Events } from '@wailsio/runtime'

vi.mock('@/lib/invoke', () => ({
  toastError: vi.fn(),
  errorMessage: (e: unknown) => String(e ?? ''),
}))
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k } } }))

import { useExecutionStore } from './execution'

describe('execution store debug state', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('mirrors debug:state and exposes the next queued node', async () => {
    const s = useExecutionStore()
    await Events.Emit('debug:state', {
      sessionId: 'dbg1',
      containerId: 'c1',
      status: 'paused',
      mode: 'entry',
      currentNodeId: 'start',
      currentNodeKind: 'Start',
      queue: [{ nodeId: 'click1', nodeKind: 'Click', inPin: 'In' }],
    })

    expect(s.debugActive).toBe(true)
    expect(s.debugSessionID).toBe('dbg1')
    expect(s.debugCurrentNodeID).toBe('start')
    expect(s.debugNextNodeID).toBe('click1')
  })

  it('normalizes Go-shaped debug:state events emitted by Wails', async () => {
    const s = useExecutionStore()
    await Events.Emit('debug:state', {
      SessionID: 'dbg1',
      ContainerID: 'c1',
      Status: 'paused',
      Mode: 'entry',
      CurrentNodeID: 'androidtarget',
      CurrentNodeKind: 'AndroidTarget',
      LastNodeID: 'androidtarget',
      LastNodeKind: 'AndroidTarget',
      LastExit: 'Done',
      Queue: [{ NodeID: 'androidstartapp', NodeKind: 'AndroidStartApp', InPin: 'In' }],
    })

    expect(s.debugActive).toBe(true)
    expect(s.debugStatus).toBe('paused')
    expect(s.debugCurrentNodeID).toBe('androidtarget')
    expect(s.debugLastNodeID).toBe('androidtarget')
    expect(s.debugLastExit).toBe('Done')
    expect(s.debugNextNodeID).toBe('androidstartapp')
    expect(s.debugCanStep).toBe(true)
  })

  it('accepts node-enter events while a debug session is running', async () => {
    const s = useExecutionStore()
    await Events.Emit('debug:state', {
      sessionId: 'dbg1',
      containerId: 'c1',
      status: 'stepping',
      mode: 'entry',
      runningNodeId: 'sleep1',
      runningNodeKind: 'Sleep',
    })
    await Events.Emit('container:node-enter', { nodeId: 'click1', nodeKind: 'Click' })

    expect(s.debugRunningNodeID).toBe('click1')
    expect(s.debugRunningNodeKind).toBe('Click')
  })

  it('keeps the failed node after a terminal debug failure', async () => {
    const s = useExecutionStore()
    s.applyDebugState({
      sessionId: 'dbg1',
      containerId: 'c1',
      status: 'failed',
      mode: 'entry',
      currentNodeId: 'bad1',
      currentNodeKind: 'Expr',
      error: { message: 'boom' },
    })

    expect(s.debugActive).toBe(false)
    expect(s.debugFailedNodeID).toBe('bad1')
    expect(s.debugError?.message).toBe('boom')
  })

  it('stores variable snapshots and last output from debug states', () => {
    const s = useExecutionStore()
    s.applyDebugState({
      sessionId: 'dbg1',
      containerId: 'c1',
      status: 'paused',
      mode: 'entry',
      lastNodeId: 'expr1',
      lastNodeKind: 'Expr',
      lastExit: 'Done',
      lastOutput: { Value: 12, Text: 'ok' },
      vars: { hp: 100, state: 'ready' },
    })

    expect(s.debugLastOutput).toEqual({ Value: 12, Text: 'ok' })
    expect(s.debugVars).toEqual({ hp: 100, state: 'ready' })
  })

  it('clears stale debug UI state when the backend session is gone', () => {
    const s = useExecutionStore()
    s.applyDebugState({
      sessionId: 'dbg1',
      containerId: 'c1',
      status: 'paused',
      mode: 'from_node',
      startNodeId: 'click1',
      currentNodeId: 'click1',
      currentNodeKind: 'ClickAt',
      lastNodeId: 'target1',
      lastNodeKind: 'Win32WindowTarget',
      lastExit: 'Done',
      queue: [{ nodeId: 'click1', nodeKind: 'ClickAt', inPin: 'In' }],
      warnings: [{ code: 'debug_skips_upstream_context', message: 'skipped' }],
      lastOutput: { Done: true },
      vars: { hp: 100 },
    })

    s.clearDebugState()

    expect(s.debugActive).toBe(false)
    expect(s.debugSessionID).toBe('')
    expect(s.debugCurrentNodeID).toBe('')
    expect(s.debugLastNodeID).toBe('')
    expect(s.debugQueue).toEqual([])
    expect(s.debugWarnings).toEqual([])
    expect(s.debugLastOutput).toEqual({})
    expect(s.debugVars).toEqual({})
  })
})
