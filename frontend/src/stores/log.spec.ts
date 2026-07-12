import { setActivePinia, createPinia } from 'pinia'
import { describe, it, expect, beforeEach } from 'vitest'
import { useLogStore } from './log'

describe('log store node dump', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('upserts ×N by (nodeId,lineKey) in place', () => {
    const s = useLogStore()
    s.appendNodeDump([
      {
        nodeId: 'n_a',
        nodeKind: 'A',
        lineKey: 'out{v=1}',
        line: 'A(n_a) out{v=1}',
        count: 3,
        final: false,
      },
    ])
    s.appendNodeDump([
      {
        nodeId: 'n_a',
        nodeKind: 'A',
        lineKey: 'out{v=1}',
        line: 'A(n_a) out{v=1}',
        count: 7,
        final: false,
      },
    ])
    const dumps = s.lines.filter((l) => l.nodeId === 'n_a' && l.lineKey === 'out{v=1}')
    expect(dumps.length).toBe(1)
    expect(dumps[0].count).toBe(7)
  })

  it('final freezes row; next same key makes a new row', () => {
    const s = useLogStore()
    s.appendNodeDump([
      {
        nodeId: 'n_a',
        nodeKind: 'A',
        lineKey: 'out{v=1}',
        line: 'A(n_a) out{v=1}',
        count: 5,
        final: true,
      },
    ])
    s.appendNodeDump([
      {
        nodeId: 'n_a',
        nodeKind: 'A',
        lineKey: 'out{v=1}',
        line: 'A(n_a) out{v=1}',
        count: 1,
        final: false,
      },
    ])
    const dumps = s.lines.filter((l) => l.nodeId === 'n_a' && l.lineKey === 'out{v=1}')
    expect(dumps.length).toBe(2)
  })

  it('interleaved nodes each keep their own live row', () => {
    const s = useLogStore()
    s.appendNodeDump([
      { nodeId: 'n_a', nodeKind: 'A', lineKey: 'ka', line: 'A', count: 2, final: false },
    ])
    s.appendNodeDump([
      { nodeId: 'n_b', nodeKind: 'B', lineKey: 'kb', line: 'B', count: 2, final: false },
    ])
    s.appendNodeDump([
      { nodeId: 'n_a', nodeKind: 'A', lineKey: 'ka', line: 'A', count: 9, final: false },
    ])
    expect(s.lines.filter((l) => l.nodeId === 'n_a' && l.lineKey === 'ka')[0].count).toBe(9)
    expect(s.lines.filter((l) => l.nodeId === 'n_b' && l.lineKey === 'kb')[0].count).toBe(2)
  })
})
