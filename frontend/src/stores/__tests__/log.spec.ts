import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useLogStore } from '../log'

describe('useLogStore', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('maps process and Workflow diagnostics without legacy node-log variants', () => {
    const store = useLogStore()
    store.appendBatch(1, [
      { time: '', level: 'info', source: 'SYS', message: 'ready' },
      {
        time: '',
        level: 'warn',
        source: 'WF',
        message: 'retry',
        graphId: 'g1',
        nodeId: 'n1',
        invocationId: 'i1',
        attempt: 2,
      },
    ])

    expect(store.lines.map((line) => line.source)).toEqual(['SYS', 'WF'])
    expect(store.lines[1]).toMatchObject({
      graphId: 'g1',
      nodeId: 'n1',
      invocationId: 'i1',
      attempt: 2,
    })
    expect(store.received).toBe(2)
  })

  it('reports transport drops and sequence gaps', () => {
    const store = useLogStore()
    store.appendBatch(3, [{ time: '', level: 'info', source: 'SYS', message: 'a' }], 4)
    store.appendBatch(5, [{ time: '', level: 'info', source: 'SYS', message: 'b' }])

    expect(store.dropDetected).toBe(true)
    expect(store.dropped).toBe(4)
    expect(store.lines.some((line) => line.message.includes('sequence gap'))).toBe(true)
  })

  it('bounds the local ring and clear resets transport state', () => {
    const store = useLogStore()
    store.appendBatch(
      1,
      Array.from({ length: 1200 }, (_, index) => ({
        time: '',
        level: 'info',
        source: 'SYS' as const,
        message: `m${index}`,
      })),
    )
    expect(store.lines).toHaveLength(1000)
    expect(store.lines[0].message).toBe('m200')
    expect(store.dropped).toBe(200)

    store.clear()
    expect(store.lines).toHaveLength(0)
    expect(store.lastSeq).toBe(0)
    expect(store.dropped).toBe(0)
  })
})
