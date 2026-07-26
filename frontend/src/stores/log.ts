import { defineStore } from 'pinia'
import { ref, shallowRef } from 'vue'
import type { BackendLogEntry } from '@/lib/backend'
import type { LogLine } from '@/lib/logFormat'

const RING_CAP = 1000

function nowIso() {
  return new Date().toISOString()
}

export const useLogStore = defineStore('log', () => {
  const lines = shallowRef<LogLine[]>([])
  const lastSeq = ref(0)
  const dropDetected = ref(false)
  const received = ref(0)
  const dropped = ref(0)
  let nextID = 1

  function appendBatch(seq: number, entries: BackendLogEntry[], transportDropped = 0) {
    let next = lines.value.slice()
    if (lastSeq.value !== 0 && seq !== lastSeq.value + 1) {
      dropDetected.value = true
      next.push({
        id: nextID++,
        time: nowIso(),
        level: 'warn',
        message: `[log:batch] sequence gap: expected ${lastSeq.value + 1}, got ${seq}`,
        source: 'SYS',
      })
    }
    lastSeq.value = seq
    if (transportDropped > 0) {
      dropDetected.value = true
      dropped.value += transportDropped
    }

    next.push(
      ...entries.map((entry) => ({
        id: nextID++,
        time: entry.time || nowIso(),
        level: entry.level || 'info',
        message: entry.message,
        source: entry.source === 'WF' ? ('WF' as const) : ('SYS' as const),
        tag: entry.tag,
        fields: entry.fields,
        graphId: entry.graphId,
        nodeId: entry.nodeId,
        invocationId: entry.invocationId,
        attempt: entry.attempt,
      })),
    )
    received.value += entries.length

    if (next.length > RING_CAP) {
      const overflow = next.length - RING_CAP
      dropped.value += overflow
      next = next.slice(overflow)
    }
    lines.value = next
  }

  function clear() {
    lines.value = []
    lastSeq.value = 0
    dropDetected.value = false
    received.value = 0
    dropped.value = 0
  }

  return { lines, lastSeq, dropDetected, received, dropped, appendBatch, clear }
})
