import { defineStore } from 'pinia'
import { ref } from 'vue'
import { parseLine, type LogLine } from '@/lib/logFormat'

const RING_CAP = 500

interface NodeEnterEntry {
  nodeId: string
  nodeKind: string
  count: number
}

interface ContainerLogPayload {
  level: string
  message: string
}

interface NodeDumpEntry {
  nodeId: string
  nodeKind: string
  lineKey: string
  line: string
  count: number
  final: boolean
}

function pushBounded(lines: LogLine[], line: LogLine) {
  lines.push(line)
  if (lines.length > RING_CAP) {
    lines.splice(0, lines.length - RING_CAP)
  }
}

function nowIso() {
  return new Date().toISOString()
}

export const useLogStore = defineStore('log', () => {
  const lines = ref<LogLine[]>([])
  const lastSeq = ref(0)
  const dropDetected = ref(false)

  function appendSystem(seq: number, raw: string[]) {
    if (lastSeq.value !== 0 && seq !== lastSeq.value + 1) {
      dropDetected.value = true
      pushBounded(lines.value, {
        time: nowIso(),
        level: 'warn',
        message: `[log:lines] sequence gap: expected ${lastSeq.value + 1}, got ${seq}`,
        source: 'SYS',
      })
    }
    lastSeq.value = seq
    for (const r of raw) {
      pushBounded(lines.value, parseLine(r))
    }
  }

  function appendContainerLog(p: ContainerLogPayload) {
    pushBounded(lines.value, {
      time: nowIso(),
      level: p.level || 'info',
      message: p.message,
      source: 'CTR',
    })
  }

  function appendNodeEnter(entries: NodeEnterEntry[]) {
    for (const e of entries) {
      const suffix = e.count > 1 ? ` × ${e.count}` : ''
      pushBounded(lines.value, {
        time: nowIso(),
        level: 'node',
        message: `→ ${e.nodeKind} (${e.nodeId})${suffix}`,
        source: 'CTR',
      })
    }
  }

  function appendNodeDump(entries: NodeDumpEntry[]) {
    for (const e of entries) {
      const idx = lines.value.findIndex(
        (l) => l.nodeId === e.nodeId && l.lineKey === e.lineKey && !l.frozen,
      )
      if (idx >= 0) {
        const row = lines.value[idx]
        row.count = e.count
        row.message = e.line
        if (e.final) row.frozen = true
        continue
      }
      pushBounded(lines.value, {
        time: nowIso(),
        level: 'dump',
        message: e.line,
        source: 'CTR',
        nodeId: e.nodeId,
        lineKey: e.lineKey,
        count: e.count,
        frozen: e.final,
      })
    }
  }

  function clear() {
    lines.value = []
    lastSeq.value = 0
    dropDetected.value = false
  }

  return {
    lines,
    lastSeq,
    dropDetected,
    appendSystem,
    appendContainerLog,
    appendNodeEnter,
    appendNodeDump,
    clear,
  }
})
