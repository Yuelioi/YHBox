import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useLogStore } from '../log'

describe('useLogStore', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('appendSystem adds SYS-tagged lines', () => {
    const s = useLogStore()
    s.appendSystem(1, [
      '{"time":"2026-05-28T10:00:00Z","level":"info","tag":"SYSTEM","message":"hello"}',
    ])
    expect(s.lines).toHaveLength(1)
    expect(s.lines[0].source).toBe('SYS')
  })

  it('appendContainerLog adds CTR-tagged lines', () => {
    const s = useLogStore()
    s.appendContainerLog({ level: 'info', message: 'container log' })
    expect(s.lines).toHaveLength(1)
    expect(s.lines[0].source).toBe('CTR')
    expect(s.lines[0].message).toBe('container log')
  })

  it('appendNodeEnter batch unfolds entries', () => {
    const s = useLogStore()
    s.appendNodeEnter([
      { nodeId: 'n1', nodeKind: 'Sleep', count: 3 },
      { nodeId: 'n2', nodeKind: 'Click', count: 1 },
    ])
    expect(s.lines).toHaveLength(2)
    expect(s.lines[0].message).toContain('Sleep')
    expect(s.lines[0].message).toContain('× 3')
    expect(s.lines[1].message).toContain('Click')
  })

  it('ring buffer caps at 500', () => {
    const s = useLogStore()
    for (let i = 0; i < 600; i++) {
      s.appendContainerLog({ level: 'info', message: `m${i}` })
    }
    expect(s.lines).toHaveLength(500)
    expect(s.lines[0].message).toBe('m100')
  })

  it('clear empties all', () => {
    const s = useLogStore()
    s.appendContainerLog({ level: 'info', message: 'x' })
    s.appendActionTrace({
      containerId: 'c1',
      action: 'click',
      source: { nodeId: 'n1', nodeKind: 'ClickAt', inPin: 'In' },
      target: { id: 'win32:42' },
      backend: 'sendinput',
      status: 'success',
      durationMs: 12,
    })
    s.clear()
    expect(s.lines).toHaveLength(0)
    expect(s.actionTraces).toHaveLength(0)
  })

  it('appendActionTrace keeps structured cache and adds compact log line', () => {
    const s = useLogStore()
    s.appendActionTrace({
      containerId: 'c1',
      action: 'click',
      source: { nodeId: 'n1', nodeKind: 'ClickAt', inPin: 'In' },
      target: { id: 'win32:42' },
      backend: 'sendinput',
      status: 'success',
      durationMs: 12,
    })
    expect(s.actionTraces).toHaveLength(1)
    expect(s.actionTraces[0].action).toBe('click')
    expect(s.lines).toHaveLength(1)
    expect(s.lines[0].level).toBe('action')
    expect(s.lines[0].source).toBe('CTR')
    expect(s.lines[0].message).toContain('ClickAt(n1)')
    expect(s.lines[0].message).toContain('click')
    expect(s.lines[0].message).toContain('12ms')
  })
})
