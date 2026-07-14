import { defineStore } from 'pinia'
import { ref, shallowRef } from 'vue'
import type { BackendLogEntry } from '@/lib/backend'
import { parseLine, type LogLine } from '@/lib/logFormat'

const RING_CAP = 1000

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

export interface ActionTraceEntry {
  containerId?: string
  action: string
  source?: {
    containerId?: string
    nodeId?: string
    nodeKind?: string
    inPin?: string
    ContainerID?: string
    NodeID?: string
    NodeKind?: string
    InPin?: string
  }
  target?: { id?: string; ID?: string }
  backend?: string
  request?: unknown
  result?: unknown
  status?: string
  error?: string
  coordinateSteps?: unknown[]
  startedAt?: string
  endedAt?: string
  durationMs?: number
}

function nowIso() {
  return new Date().toISOString()
}

function traceSource(trace: ActionTraceEntry) {
  const source = trace.source ?? {}
  const nodeKind = source.nodeKind ?? source.NodeKind ?? '?'
  const nodeId = source.nodeId ?? source.NodeID ?? '?'
  const inPin = source.inPin ?? source.InPin ?? ''
  return `${nodeKind}(${nodeId})${inPin ? '.' + inPin : ''}`
}

function formatActionTrace(trace: ActionTraceEntry) {
  const status = trace.status || 'unknown'
  const duration = Number.isFinite(trace.durationMs) ? ` ${trace.durationMs}ms` : ''
  const target = trace.target?.id ?? trace.target?.ID ?? ''
  const targetPart = target ? ` @ ${target}` : ''
  const backend = trace.backend ? ` via ${trace.backend}` : ''
  const error = trace.error ? `: ${trace.error}` : ''
  return `${traceSource(trace)} -> ${trace.action} ${status}${duration}${targetPart}${backend}${error}`
}

export const useLogStore = defineStore('log', () => {
  // One shallow array replacement per backend batch avoids hundreds of deep
  // reactive push/splice notifications under noisy workflows.
  const lines = shallowRef<LogLine[]>([])
  const actionTraces = shallowRef<ActionTraceEntry[]>([])
  const lastSeq = ref(0)
  const dropDetected = ref(false)
  const received = ref(0)
  const dropped = ref(0)
  let nextID = 1

  function withID(line: LogLine): LogLine {
    return { ...line, id: nextID++ }
  }

  function commit(nextLines: LogLine[], nextTraces = actionTraces.value) {
    if (nextLines.length > RING_CAP) {
      const overflow = nextLines.length - RING_CAP
      dropped.value += overflow
      nextLines = nextLines.slice(overflow)
    }
    if (nextTraces.length > RING_CAP) nextTraces = nextTraces.slice(-RING_CAP)
    lines.value = nextLines
    actionTraces.value = nextTraces
  }

  function appendBatch(seq: number, entries: BackendLogEntry[], transportDropped = 0) {
    const next = lines.value.slice()
    const traces = actionTraces.value.slice()
    if (lastSeq.value !== 0 && seq !== lastSeq.value + 1) {
      dropDetected.value = true
      next.push(
        withID({
          time: nowIso(),
          level: 'warn',
          message: `[log:batch] sequence gap: expected ${lastSeq.value + 1}, got ${seq}`,
          source: 'SYS',
        }),
      )
    }
    lastSeq.value = seq
    if (transportDropped > 0) {
      dropDetected.value = true
      dropped.value += transportDropped
    }

    for (const entry of entries) {
      received.value++
      if (entry.kind === 'dump') {
        const idx = next.findIndex(
          (line) => line.nodeId === entry.nodeId && line.lineKey === entry.lineKey && !line.frozen,
        )
        if (idx >= 0) {
          next[idx] = {
            ...next[idx],
            count: entry.count ?? 1,
            message: entry.message,
            frozen: entry.final,
          }
          continue
        }
      }

      let message = entry.message
      if (entry.kind === 'action' && entry.trace && typeof entry.trace === 'object') {
        const trace = entry.trace as ActionTraceEntry
        traces.push(trace)
        message = formatActionTrace(trace)
      }
      next.push(
        withID({
          time: entry.time || nowIso(),
          level: entry.level || 'info',
          message,
          source: entry.source === 'CTR' ? 'CTR' : 'SYS',
          tag: entry.tag,
          fields: entry.fields,
          nodeId: entry.nodeId,
          lineKey: entry.lineKey,
          count: entry.count,
          frozen: entry.final,
        }),
      )
    }
    commit(next, traces)
  }

  // Local adapters keep tests and non-Wails callers on the same batch path.
  function appendSystem(seq: number, raw: string[]) {
    const entries = raw.map((line) => {
      const parsed = parseLine(line)
      return {
        time: parsed.time,
        level: parsed.level,
        source: 'SYS' as const,
        kind: 'system' as const,
        tag: parsed.tag,
        message: parsed.message,
      }
    })
    appendBatch(seq, entries)
  }

  function appendContainerLog(payload: ContainerLogPayload) {
    appendBatch(lastSeq.value + 1, [
      {
        time: nowIso(),
        level: payload.level || 'info',
        source: 'CTR',
        kind: 'log',
        message: payload.message,
      },
    ])
  }

  function appendNodeEnter(entries: NodeEnterEntry[]) {
    appendBatch(
      lastSeq.value + 1,
      entries.map((entry) => ({
        time: nowIso(),
        level: 'node',
        source: 'CTR' as const,
        kind: 'node' as const,
        message: `→ ${entry.nodeKind} (${entry.nodeId})${entry.count > 1 ? ` × ${entry.count}` : ''}`,
        count: entry.count,
      })),
    )
  }

  function appendNodeDump(entries: NodeDumpEntry[]) {
    appendBatch(
      lastSeq.value + 1,
      entries.map((entry) => ({
        time: nowIso(),
        level: 'dump',
        source: 'CTR' as const,
        kind: 'dump' as const,
        message: entry.line,
        nodeId: entry.nodeId,
        lineKey: entry.lineKey,
        count: entry.count,
        final: entry.final,
      })),
    )
  }

  function appendActionTrace(trace: ActionTraceEntry) {
    appendBatch(lastSeq.value + 1, [
      {
        time: nowIso(),
        level: 'action',
        source: 'CTR',
        kind: 'action',
        message: '',
        trace,
      },
    ])
  }

  function clear() {
    lines.value = []
    actionTraces.value = []
    lastSeq.value = 0
    dropDetected.value = false
    received.value = 0
    dropped.value = 0
  }

  return {
    lines,
    actionTraces,
    lastSeq,
    dropDetected,
    received,
    dropped,
    appendBatch,
    appendSystem,
    appendContainerLog,
    appendNodeEnter,
    appendNodeDump,
    appendActionTrace,
    clear,
  }
})
